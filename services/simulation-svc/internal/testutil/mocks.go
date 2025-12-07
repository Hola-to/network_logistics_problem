// services/simulation-svc/internal/testutil/mocks.go
package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/client"
	"logistics/services/simulation-svc/internal/repository"
)

// ================== Mock Repository ==================

type MockSimulationRepository struct {
	mu          sync.RWMutex
	simulations map[string]*repository.Simulation

	// For controlling behavior
	CreateErr    error
	GetByIDErr   error
	DeleteErr    error
	ListErr      error
	GetByUserErr error

	// Call tracking
	CreateCalls      int
	GetByIDCalls     int
	DeleteCalls      int
	ListCalls        int
	ListByUserCalls  int
	GetByUserIDCalls int
}

func NewMockSimulationRepository() *MockSimulationRepository {
	return &MockSimulationRepository{
		simulations: make(map[string]*repository.Simulation),
	}
}

func (m *MockSimulationRepository) Create(ctx context.Context, sim *repository.Simulation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateCalls++

	if m.CreateErr != nil {
		return m.CreateErr
	}

	sim.ID = generateID()
	sim.CreatedAt = time.Now()
	sim.UpdatedAt = time.Now()
	m.simulations[sim.ID] = sim
	return nil
}

func (m *MockSimulationRepository) GetByID(ctx context.Context, id string) (*repository.Simulation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByIDCalls++

	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	sim, ok := m.simulations[id]
	if !ok {
		return nil, repository.ErrSimulationNotFound
	}
	return sim, nil
}

func (m *MockSimulationRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteCalls++

	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	if _, ok := m.simulations[id]; !ok {
		return repository.ErrSimulationNotFound
	}
	delete(m.simulations, id)
	return nil
}

func (m *MockSimulationRepository) List(
	ctx context.Context,
	userID string,
	simType string,
	opts *repository.ListOptions,
) ([]*repository.SimulationSummary, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.ListCalls++

	if m.ListErr != nil {
		return nil, 0, m.ListErr
	}

	var results []*repository.SimulationSummary
	for _, sim := range m.simulations {
		if sim.UserID != userID {
			continue
		}
		if simType != "" && sim.SimulationType != simType {
			continue
		}
		results = append(results, &repository.SimulationSummary{
			ID:             sim.ID,
			Name:           sim.Name,
			SimulationType: sim.SimulationType,
			CreatedAt:      sim.CreatedAt,
			Tags:           sim.Tags,
		})
	}

	// Apply pagination
	total := int64(len(results))
	if opts != nil {
		start := opts.Offset
		end := opts.Offset + opts.Limit
		if start > len(results) {
			start = len(results)
		}
		if end > len(results) {
			end = len(results)
		}
		results = results[start:end]
	}

	return results, total, nil
}

func (m *MockSimulationRepository) ListByUser(
	ctx context.Context,
	userID string,
	opts *repository.ListOptions,
) ([]*repository.SimulationSummary, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.ListByUserCalls++

	return m.List(ctx, userID, "", opts)
}

func (m *MockSimulationRepository) GetByUserAndID(
	ctx context.Context,
	userID, id string,
) (*repository.Simulation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetByUserIDCalls++

	if m.GetByUserErr != nil {
		return nil, m.GetByUserErr
	}

	sim, ok := m.simulations[id]
	if !ok {
		return nil, repository.ErrSimulationNotFound
	}
	if sim.UserID != userID {
		return nil, repository.ErrAccessDenied
	}
	return sim, nil
}

// AddSimulation добавляет симуляцию напрямую (для тестов)
func (m *MockSimulationRepository) AddSimulation(sim *repository.Simulation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sim.ID == "" {
		sim.ID = generateID()
	}
	m.simulations[sim.ID] = sim
}

var idCounter int
var idMu sync.Mutex

func generateID() string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return "sim_" + string(rune('a'+idCounter-1))
}

// ================== Mock Solver Client ==================

type MockSolverClient struct {
	mu sync.RWMutex

	// Default response
	DefaultResult *client.SolveResult
	DefaultError  error

	// Per-call responses (for sequences)
	Results []*client.SolveResult
	Errors  []error
	callIdx int

	// Call tracking
	SolveCalls int
}

func NewMockSolverClient() *MockSolverClient {
	return &MockSolverClient{
		DefaultResult: &client.SolveResult{
			MaxFlow:            100.0,
			TotalCost:          50.0,
			AverageUtilization: 0.75,
			SaturatedEdges:     2,
			ActivePaths:        3,
			Status:             commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
			Graph:              CreateTestGraph(),
		},
	}
}

