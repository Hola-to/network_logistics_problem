// Package converter provides utilities for converting between protobuf graph
// representations and the internal residual graph structures used by flow algorithms.
//
// The package handles:
// - Converting proto Graph messages to ResidualGraph for algorithm execution
// - Converting algorithm results back to proto format for API responses
// - Computing graph statistics and metrics
// - Filtering and selecting edges based on various criteria
//
// Thread Safety:
// All functions in this package are stateless and thread-safe. The returned
// slices and maps are newly allocated and safe to modify.
//
// Determinism:
// All functions that iterate over graph structures use sorted node orderings
// to ensure deterministic output regardless of map iteration order.
package converter

import (
	"sort"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"
)

// PathWithFlow represents an augmenting path with its associated flow value.
// Used to track individual paths found during flow algorithm execution.
type PathWithFlow struct {
	// NodeIDs contains the sequence of node IDs from source to sink.
	// The path is valid if len(NodeIDs) >= 2 (at least source and sink).
	NodeIDs []int64

	// Flow is the amount of flow pushed along this path.
	// Always positive for valid paths.
	Flow float64
}

// =============================================================================
// Proto to Internal Conversion
// =============================================================================

// ToResidualGraph converts a protobuf Graph message to an internal ResidualGraph
// structure suitable for flow algorithm execution.
//
// The conversion process:
// 1. Creates a new ResidualGraph
// 2. Adds all nodes from the proto graph
// 3. Adds edges with their reverse edges for residual capacity tracking
// 4. Handles bidirectional edges by adding both directions
//
// Parameters:
// - protoGraph: The protobuf Graph message to convert
//
// Returns:
// - A new ResidualGraph ready for algorithm execution
//
// Example:
//
//	rg := ToResidualGraph(request.Graph)
//	result := algorithms.Dinic(rg, sourceID, sinkID, nil)
func ToResidualGraph(protoGraph *commonv1.Graph) *graph.ResidualGraph {
	rg := graph.NewResidualGraph()

	// Add all nodes first
	for _, node := range protoGraph.Nodes {
		rg.AddNode(node.Id)
	}

	// Add edges with reverse edges for residual graph structure
	for _, edge := range protoGraph.Edges {
		rg.AddEdgeWithReverse(edge.From, edge.To, edge.Capacity, edge.Cost)

		// For bidirectional edges, add the reverse direction as well
		if edge.Bidirectional {
			rg.AddEdgeWithReverse(edge.To, edge.From, edge.Capacity, edge.Cost)
		}
	}

	return rg
}

// =============================================================================
// Flow Calculation Helpers
// =============================================================================

// GetNetFlow calculates the net flow on a forward edge.
//
// In a residual graph, the net flow is the difference between the original
// capacity and the current remaining capacity:
//
//	NetFlow = OriginalCapacity - RemainingCapacity
//
// This correctly accounts for flow that was "cancelled" by pushing flow
// through reverse edges. For example:
//   - Push 5 units through edge A→B: capacity becomes 0, net flow = 5
//   - Push 1 unit through reverse edge B→A: capacity becomes 1, net flow = 4
//
// Parameters:
// - edge: The residual edge to calculate flow for
//
// Returns:
// - The net flow on the edge (always >= 0)
func GetNetFlow(edge *graph.ResidualEdge) float64 {
	if edge == nil || edge.IsReverse {
		return 0
	}

	netFlow := edge.OriginalCapacity - edge.Capacity
	if netFlow < 0 {
		netFlow = 0
	}

	return netFlow
}

// GetUtilization calculates the utilization ratio for an edge.
//
// Utilization = NetFlow / OriginalCapacity
//
// Returns a value in the range [0.0, 1.0].
func GetUtilization(edge *graph.ResidualEdge) float64 {
	if edge == nil || edge.IsReverse || edge.OriginalCapacity <= graph.Epsilon {
		return 0
	}

	netFlow := GetNetFlow(edge)
	utilization := netFlow / edge.OriginalCapacity

	// Clamp to valid range (shouldn't be needed, but safety first)
	if utilization > 1.0 {
		utilization = 1.0
	}
	if utilization < 0.0 {
		utilization = 0.0
	}

	return utilization
}

// =============================================================================
// Path Conversion
// =============================================================================

