package intervaltree

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func defaultTree(capacity int) *IntervalTree[int] {
	cmp := func(a, b int) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
	prev := func(a int) int { return a - 1 }
	next := func(a int) int { return a + 1 }
	return New[int](capacity, cmp, prev, next)
}

func BenchmarkInsert(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	intervals := make([][2]int, b.N)
	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		intervals[i] = [2]int{s, s + rng.Intn(100)}
	}

	b.ResetTimer()
	b.ReportAllocs()

	tree := defaultTree(b.N)
	for i := 0; i < b.N; i++ {
		tree.Insert(intervals[i][0], intervals[i][1])
	}
}

func BenchmarkExactMatch(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	var queries [][2]int
	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		e := s + rng.Intn(100)
		tree.Insert(s, e)
		if i%10 == 0 {
			queries = append(queries, [2]int{s, e})
		}
	}

	qLen := len(queries)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		q := queries[i%qLen]
		_ = tree.ExactMatch(q[0], q[1])
	}
}

func BenchmarkCovers(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		tree.Insert(s, s+rng.Intn(500))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Covers(s, s+50)
	}
}

func BenchmarkFindOverlaps(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		tree.Insert(s, s+rng.Intn(200))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		for _ = range tree.FindOverlaps(s, s+100) {}
	}
}

func BenchmarkDelete(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	intervals := make([][2]int, b.N)
	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		intervals[i] = [2]int{s, s + rng.Intn(100)}
	}

	tree := defaultTree(b.N)
	for i := 0; i < b.N; i++ {
		tree.Insert(intervals[i][0], intervals[i][1])
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tree.Delete(intervals[i][0], intervals[i][1])
	}
}

func BenchmarkGCPause(b *testing.B) {
	const nodes = 1_000_000
	tree := defaultTree(nodes)
	for i := 0; i < nodes; i++ {
		tree.Insert(i*10, i*10+5)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		runtime.GC()
		b.ReportMetric(float64(time.Since(start).Microseconds()), "µs/GC")
	}
}
