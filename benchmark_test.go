// Copyright 2026 Arslan Atayev
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package intervaltree

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func defaultTree(capacity int) *IntervalTree[int, struct{}] {
	cmp := func(a, b int) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}

	return New[int, struct{}](capacity, cmp)
}

// benchSize is the tree size used by the lookup, insert and delete benchmarks.
const benchSize = 100_000

// genIntervals returns n unique random intervals with starts in [0, 1_000_000)
// and lengths in [0, maxLen). The seed is fixed so every benchmark that uses it
// sees the same data.
func genIntervals(n, maxLen int) [][2]int {
	rng := rand.New(rand.NewSource(42))
	seen := make(map[[2]int]struct{}, n)
	intervals := make([][2]int, 0, n)
	for len(intervals) < n {
		s := rng.Intn(1_000_000)
		iv := [2]int{s, s + rng.Intn(maxLen)}
		if _, ok := seen[iv]; ok {
			continue
		}
		seen[iv] = struct{}{}
		intervals = append(intervals, iv)
	}
	return intervals
}

func BenchmarkInsert(b *testing.B) {
	intervals := genIntervals(benchSize, 100)
	tree := defaultTree(benchSize)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % benchSize
		if i > 0 && idx == 0 {
			b.StopTimer()
			tree.Reset()
			b.StartTimer()
		}
		_ = tree.Insert(intervals[idx][0], intervals[idx][1], struct{}{})
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
		_ = tree.Insert(s, e, struct{}{})
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
		_ = tree.Insert(s, s+rng.Intn(500), struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Covers(s, s+50)
	}
}

func BenchmarkOverlaps(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Insert(s, s+rng.Intn(200), struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		for range tree.Overlaps(s, s+100) {
		}
	}
}

func BenchmarkOverlapsSlice(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Insert(s, s+rng.Intn(200), struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.OverlapsSlice(s, s+100)
	}
}

func BenchmarkGet(b *testing.B) {
	intervals := genIntervals(benchSize, 100)
	tree := defaultTree(benchSize)
	for _, iv := range intervals {
		_ = tree.Insert(iv[0], iv[1], struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		iv := intervals[i%benchSize]
		_, _ = tree.Get(iv[0], iv[1])
	}
}

func BenchmarkComputeMerge(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Insert(s, s+rng.Intn(100), struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		_, _, _ = tree.ComputeMerge(s, s+50, struct{}{})
	}
}

func BenchmarkComputeContiguousMerge(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Insert(s, s+rng.Intn(100), struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	prev := func(a int) int { return a - 1 }
	next := func(a int) int { return a + 1 }
	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		_, _, _ = tree.ComputeContiguousMerge(s, s+50, struct{}{}, prev, next)
	}
}

func BenchmarkMergeAll(b *testing.B) {
	const size = 10_000
	rng := rand.New(rand.NewSource(42))
	next := func(a int) int { return a + 1 }

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tree := defaultTree(size)
		for j := 0; j < size; j++ {
			s := rng.Intn(50_000)
			_ = tree.Insert(s, s+rng.Intn(100), struct{}{})
		}
		b.StartTimer()

		tree.MergeAll(next)
	}
}

func BenchmarkComputeSplit(b *testing.B) {
	const size = 100_000
	tree := defaultTree(size)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < size; i++ {
		s := rng.Intn(1_000_000)
		_ = tree.Insert(s, s+rng.Intn(200), struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	prev := func(a int) int { return a - 1 }
	next := func(a int) int { return a + 1 }
	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		_, _, _ = tree.ComputeSplit(s, s+100, prev, next)
	}
}

func BenchmarkDelete(b *testing.B) {
	intervals := genIntervals(benchSize, 100)
	tree := defaultTree(benchSize)
	fill := func() {
		for _, iv := range intervals {
			_ = tree.Insert(iv[0], iv[1], struct{}{})
		}
	}
	fill()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % benchSize
		if i > 0 && idx == 0 {
			b.StopTimer()
			tree.Reset()
			fill()
			b.StartTimer()
		}
		_ = tree.Delete(intervals[idx][0], intervals[idx][1])
	}
}

func BenchmarkGCPause(b *testing.B) {
	const nodes = 1_000_000
	tree := defaultTree(nodes)
	for i := 0; i < nodes; i++ {
		_ = tree.Insert(i*10, i*10+5, struct{}{})
	}

	b.ResetTimer()

	var total time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		runtime.GC()
		total += time.Since(start)
	}

	b.StopTimer()
	b.ReportMetric(float64(total.Nanoseconds())/1e3/float64(b.N), "µs/GC")

	// Sanity check: the tree must still be live, so the heap should hold it.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	b.ReportMetric(float64(ms.HeapAlloc)/(1<<20), "heapMB")

	runtime.KeepAlive(tree)
}
