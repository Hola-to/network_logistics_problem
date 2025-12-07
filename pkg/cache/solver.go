package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
)

// SolverCache специализированный кэш для результатов solver
type SolverCache struct {
	cache      Cache
	defaultTTL time.Duration
}

// CachedSolveResult кэшированный результат
type CachedSolveResult struct {
	MaxFlow           float64          `json:"max_flow"`
	TotalCost         float64          `json:"total_cost"`
	Status            string           `json:"status"`
	Iterations        int32            `json:"iterations"`
	ComputationTimeMs float64          `json:"computation_time_ms"`
	FlowEdges         []*FlowEdgeCache `json:"flow_edges,omitempty"`
	ComputedAt        time.Time        `json:"computed_at"`
}

// FlowEdgeCache кэшированное ребро с потоком
type FlowEdgeCache struct {
	From        int64   `json:"from"`
	To          int64   `json:"to"`
	Flow        float64 `json:"flow"`
	Capacity    float64 `json:"capacity"`
	Utilization float64 `json:"utilization"`
}

// NewSolverCache создаёт кэш для solver результатов
func NewSolverCache(cache Cache, defaultTTL time.Duration) *SolverCache {
	if defaultTTL <= 0 {
		defaultTTL = 10 * time.Minute
	}
	return &SolverCache{
		cache:      cache,
		defaultTTL: defaultTTL,
	}
}

// Get получает кэшированный результат
func (sc *SolverCache) Get(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm) (*CachedSolveResult, bool, error) {
	graphHash := GraphHash(graph)
	key := BuildSolveKey(graphHash, algorithm.String())

	data, err := sc.cache.Get(ctx, key)
	if err != nil {
		if err == ErrKeyNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	var result CachedSolveResult
	if err := json.Unmarshal(data, &result); err != nil {
		// Повреждённый кэш — удаляем, ошибку удаления игнорируем намеренно
		_ = sc.cache.Delete(ctx, key) //nolint:errcheck // best effort cleanup
		return nil, false, nil
	}

	return &result, true, nil
}

// Set сохраняет результат в кэш
func (sc *SolverCache) Set(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, result *CachedSolveResult, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = sc.defaultTTL
	}

	graphHash := GraphHash(graph)
	key := BuildSolveKey(graphHash, algorithm.String())

	result.ComputedAt = time.Now()

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return sc.cache.Set(ctx, key, data, ttl)
}

// SetFromResponse сохраняет результат из SolveResponse
func (sc *SolverCache) SetFromResponse(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, resp *optimizationv1.SolveResponse, ttl time.Duration) error {
	if resp == nil || resp.Result == nil {
		return nil
	}

	result := &CachedSolveResult{
		MaxFlow:           resp.Result.MaxFlow,
		TotalCost:         resp.Result.TotalCost,
		Status:            resp.Result.Status.String(),
		Iterations:        resp.Result.Iterations,
		ComputationTimeMs: resp.Result.ComputationTimeMs,
	}

	// Кэшируем flow edges
	for _, edge := range resp.Result.Edges {
		result.FlowEdges = append(result.FlowEdges, &FlowEdgeCache{
			From:        edge.From,
			To:          edge.To,
			Flow:        edge.Flow,
			Capacity:    edge.Capacity,
			Utilization: edge.Utilization,
		})
	}

	return sc.Set(ctx, graph, algorithm, result, ttl)
}

// Invalidate удаляет кэш для графа
func (sc *SolverCache) Invalidate(ctx context.Context, graph *commonv1.Graph) error {
	graphHash := GraphHash(graph)
	pattern := fmt.Sprintf("solve:*:%s", graphHash)
	_, err := sc.cache.DeleteByPattern(ctx, pattern)
	return err
}

// InvalidateAll удаляет весь кэш solver результатов
func (sc *SolverCache) InvalidateAll(ctx context.Context) (int64, error) {
	return sc.cache.DeleteByPattern(ctx, "solve:*")
}

// ToFlowResult конвертирует кэшированный результат в FlowResult
func (r *CachedSolveResult) ToFlowResult() *commonv1.FlowResult {
	result := &commonv1.FlowResult{
		MaxFlow:           r.MaxFlow,
		TotalCost:         r.TotalCost,
		Iterations:        r.Iterations,
		ComputationTimeMs: r.ComputationTimeMs,
	}

	// Парсим статус
	if v, ok := commonv1.FlowStatus_value[r.Status]; ok {
		result.Status = commonv1.FlowStatus(v)
	}

	// Конвертируем edges
	for _, e := range r.FlowEdges {
		result.Edges = append(result.Edges, &commonv1.FlowEdge{
			From:        e.From,
			To:          e.To,
			Flow:        e.Flow,
			Capacity:    e.Capacity,
			Utilization: e.Utilization,
		})
	}

	return result
}
