package graph

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPool(t *testing.T) {
	pool := GetPool()

	assert.NotNil(t, pool)
	assert.Equal(t, pool, GetPool()) // Should return same instance
}

func TestGraphPool_AcquireReleaseGraph(t *testing.T) {
	pool := GetPool()

	g := pool.AcquireGraph()
	require.NotNil(t, g)

	// Use the graph
	g.AddNode(1)
	g.AddNode(2)
	g.AddEdgeWithReverse(1, 2, 10, 1)

	assert.Equal(t, 2, g.NodeCount())

	// Release
	pool.ReleaseGraph(g)

	// After release, graph should be cleared
	// Acquire again - may or may not be the same object
	g2 := pool.AcquireGraph()
	assert.Equal(t, 0, g2.NodeCount()) // Should be cleared
	pool.ReleaseGraph(g2)
}

func TestGraphPool_ReleaseNilGraph(t *testing.T) {
	pool := GetPool()

	// Should not panic
	pool.ReleaseGraph(nil)
}

func TestGraphPool_AcquireReleaseInt64Slice(t *testing.T) {
	pool := GetPool()

	s := pool.AcquireInt64Slice()
	require.NotNil(t, s)
	assert.Empty(t, *s)

	// Use the slice
	*s = append(*s, 1, 2, 3)
	assert.Len(t, *s, 3)

	pool.ReleaseInt64Slice(s)

	// Acquire again
	s2 := pool.AcquireInt64Slice()
	assert.Empty(t, *s2)
	pool.ReleaseInt64Slice(s2)
}

func TestGraphPool_ReleaseNilInt64Slice(t *testing.T) {
	pool := GetPool()
	pool.ReleaseInt64Slice(nil) // Should not panic
}

func TestGraphPool_AcquireReleaseInt64Map(t *testing.T) {
	pool := GetPool()

	m := pool.AcquireInt64Map()
	require.NotNil(t, m)
	assert.Empty(t, m)

	m[1] = 100
	m[2] = 200
	assert.Len(t, m, 2)

	pool.ReleaseInt64Map(m)

	m2 := pool.AcquireInt64Map()
	assert.Empty(t, m2)
	pool.ReleaseInt64Map(m2)
}

func TestGraphPool_ReleaseNilInt64Map(t *testing.T) {
	pool := GetPool()
	pool.ReleaseInt64Map(nil) // Should not panic
}

func TestGraphPool_AcquireReleaseFloatMap(t *testing.T) {
	pool := GetPool()

	m := pool.AcquireFloatMap()
	require.NotNil(t, m)
	assert.Empty(t, m)

	m[1] = 1.5
	m[2] = 2.5
	assert.Len(t, m, 2)

	pool.ReleaseFloatMap(m)

	m2 := pool.AcquireFloatMap()
	assert.Empty(t, m2)
	pool.ReleaseFloatMap(m2)
}

func TestGraphPool_ReleaseNilFloatMap(t *testing.T) {
	pool := GetPool()
	pool.ReleaseFloatMap(nil)
}

func TestGraphPool_AcquireReleaseBoolMap(t *testing.T) {
	pool := GetPool()

	m := pool.AcquireBoolMap()
	require.NotNil(t, m)
	assert.Empty(t, m)

	m[1] = true
	m[2] = false
	assert.Len(t, m, 2)

	pool.ReleaseBoolMap(m)

	m2 := pool.AcquireBoolMap()
	assert.Empty(t, m2)
	pool.ReleaseBoolMap(m2)
}

func TestGraphPool_ReleaseNilBoolMap(t *testing.T) {
	pool := GetPool()
	pool.ReleaseBoolMap(nil)
}

func TestGraphPool_AcquireReleaseIntMap(t *testing.T) {
	pool := GetPool()

	m := pool.AcquireIntMap()
	require.NotNil(t, m)
	assert.Empty(t, m)

	m[1] = 10
	m[2] = 20
	assert.Len(t, m, 2)

	pool.ReleaseIntMap(m)

	m2 := pool.AcquireIntMap()
	assert.Empty(t, m2)
	pool.ReleaseIntMap(m2)
}

func TestGraphPool_ReleaseNilIntMap(t *testing.T) {
	pool := GetPool()
	pool.ReleaseIntMap(nil)
}