// ToPaths converts internal PathWithFlow slices to protobuf Path messages.
// Each path includes the flow amount and computed cost.
//
// The cost for each path is computed as: sum of edge costs * path flow.
//
// Parameters:
// - paths: Slice of internal path representations
// - rg: The residual graph (used to look up edge costs)
//
// Returns:
// - Slice of protobuf Path messages
//
// Note: Paths with fewer than 2 nodes are filtered out as invalid.
func ToPaths(paths []PathWithFlow, rg *graph.ResidualGraph) []*commonv1.Path {
	result := make([]*commonv1.Path, 0, len(paths))

	for _, p := range paths {
		// Valid paths must have at least source and sink
		if len(p.NodeIDs) < 2 {
			continue
		}

		// Compute total edge cost for the path
		unitCost := calculatePathCost(rg, p.NodeIDs)

		result = append(result, &commonv1.Path{
			NodeIds: p.NodeIDs,
			Flow:    p.Flow,
			Cost:    unitCost * p.Flow, // Total cost = unit cost * flow
		})
	}

	return result
}

// ToPathsFromNodeIDs converts raw node ID sequences to protobuf Paths.
// Flow is computed as the minimum residual capacity along each path.
//
// This is useful when you have path sequences but haven't tracked flow values.
//
// Parameters:
// - paths: Slice of node ID sequences
// - rg: The residual graph
//
// Returns:
// - Slice of protobuf Path messages with computed flows
func ToPathsFromNodeIDs(paths [][]int64, rg *graph.ResidualGraph) []*commonv1.Path {
	result := make([]*commonv1.Path, 0, len(paths))

	for _, nodeIDs := range paths {
		if len(nodeIDs) < 2 {
			continue
		}

		// Flow is the bottleneck capacity
		flow := graph.FindMinCapacityOnPath(rg, nodeIDs)
		unitCost := calculatePathCost(rg, nodeIDs)

		result = append(result, &commonv1.Path{
			NodeIds: nodeIDs,
			Flow:    flow,
			Cost:    unitCost * flow,
		})
	}

	return result
}

// ToPathsWithFlowReconstruction reconstructs path flows from edge flow values.
// The flow for each path is the minimum net flow along the path.
//
// This is useful when paths were recorded during execution but flow values
// need to be recomputed from the final graph state.
//
// Parameters:
// - paths: Slice of node ID sequences
// - rg: The residual graph with flow values
//
// Returns:
// - Slice of protobuf Path messages
func ToPathsWithFlowReconstruction(paths [][]int64, rg *graph.ResidualGraph) []*commonv1.Path {
	result := make([]*commonv1.Path, 0, len(paths))

	for _, nodeIDs := range paths {
		if len(nodeIDs) < 2 {
			continue
		}

		// Find minimum net flow along the path
		minFlow := graph.Infinity
		for i := 0; i < len(nodeIDs)-1; i++ {
			edge := rg.GetEdge(nodeIDs[i], nodeIDs[i+1])
			if edge == nil {
				minFlow = 0
				break
			}

			// Use net flow calculation
			netFlow := GetNetFlow(edge)
			if netFlow < minFlow {
				minFlow = netFlow
			}
		}

		// Handle edge cases
		if minFlow >= graph.Infinity || minFlow < 0 {
			minFlow = 0
		}

		unitCost := calculatePathCost(rg, nodeIDs)

		result = append(result, &commonv1.Path{
			NodeIds: nodeIDs,
			Flow:    minFlow,
			Cost:    unitCost * minFlow,
		})
	}

	return result
}

// calculatePathCost computes the sum of edge costs along a path.
// This is the cost per unit of flow, not the total cost.
func calculatePathCost(rg *graph.ResidualGraph, nodeIDs []int64) float64 {
	var totalCost float64

	for i := 0; i < len(nodeIDs)-1; i++ {
		edge := rg.GetEdge(nodeIDs[i], nodeIDs[i+1])
		if edge != nil {
			totalCost += edge.Cost
		}
	}

	return totalCost
}

// =============================================================================
// Edge Conversion
// =============================================================================

// FlowEdgeOptions configures which edges to include in conversion output.
type FlowEdgeOptions struct {
	// IncludeZeroFlow includes edges with no flow when true.
	// Default: false (only edges with positive flow are included)
	IncludeZeroFlow bool

	// IncludeReverseEdge includes reverse/residual edges when true.
	// Default: false (only forward edges are included)
	IncludeReverseEdge bool

	// MinFlowThreshold is the minimum flow value to include an edge.
	// Edges with flow < MinFlowThreshold are excluded (unless IncludeZeroFlow).
	// Default: graph.Epsilon
	MinFlowThreshold float64
}

