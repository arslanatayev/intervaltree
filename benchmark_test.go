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
		_ = tree.Insert(intervals[i][0], intervals[i][1], struct{}{})
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
	rng := rand.New(rand.NewSource(42))
	intervals := make([][2]int, b.N)
	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		intervals[i] = [2]int{s, s + rng.Intn(100)}
	}

	tree := defaultTree(b.N)
	for i := 0; i < b.N; i++ {
		_ = tree.Insert(intervals[i][0], intervals[i][1], struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tree.Get(intervals[i][0], intervals[i][1])
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

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tree := defaultTree(size)
		for j := 0; j < size; j++ {
			s := rng.Intn(50_000)
			_ = tree.Insert(s, s+rng.Intn(100), struct{}{})
		}

		next := func(a int) int { return a + 1 }
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
	rng := rand.New(rand.NewSource(42))
	intervals := make([][2]int, b.N)
	for i := 0; i < b.N; i++ {
		s := rng.Intn(1_000_000)
		intervals[i] = [2]int{s, s + rng.Intn(100)}
	}

	tree := defaultTree(b.N)
	for i := 0; i < b.N; i++ {
		_ = tree.Insert(intervals[i][0], intervals[i][1], struct{}{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = tree.Delete(intervals[i][0], intervals[i][1])
	}
}

func BenchmarkGCPause(b *testing.B) {
	const nodes = 1_000_000
	tree := defaultTree(nodes)
	for i := 0; i < nodes; i++ {
		_ = tree.Insert(i*10, i*10+5, struct{}{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		runtime.GC()
		b.ReportMetric(float64(time.Since(start).Microseconds()), "µs/GC")
	}
}
