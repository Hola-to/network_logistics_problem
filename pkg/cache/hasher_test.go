package cache

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestGraphHash(t *testing.T) {
	t.Run("nil graph", func(t *testing.T) {
		hash := GraphHash(nil)
		if hash != "" {
			t.Errorf("GraphHash(nil) = %v, want empty string", hash)
		}
	})

	t.Run("same graph produces same hash", func(t *testing.T) {
		g := &commonv1.Graph{
			SourceId: 1,
			SinkId:   4,
			Nodes: []*commonv1.Node{
				{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
				{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
				{Id: 4, Type: commonv1.NodeType_NODE_TYPE_SINK},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10, Cost: 1},
				{From: 2, To: 4, Capacity: 5, Cost: 2},
			},
		}

		hash1 := GraphHash(g)
		hash2 := GraphHash(g)

		if hash1 != hash2 {
			t.Errorf("same graph should produce same hash: %v != %v", hash1, hash2)
		}
	})

	t.Run("different graphs produce different hashes", func(t *testing.T) {
		g1 := &commonv1.Graph{
			SourceId: 1,
			SinkId:   2,
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
		}
		g2 := &commonv1.Graph{
			SourceId: 1,
			SinkId:   2,
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 20}}, // different capacity
		}

		hash1 := GraphHash(g1)
		hash2 := GraphHash(g2)

		if hash1 == hash2 {
			t.Error("different graphs should produce different hashes")
		}
	})

	t.Run("node order does not affect hash", func(t *testing.T) {
		g1 := &commonv1.Graph{
			SourceId: 1,
			SinkId:   3,
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
		}
		g2 := &commonv1.Graph{
			SourceId: 1,
			SinkId:   3,
			Nodes:    []*commonv1.Node{{Id: 3}, {Id: 1}, {Id: 2}}, // different order
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
		}

		hash1 := GraphHash(g1)
		hash2 := GraphHash(g2)

		if hash1 != hash2 {
			t.Error("node order should not affect hash")
		}
	})
}

func TestBuildSolveKey(t *testing.T) {
	key := BuildSolveKey("abc123", "DINIC")
	expected := "solve:DINIC:abc123"
	if key != expected {
		t.Errorf("BuildSolveKey() = %v, want %v", key, expected)
	}
}

func TestBuildSolveKeyWithOptions(t *testing.T) {
	tests := []struct {
		name        string
		graphHash   string
		algorithm   string
		optionsHash string
		expected    string
	}{
		{
			name:        "without options",
			graphHash:   "abc123",
			algorithm:   "DINIC",
			optionsHash: "",
			expected:    "solve:DINIC:abc123",
		},
		{
			name:        "with options",
			graphHash:   "abc123",
			algorithm:   "DINIC",
			optionsHash: "opt456",
			expected:    "solve:DINIC:abc123:opt456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := BuildSolveKeyWithOptions(tt.graphHash, tt.algorithm, tt.optionsHash)
			if key != tt.expected {
				t.Errorf("BuildSolveKeyWithOptions() = %v, want %v", key, tt.expected)
			}
		})
	}
}

func TestQuickHash(t *testing.T) {
	data := []byte("test data")
	hash := QuickHash(data)

	if len(hash) != 64 { // SHA256 hex = 64 chars
		t.Errorf("QuickHash length = %d, want 64", len(hash))
	}

	// Same data should produce same hash
	hash2 := QuickHash(data)
	if hash != hash2 {
		t.Error("same data should produce same hash")
	}
}

func TestShortHash(t *testing.T) {
	data := []byte("test data")
	hash := ShortHash(data)

	if len(hash) != 16 {
		t.Errorf("ShortHash length = %d, want 16", len(hash))
	}
}
