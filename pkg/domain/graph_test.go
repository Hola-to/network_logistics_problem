package domain

import (
	"sync"
	"testing"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()

	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.Nodes == nil {
		t.Error("expected non-nil Nodes map")
	}
	if g.Edges == nil {
		t.Error("expected non-nil Edges map")
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(g.Nodes))
	}
}

func TestGraph_AddNode(t *testing.T) {
	g := NewGraph()

	node := &Node{
		ID:   1,
		X:    10.5,
		Y:    20.5,
		Type: NodeTypeWarehouse,
		Name: "Warehouse A",
	}

	g.AddNode(node)

	if len(g.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(g.Nodes))
	}

	got, ok := g.GetNode(1)
	if !ok {
		t.Fatal("expected to find node")
	}
	if got.Name != "Warehouse A" {
		t.Errorf("expected name 'Warehouse A', got %s", got.Name)
	}
}

func TestGraph_AddEdge(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})

	edge := &Edge{
		From:     1,
		To:       2,
		Capacity: 100,
		Cost:     10,
		Length:   50,
	}

	g.AddEdge(edge)

	if len(g.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(g.Edges))
	}

	got, ok := g.GetEdge(1, 2)
	if !ok {
		t.Fatal("expected to find edge")
	}
	if got.Capacity != 100 {
		t.Errorf("expected capacity 100, got %f", got.Capacity)
	}

	// Check indices
	outgoing := g.GetOutgoing(1)
	if len(outgoing) != 1 || outgoing[0] != 2 {
		t.Error("expected outgoing neighbor 2")
	}

	incoming := g.GetIncoming(2)
	if len(incoming) != 1 || incoming[0] != 1 {
		t.Error("expected incoming neighbor 1")
	}
}

func TestGraph_Clone(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 3
	g.Name = "Test Graph"
	g.Metadata["key"] = "value"

	g.AddNode(&Node{ID: 1, Name: "Source"})
	g.AddNode(&Node{ID: 2, Name: "Middle"})
	g.AddNode(&Node{ID: 3, Name: "Sink"})
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10})
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10})

	clone := g.Clone()

	// Check basic properties
	if clone.SourceID != g.SourceID {
		t.Error("SourceID not cloned")
	}
	if clone.SinkID != g.SinkID {
		t.Error("SinkID not cloned")
	}
	if clone.Name != g.Name {
		t.Error("Name not cloned")
	}
	if clone.Metadata["key"] != "value" {
		t.Error("Metadata not cloned")
	}

	// Check nodes
	if len(clone.Nodes) != len(g.Nodes) {
		t.Error("Nodes count mismatch")
	}

	// Check edges
	if len(clone.Edges) != len(g.Edges) {
		t.Error("Edges count mismatch")
	}

	// Modify original, clone should not change
	g.Nodes[1].Name = "Modified"
	if clone.Nodes[1].Name == "Modified" {
		t.Error("Clone should be independent")
	}
}

func TestEdge_Utilization(t *testing.T) {
	tests := []struct {
		capacity    float64
		currentFlow float64
		expected    float64
	}{
		{100, 50, 0.5},
		{100, 100, 1.0},
		{100, 0, 0.0},
		{0, 0, 0.0}, // Zero capacity
	}

	for _, tt := range tests {
		edge := &Edge{Capacity: tt.capacity, CurrentFlow: tt.currentFlow}
		if got := edge.Utilization(); got != tt.expected {
			t.Errorf("Utilization() = %f, want %f", got, tt.expected)
		}
	}
}

func TestEdge_IsSaturated(t *testing.T) {
	tests := []struct {
		capacity    float64
		currentFlow float64
		expected    bool
	}{
		{100, 99.999999999, true},
		{100, 100, true},
		{100, 99, false},
		{100, 0, false},
	}

	for _, tt := range tests {
		edge := &Edge{Capacity: tt.capacity, CurrentFlow: tt.currentFlow}
		if got := edge.IsSaturated(); got != tt.expected {
			t.Errorf("IsSaturated() with %f/%f = %v, want %v",
				tt.currentFlow, tt.capacity, got, tt.expected)
		}
	}
}

func TestEdge_HasFlow(t *testing.T) {
	tests := []struct {
		flow     float64
		expected bool
	}{
		{0, false},
		{0.0000000001, false}, // Below epsilon
		{0.000001, true},
		{1, true},
	}

	for _, tt := range tests {
		edge := &Edge{CurrentFlow: tt.flow}
		if got := edge.HasFlow(); got != tt.expected {
			t.Errorf("HasFlow() with %f = %v, want %v", tt.flow, got, tt.expected)
		}
	}
}

