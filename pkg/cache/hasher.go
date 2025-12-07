package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

// GraphHash вычисляет хеш графа для использования как ключ кэша
func GraphHash(graph *commonv1.Graph) string {
	if graph == nil {
		return ""
	}

	data := graphToCanonical(graph)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:16])
}

// graphToCanonical создаёт детерминированное представление графа
func graphToCanonical(graph *commonv1.Graph) []byte {
	// Сортируем узлы по ID
	nodeIDs := make([]int64, 0, len(graph.Nodes))
	nodeTypes := make(map[int64]int32)
	for _, node := range graph.Nodes {
		nodeIDs = append(nodeIDs, node.Id)
		nodeTypes[node.Id] = int32(node.Type)
	}
	sort.Slice(nodeIDs, func(i, j int) bool {
		return nodeIDs[i] < nodeIDs[j]
	})

	// Сортируем рёбра
	type edgeData struct {
		from, to int64
		capacity float64
		cost     float64
	}
	edges := make([]edgeData, 0, len(graph.Edges))
	for _, e := range graph.Edges {
		edges = append(edges, edgeData{e.From, e.To, e.Capacity, e.Cost})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].from != edges[j].from {
			return edges[i].from < edges[j].from
		}
		return edges[i].to < edges[j].to
	})

	// Строим каноническую строку
	var result []byte

	// Source и Sink
	result = append(result, []byte(fmt.Sprintf("s:%d,t:%d;", graph.SourceId, graph.SinkId))...)

	// Узлы
	for _, id := range nodeIDs {
		result = append(result, []byte(fmt.Sprintf("n:%d:%d;", id, nodeTypes[id]))...)
	}

	// Рёбра
	for _, e := range edges {
		result = append(result, []byte(fmt.Sprintf("e:%d:%d:%.6f:%.6f;",
			e.from, e.to, e.capacity, e.cost))...)
	}

	return result
}

// BuildSolveKey строит ключ кэша для результата решения
func BuildSolveKey(graphHash, algorithm string) string {
	return fmt.Sprintf("solve:%s:%s", algorithm, graphHash)
}

// BuildSolveKeyWithOptions строит ключ с учётом опций
func BuildSolveKeyWithOptions(graphHash, algorithm, optionsHash string) string {
	if optionsHash == "" {
		return BuildSolveKey(graphHash, algorithm)
	}
	return fmt.Sprintf("solve:%s:%s:%s", algorithm, graphHash, optionsHash)
}

// QuickHash быстрый хеш для произвольных данных
func QuickHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ShortHash короткий хеш (16 символов)
func ShortHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}