func TestGraphPool_Concurrency(t *testing.T) {
	pool := GetPool()

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// Acquire resources
				g := pool.AcquireGraph()
				m1 := pool.AcquireInt64Map()
				m2 := pool.AcquireFloatMap()
				m3 := pool.AcquireBoolMap()
				s := pool.AcquireInt64Slice()

				// Use them
				g.AddNode(1)
				m1[1] = 1
				m2[1] = 1.0
				m3[1] = true
				*s = append(*s, 1)

				// Release
				pool.ReleaseGraph(g)
				pool.ReleaseInt64Map(m1)
				pool.ReleaseFloatMap(m2)
				pool.ReleaseBoolMap(m3)
				pool.ReleaseInt64Slice(s)
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// PooledResources Tests
// =============================================================================

func TestNewPooledResources(t *testing.T) {
	pr := NewPooledResources()
	require.NotNil(t, pr)
	defer pr.Release()
}

func TestNewPooledResourcesWithPool(t *testing.T) {
	pool := GetPool()
	pr := NewPooledResourcesWithPool(pool)
	require.NotNil(t, pr)
	defer pr.Release()
}

func TestNewPooledResourcesWithPool_Nil(t *testing.T) {
	pr := NewPooledResourcesWithPool(nil)
	require.NotNil(t, pr)
	defer pr.Release()
}

func TestPooledResources_Graph(t *testing.T) {
	pr := NewPooledResources()
	defer pr.Release()

	g := pr.Graph()
	require.NotNil(t, g)

	g.AddNode(1)
	g.AddNode(2)
	g.AddEdgeWithReverse(1, 2, 10, 0)

	assert.Equal(t, 2, g.NodeCount())

	// Get another graph
	g2 := pr.Graph()
	require.NotNil(t, g2)
	assert.NotEqual(t, g, g2) // Different objects
}

func TestPooledResources_Int64Map(t *testing.T) {
	pr := NewPooledResources()
	defer pr.Release()

	m := pr.Int64Map()
	require.NotNil(t, m)

	m[1] = 100
	assert.Equal(t, int64(100), m[1])
}

func TestPooledResources_FloatMap(t *testing.T) {
	pr := NewPooledResources()
	defer pr.Release()

	m := pr.FloatMap()
	require.NotNil(t, m)

	m[1] = 1.5
	assert.Equal(t, 1.5, m[1])
}

func TestPooledResources_BoolMap(t *testing.T) {
	pr := NewPooledResources()
	defer pr.Release()

	m := pr.BoolMap()
	require.NotNil(t, m)

	m[1] = true
	assert.True(t, m[1])
}

func TestPooledResources_IntMap(t *testing.T) {
	pr := NewPooledResources()
	defer pr.Release()

	m := pr.IntMap()
	require.NotNil(t, m)

	m[1] = 42
	assert.Equal(t, 42, m[1])
}

func TestPooledResources_Int64Slice(t *testing.T) {
	pr := NewPooledResources()
	defer pr.Release()

	s := pr.Int64Slice()
	require.NotNil(t, s)

	*s = append(*s, 1, 2, 3)
	assert.Len(t, *s, 3)
}

func TestPooledResources_Release(t *testing.T) {
	pr := NewPooledResources()

	// Acquire multiple resources
	g1 := pr.Graph()
	g2 := pr.Graph()
	m1 := pr.Int64Map()
	m2 := pr.FloatMap()
	m3 := pr.BoolMap()
	m4 := pr.IntMap()
	s1 := pr.Int64Slice()

	// Use them
	g1.AddNode(1)
	g2.AddNode(2)
	m1[1] = 1
	m2[1] = 1.0
	m3[1] = true
	m4[1] = 1
	*s1 = append(*s1, 1)

	// Release all
	pr.Release()

	// Multiple releases should be safe
	pr.Release()
	pr.Release()
}

func TestPooledResources_Reset(t *testing.T) {
	pr := NewPooledResources()

	g := pr.Graph()
	g.AddNode(1)

	pr.Reset() // Alias for Release()

	// Should be safe to use after reset
	g2 := pr.Graph()
	require.NotNil(t, g2)
	pr.Release()
}

func TestPooledResources_MultipleAcquireReleaseCycles(t *testing.T) {
	pr := NewPooledResources()

	for i := 0; i < 10; i++ {
		g := pr.Graph()
		g.AddNode(int64(i))
		m := pr.FloatMap()
		m[int64(i)] = float64(i)

		pr.Release()
	}
}

func TestPooledResources_Concurrency(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			pr := NewPooledResources()
			defer pr.Release()

			g := pr.Graph()
			g.AddNode(int64(id))

			m := pr.FloatMap()
			m[int64(id)] = float64(id)

			b := pr.BoolMap()
			b[int64(id)] = true

			s := pr.Int64Slice()
			*s = append(*s, int64(id))
		}(i)
	}

	wg.Wait()
}