// DefaultFlowEdgeOptions returns the default options for edge conversion.
// By default, only forward edges with positive flow are included.
func DefaultFlowEdgeOptions() *FlowEdgeOptions {
	return &FlowEdgeOptions{
		IncludeZeroFlow:    false,
		IncludeReverseEdge: false,
		MinFlowThreshold:   graph.Epsilon,
	}
}

// ToFlowEdges converts residual graph edges to protobuf FlowEdge messages.
// Uses default options: only forward edges with positive flow.
//
// The edges are returned in deterministic order (sorted by from node, then to node).
//
// Parameters:
// - rg: The residual graph with flow values
//
// Returns:
// - Slice of protobuf FlowEdge messages
func ToFlowEdges(rg *graph.ResidualGraph) []*commonv1.FlowEdge {
	return ToFlowEdgesWithOptions(rg, DefaultFlowEdgeOptions())
}

// ToFlowEdgesWithOptions converts edges with custom filtering options.
//
// For forward edges, the actual flow is calculated as:
//
//	NetFlow = OriginalCapacity - CurrentResidualCapacity
//
// This correctly handles cases where flow was "cancelled" by using reverse edges.
//
// Parameters:
// - rg: The residual graph
// - opts: Options controlling which edges to include
//
// Returns:
// - Filtered slice of protobuf FlowEdge messages in deterministic order
func ToFlowEdgesWithOptions(rg *graph.ResidualGraph, opts *FlowEdgeOptions) []*commonv1.FlowEdge {
	if opts == nil {
		opts = DefaultFlowEdgeOptions()
	}

	var result []*commonv1.FlowEdge

	// Iterate in deterministic order using sorted node list
	nodes := rg.GetSortedNodes()
	for _, from := range nodes {
		// Use EdgesList which maintains insertion order
		edges := rg.GetNeighborsList(from)
		for _, edge := range edges {
			// Skip reverse edges unless explicitly requested
			if edge.IsReverse && !opts.IncludeReverseEdge {
				continue
			}

			// Calculate net flow for forward edges
			// NetFlow = OriginalCapacity - RemainingCapacity
			var netFlow float64
			if edge.IsReverse {
				// For reverse edges, flow represents capacity available to "cancel"
				netFlow = edge.Capacity
			} else {
				netFlow = GetNetFlow(edge)
			}

			// Apply flow threshold filter
			if netFlow < opts.MinFlowThreshold && !opts.IncludeZeroFlow {
				continue
			}

			// Compute utilization ratio based on net flow
			utilization := 0.0
			if !edge.IsReverse && edge.OriginalCapacity > graph.Epsilon {
				utilization = netFlow / edge.OriginalCapacity
				// Clamp to valid range [0, 1]
				if utilization > 1.0 {
					utilization = 1.0
				}
				if utilization < 0.0 {
					utilization = 0.0
				}
			}

			result = append(result, &commonv1.FlowEdge{
				From:        from,
				To:          edge.To,
				Flow:        netFlow,
				Capacity:    edge.OriginalCapacity,
				Cost:        edge.Cost,
				Utilization: utilization,
			})
		}
	}

	return result
}

// ToAllEdges returns all forward edges regardless of flow.
// Useful for visualizing the complete network structure.
func ToAllEdges(rg *graph.ResidualGraph) []*commonv1.FlowEdge {
	opts := &FlowEdgeOptions{
		IncludeZeroFlow:    true,
		IncludeReverseEdge: false,
		MinFlowThreshold:   0,
	}
	return ToFlowEdgesWithOptions(rg, opts)
}

// ToDebugEdges returns all edges including reverse edges.
// Useful for debugging residual graph structure.
func ToDebugEdges(rg *graph.ResidualGraph) []*commonv1.FlowEdge {
	opts := &FlowEdgeOptions{
		IncludeZeroFlow:    true,
		IncludeReverseEdge: true,
		MinFlowThreshold:   0,
	}
	return ToFlowEdgesWithOptions(rg, opts)
}

// =============================================================================
// Edge Filters
// =============================================================================

// EdgeFilter is a function type for custom edge filtering.
// Returns true if the edge should be included in the output.
type EdgeFilter func(from int64, edge *graph.ResidualEdge) bool