func (m *MockSolverClient) Solve(
	ctx context.Context,
	graph *commonv1.Graph,
	algorithm commonv1.Algorithm,
	opts *optimizationv1.SolveOptions,
) (*client.SolveResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SolveCalls++

	// Check for sequence responses
	if m.callIdx < len(m.Results) || m.callIdx < len(m.Errors) {
		var result *client.SolveResult
		var err error

		if m.callIdx < len(m.Results) {
			result = m.Results[m.callIdx]
		}
		if m.callIdx < len(m.Errors) {
			err = m.Errors[m.callIdx]
		}
		m.callIdx++

		if err != nil {
			return nil, err
		}
		if result != nil {
			// Clone and update graph
			result.Graph = graph
			return result, nil
		}
	}

	if m.DefaultError != nil {
		return nil, m.DefaultError
	}

	// Clone default result with current graph
	result := *m.DefaultResult
	result.Graph = graph
	return &result, nil
}

func (m *MockSolverClient) Close() error {
	return nil
}

func (m *MockSolverClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callIdx = 0
	m.SolveCalls = 0
}

// ================== Mock Database ==================

type MockDB struct {
	mu sync.RWMutex

	ExecResult  pgconn.CommandTag
	ExecErr     error
	QueryRows   pgx.Rows
	QueryErr    error
	QueryRowRow pgx.Row
	BeginTxTx   pgx.Tx
	BeginTxErr  error
	PingErr     error

	ExecCalls     int
	QueryCalls    int
	QueryRowCalls int
}

func NewMockDB() *MockDB {
	return &MockDB{}
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecCalls++
	return m.ExecResult, m.ExecErr
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueryCalls++
	return m.QueryRows, m.QueryErr
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueryRowCalls++
	return m.QueryRowRow
}

func (m *MockDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return m.BeginTxTx, m.BeginTxErr
}

func (m *MockDB) Close() {}

func (m *MockDB) Ping(ctx context.Context) error {
	return m.PingErr
}

// ================== Test Data Builders ==================

func CreateTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Name:     "test-graph",
		Metadata: map[string]string{"test": "true"},
		Nodes: []*commonv1.Node{
			{Id: 1, X: 0, Y: 0, Type: commonv1.NodeType_NODE_TYPE_SOURCE, Name: "Source"},
			{Id: 2, X: 1, Y: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Warehouse1"},
			{Id: 3, X: 1, Y: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Name: "Warehouse2"},
			{Id: 4, X: 2, Y: 0, Type: commonv1.NodeType_NODE_TYPE_SINK, Name: "Sink"},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 50, Cost: 1, Length: 10},
			{From: 1, To: 3, Capacity: 50, Cost: 2, Length: 15},
			{From: 2, To: 4, Capacity: 40, Cost: 1, Length: 10},
			{From: 3, To: 4, Capacity: 60, Cost: 1, Length: 12},
			{From: 2, To: 3, Capacity: 20, Cost: 1, Length: 5},
		},
	}
}

func CreateTestGraphWithFlow() *commonv1.Graph {
	g := CreateTestGraph()
	g.Edges[0].CurrentFlow = 40
	g.Edges[1].CurrentFlow = 50
	g.Edges[2].CurrentFlow = 40
	g.Edges[3].CurrentFlow = 50
	g.Edges[4].CurrentFlow = 0
	return g
}

func CreateLargeTestGraph(nodeCount int) *commonv1.Graph {
	nodes := make([]*commonv1.Node, nodeCount)
	edges := make([]*commonv1.Edge, 0)

	for i := 0; i < nodeCount; i++ {
		nodeType := commonv1.NodeType_NODE_TYPE_INTERSECTION
		if i == 0 {
			nodeType = commonv1.NodeType_NODE_TYPE_SOURCE
		} else if i == nodeCount-1 {
			nodeType = commonv1.NodeType_NODE_TYPE_SINK
		}

		nodes[i] = &commonv1.Node{
			Id:   int64(i + 1),
			X:    float64(i % 10),
			Y:    float64(i / 10),
			Type: nodeType,
			Name: "Node" + string(rune('A'+i)),
		}
	}

	// Create edges (forward connections)
	for i := 0; i < nodeCount-1; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     int64(i + 1),
			To:       int64(i + 2),
			Capacity: 100,
			Cost:     1,
			Length:   10,
		})
	}

	// Add some cross edges
	for i := 0; i < nodeCount-2; i += 2 {
		edges = append(edges, &commonv1.Edge{
			From:     int64(i + 1),
			To:       int64(i + 3),
			Capacity: 50,
			Cost:     2,
			Length:   15,
		})
	}

	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   int64(nodeCount),
		Name:     "large-test-graph",
		Nodes:    nodes,
		Edges:    edges,
	}
}

func CreateDisconnectedGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 50},
			// No path from 2 to 4, and node 3 is isolated
		},
	}
}