func TestEdge_ResidualCapacity(t *testing.T) {
	edge := &Edge{Capacity: 100, CurrentFlow: 30}
	if got := edge.ResidualCapacity(); got != 70 {
		t.Errorf("ResidualCapacity() = %f, want 70", got)
	}
}

func TestGraph_GetNodesByType(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: 1, Type: NodeTypeWarehouse})
	g.AddNode(&Node{ID: 2, Type: NodeTypeWarehouse})
	g.AddNode(&Node{ID: 3, Type: NodeTypeDeliveryPoint})
	g.AddNode(&Node{ID: 4, Type: NodeTypeIntersection})

	warehouses := g.GetNodesByType(NodeTypeWarehouse)
	if len(warehouses) != 2 {
		t.Errorf("expected 2 warehouses, got %d", len(warehouses))
	}

	deliveryPoints := g.GetNodesByType(NodeTypeDeliveryPoint)
	if len(deliveryPoints) != 1 {
		t.Errorf("expected 1 delivery point, got %d", len(deliveryPoints))
	}
}

func TestGraph_GetActiveEdges(t *testing.T) {
	g := NewGraph()
	g.AddEdge(&Edge{From: 1, To: 2, CurrentFlow: 10})
	g.AddEdge(&Edge{From: 2, To: 3, CurrentFlow: 0})
	g.AddEdge(&Edge{From: 3, To: 4, CurrentFlow: 5})

	active := g.GetActiveEdges()
	if len(active) != 2 {
		t.Errorf("expected 2 active edges, got %d", len(active))
	}
}

func TestGraph_GetSaturatedEdges(t *testing.T) {
	g := NewGraph()
	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, CurrentFlow: 10})
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10, CurrentFlow: 5})
	g.AddEdge(&Edge{From: 3, To: 4, Capacity: 10, CurrentFlow: 9.9999999999})

	saturated := g.GetSaturatedEdges()
	if len(saturated) != 2 {
		t.Errorf("expected 2 saturated edges, got %d", len(saturated))
	}
}

func TestGraph_ResetFlow(t *testing.T) {
	g := NewGraph()
	g.AddEdge(&Edge{From: 1, To: 2, CurrentFlow: 10})
	g.AddEdge(&Edge{From: 2, To: 3, CurrentFlow: 5})

	g.ResetFlow()

	for _, edge := range g.Edges {
		if edge.CurrentFlow != 0 {
			t.Errorf("expected flow to be reset, got %f", edge.CurrentFlow)
		}
	}
}

func TestGraph_TotalFlow(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddEdge(&Edge{From: 1, To: 2, CurrentFlow: 10})
	g.AddEdge(&Edge{From: 1, To: 3, CurrentFlow: 5})

	total := g.TotalFlow()
	if total != 15 {
		t.Errorf("expected total flow 15, got %f", total)
	}
}

func TestGraph_TotalCost(t *testing.T) {
	g := NewGraph()
	g.SourceID = 0
	g.SinkID = 10
	g.AddEdge(&Edge{From: 1, To: 2, CurrentFlow: 10, Cost: 5})
	g.AddEdge(&Edge{From: 2, To: 3, CurrentFlow: 5, Cost: 3})

	total := g.TotalCost()
	expected := 10*5.0 + 5*3.0 // 65
	if total != expected {
		t.Errorf("expected total cost %f, got %f", expected, total)
	}
}