// ToFlowEdgesFiltered converts edges using a custom filter function.
// Edges are returned in deterministic order.
//
// Parameters:
// - rg: The residual graph
// - filter: Function to determine if each edge should be included
//
// Returns:
// - Filtered slice of protobuf FlowEdge messages
func ToFlowEdgesFiltered(rg *graph.ResidualGraph, filter EdgeFilter) []*commonv1.FlowEdge {
	var result []*commonv1.FlowEdge

	nodes := rg.GetSortedNodes()
	for _, from := range nodes {
		edges := rg.GetNeighborsList(from)
		for _, edge := range edges {
			if filter != nil && !filter(from, edge) {
				continue
			}

			// Calculate net flow
			netFlow := GetNetFlow(edge)
			utilization := GetUtilization(edge)

			result = append(result, &commonv1.FlowEdge{
				From:        from,
				To:          edge.To,
				Flow:        netFlow,
				Capacity:    edge.OriginalCapacity,
				Cost:        edge.Cost,
				Utilization: utilization,
			})
		}
	}

	return result
}

// FilterActiveEdges returns a filter that selects only forward edges with positive net flow.
func FilterActiveEdges() EdgeFilter {
	return func(from int64, edge *graph.ResidualEdge) bool {
		if edge.IsReverse {
			return false
		}
		netFlow := GetNetFlow(edge)
		return netFlow > graph.Epsilon
	}
}

// FilterHighUtilization returns a filter that selects edges with utilization >= threshold.
//
// Parameters:
// - threshold: Minimum utilization ratio (0.0 to 1.0)
//
// Example:
//
//	// Get edges that are at least 80% utilized
//	filter := FilterHighUtilization(0.8)
//	edges := ToFlowEdgesFiltered(rg, filter)
func FilterHighUtilization(threshold float64) EdgeFilter {
	return func(from int64, edge *graph.ResidualEdge) bool {
		if edge.IsReverse {
			return false
		}
		utilization := GetUtilization(edge)
		return utilization >= threshold
	}
}

// FilterSaturatedEdges returns a filter that selects fully saturated edges
// (edges where flow equals capacity within epsilon tolerance).
func FilterSaturatedEdges() EdgeFilter {
	return FilterHighUtilization(1.0 - graph.Epsilon)
}

// FilterByNodes returns a filter that selects edges between specified nodes.
//
// Parameters:
// - nodes: Map of node IDs to include (both endpoints must be in the map)
func FilterByNodes(nodes map[int64]bool) EdgeFilter {
	return func(from int64, edge *graph.ResidualEdge) bool {
		if edge.IsReverse {
			return false
		}
		return nodes[from] && nodes[edge.To]
	}
}

// =============================================================================
// Graph Update
// =============================================================================

// UpdateGraphWithFlow creates a copy of a proto Graph with flow values
// populated from algorithm results.
//
// This is useful for returning the modified graph state to clients.
//
// The flow on each edge is calculated as:
//
//	NetFlow = OriginalCapacity - RemainingCapacity
//
// This correctly handles flow cancellation via reverse edges.
//
// Parameters:
// - protoGraph: Original protobuf graph
// - rg: Residual graph with computed flow values
//
// Returns:
// - New protobuf Graph with CurrentFlow fields populated
func UpdateGraphWithFlow(protoGraph *commonv1.Graph, rg *graph.ResidualGraph) *commonv1.Graph {
	result := &commonv1.Graph{
		Nodes:    protoGraph.Nodes,
		Edges:    make([]*commonv1.Edge, len(protoGraph.Edges)),
		SourceId: protoGraph.SourceId,
		SinkId:   protoGraph.SinkId,
		Name:     protoGraph.Name,
		Metadata: protoGraph.Metadata,
	}

	for i, edge := range protoGraph.Edges {
		newEdge := &commonv1.Edge{
			From:          edge.From,
			To:            edge.To,
			Capacity:      edge.Capacity,
			Cost:          edge.Cost,
			Length:        edge.Length,
			RoadType:      edge.RoadType,
			Bidirectional: edge.Bidirectional,
		}

		// Populate net flow from residual graph
		if re := rg.GetEdge(edge.From, edge.To); re != nil {
			newEdge.CurrentFlow = GetNetFlow(re)
		}

		result.Edges[i] = newEdge
	}

	return result
}

// =============================================================================
// Statistics
// =============================================================================

// CalculateGraphStatistics computes various metrics about the graph structure.
//
// Computed metrics include:
// - Node and edge counts
// - Warehouse and delivery point counts (by node type)
// - Total and average capacity/length
// - Graph density
//
// Parameters:
// - protoGraph: The protobuf graph to analyze
//
// Returns:
// - GraphStatistics message with computed metrics
func CalculateGraphStatistics(protoGraph *commonv1.Graph) *commonv1.GraphStatistics {
	var warehouseCount, deliveryPointCount int64
	var totalCapacity, totalLength float64

	// Count node types
	for _, node := range protoGraph.Nodes {
		switch node.Type {
		case commonv1.NodeType_NODE_TYPE_WAREHOUSE:
			warehouseCount++
		case commonv1.NodeType_NODE_TYPE_DELIVERY_POINT:
			deliveryPointCount++
		}
	}

	// Sum edge metrics
	for _, edge := range protoGraph.Edges {
		totalCapacity += edge.Capacity
		totalLength += edge.Length
	}

	nodeCount := int64(len(protoGraph.Nodes))
	edgeCount := int64(len(protoGraph.Edges))

	// Compute averages
	avgLength := 0.0
	if edgeCount > 0 {
		avgLength = totalLength / float64(edgeCount)
	}

	// Compute density: actual edges / max possible edges
	// For directed graph: max edges = n * (n-1)
	density := 0.0
	if nodeCount > 1 {
		maxEdges := nodeCount * (nodeCount - 1)
		density = float64(edgeCount) / float64(maxEdges)
	}

	return &commonv1.GraphStatistics{
		NodeCount:          nodeCount,
		EdgeCount:          edgeCount,
		WarehouseCount:     warehouseCount,
		DeliveryPointCount: deliveryPointCount,
		TotalCapacity:      totalCapacity,
		AverageEdgeLength:  avgLength,
		IsConnected:        true, // TODO: implement actual connectivity check
		Density:            density,
	}
}

// =============================================================================
// Flow Statistics
// =============================================================================

// FlowStatistics contains computed statistics about the flow solution.
type FlowStatistics struct {
	// TotalFlow is the total flow from source to sink
	TotalFlow float64

	// TotalCost is the sum of flow * cost for all edges
	TotalCost float64

	// SaturatedEdges is the count of edges at full capacity
	SaturatedEdges int

	// ActiveEdges is the count of edges with positive flow
	ActiveEdges int

	// AverageUtilization is the mean utilization across all edges with capacity
	AverageUtilization float64
}

// CalculateFlowStatistics computes statistics about the current flow solution.
//
// Parameters:
// - rg: The residual graph with computed flow
// - source: The source node ID
//
// Returns:
// - FlowStatistics with computed metrics
func CalculateFlowStatistics(rg *graph.ResidualGraph, source int64) *FlowStatistics {
	stats := &FlowStatistics{}

	// Calculate total flow from source
	stats.TotalFlow = rg.GetTotalFlow(source)

	var totalUtilization float64
	var edgesWithCapacity int

	nodes := rg.GetSortedNodes()
	for _, from := range nodes {
		edges := rg.GetNeighborsList(from)
		for _, edge := range edges {
			if edge.IsReverse {
				continue
			}

			netFlow := GetNetFlow(edge)
			utilization := GetUtilization(edge)

			// Accumulate cost
			stats.TotalCost += netFlow * edge.Cost

			// Count active edges
			if netFlow > graph.Epsilon {
				stats.ActiveEdges++
			}

			// Count saturated edges
			if utilization >= 1.0-graph.Epsilon {
				stats.SaturatedEdges++
			}

			// Accumulate for average utilization
			if edge.OriginalCapacity > graph.Epsilon {
				totalUtilization += utilization
				edgesWithCapacity++
			}
		}
	}

	// Calculate average utilization
	if edgesWithCapacity > 0 {
		stats.AverageUtilization = totalUtilization / float64(edgesWithCapacity)
	}

	return stats
}

// =============================================================================
// Utility Functions
// =============================================================================

// GetSortedNodeIDs extracts and sorts node IDs from a boolean map.
// Useful for deterministic iteration over node sets.
func GetSortedNodeIDs(nodes map[int64]bool) []int64 {
	result := make([]int64, 0, len(nodes))
	for id := range nodes {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