func TestGraph_Validate(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Graph)
		wantError bool
	}{
		{
			name: "valid graph",
			setup: func(g *Graph) {
				g.SourceID = 1
				g.SinkID = 3
				g.AddNode(&Node{ID: 1})
				g.AddNode(&Node{ID: 2})
				g.AddNode(&Node{ID: 3})
				g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10})
				g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10})
			},
			wantError: false,
		},
		{
			name: "missing source",
			setup: func(g *Graph) {
				g.SourceID = 999
				g.SinkID = 1
				g.AddNode(&Node{ID: 1})
			},
			wantError: true,
		},
		{
			name: "missing sink",
			setup: func(g *Graph) {
				g.SourceID = 1
				g.SinkID = 999
				g.AddNode(&Node{ID: 1})
			},
			wantError: true,
		},
		{
			name: "source equals sink",
			setup: func(g *Graph) {
				g.SourceID = 1
				g.SinkID = 1
				g.AddNode(&Node{ID: 1})
			},
			wantError: true,
		},
		{
			name: "dangling edge",
			setup: func(g *Graph) {
				g.SourceID = 1
				g.SinkID = 2
				g.AddNode(&Node{ID: 1})
				g.AddNode(&Node{ID: 2})
				g.AddEdge(&Edge{From: 1, To: 999})
			},
			wantError: true,
		},
		{
			name: "self loop",
			setup: func(g *Graph) {
				g.SourceID = 1
				g.SinkID = 2
				g.AddNode(&Node{ID: 1})
				g.AddNode(&Node{ID: 2})
				g.AddEdge(&Edge{From: 1, To: 1})
			},
			wantError: true,
		},
		{
			name: "negative capacity",
			setup: func(g *Graph) {
				g.SourceID = 1
				g.SinkID = 2
				g.AddNode(&Node{ID: 1})
				g.AddNode(&Node{ID: 2})
				g.AddEdge(&Edge{From: 1, To: 2, Capacity: -10})
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGraph()
			tt.setup(g)
			errs := g.Validate()
			if (len(errs) > 0) != tt.wantError {
				t.Errorf("Validate() errors = %v, wantError %v", errs, tt.wantError)
			}
		})
	}
}

func TestGraph_Concurrent(t *testing.T) {
	g := NewGraph()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			g.AddNode(&Node{ID: id})
		}(int64(i))
	}
	wg.Wait()

	if g.NodeCount() != 100 {
		t.Errorf("expected 100 nodes, got %d", g.NodeCount())
	}
}

func TestNodeType_String(t *testing.T) {
	tests := []struct {
		nodeType NodeType
		expected string
	}{
		{NodeTypeWarehouse, "warehouse"},
		{NodeTypeDeliveryPoint, "delivery_point"},
		{NodeTypeIntersection, "intersection"},
		{NodeTypeSource, "source"},
		{NodeTypeSink, "sink"},
		{NodeTypeUnspecified, "unspecified"},
	}

	for _, tt := range tests {
		if got := tt.nodeType.String(); got != tt.expected {
			t.Errorf("%v.String() = %s, want %s", tt.nodeType, got, tt.expected)
		}
	}
}

func TestRoadType_String(t *testing.T) {
	tests := []struct {
		roadType RoadType
		expected string
	}{
		{RoadTypeHighway, "highway"},
		{RoadTypePrimary, "primary"},
		{RoadTypeSecondary, "secondary"},
		{RoadTypeLocal, "local"},
		{RoadTypeUrban, "urban"},
		{RoadTypeUnspecified, "unspecified"},
	}

	for _, tt := range tests {
		if got := tt.roadType.String(); got != tt.expected {
			t.Errorf("%v.String() = %s, want %s", tt.roadType, got, tt.expected)
		}
	}
}

func TestEdgeKey_String(t *testing.T) {
	key := EdgeKey{From: 1, To: 2}
	expected := "1->2"
	if got := key.String(); got != expected {
		t.Errorf("EdgeKey.String() = %s, want %s", got, expected)
	}
}

func TestNode_Clone(t *testing.T) {
	node := &Node{
		ID:       1,
		X:        10.5,
		Y:        20.5,
		Type:     NodeTypeWarehouse,
		Name:     "Test",
		Metadata: map[string]string{"key": "value"},
		Supply:   100,
		Demand:   50,
	}

	clone := node.Clone()

	if clone.ID != node.ID {
		t.Error("ID not cloned")
	}
	if clone.Metadata["key"] != "value" {
		t.Error("Metadata not cloned")
	}

	// Modify original
	node.Metadata["key"] = "modified"
	if clone.Metadata["key"] == "modified" {
		t.Error("Clone should be independent")
	}
}

func TestEdge_Clone(t *testing.T) {
	edge := &Edge{
		From:        1,
		To:          2,
		Capacity:    100,
		Cost:        10,
		CurrentFlow: 50,
	}

	clone := edge.Clone()

	if clone.From != edge.From {
		t.Error("From not cloned")
	}
	if clone.Capacity != edge.Capacity {
		t.Error("Capacity not cloned")
	}

	// Modify original
	edge.CurrentFlow = 75
	if clone.CurrentFlow == 75 {
		t.Error("Clone should be independent")
	}
}
