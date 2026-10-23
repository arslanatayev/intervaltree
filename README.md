# intervaltree

[![Go Reference](https://pkg.go.dev/badge/github.com/arslanatayev/intervaltree.svg)](https://pkg.go.dev/github.com/arslanatayev/intervaltree)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/arslanatayev/intervaltree)](https://goreportcard.com/report/github.com/arslanatayev/intervaltree)

A high-performance, memory-efficient, and pointer-free **AVL-augmented Interval Tree** written in Go using generics.

Designed with **Data-Oriented Design (DOD)** principles, `intervaltree` eliminates pointer chasing and minimizes GC overhead by storing nodes in contiguous memory backed by flat slices.

---

## Key Highlights

- **Cache-Friendly & Pointer-Free:** Nodes are indexed via `uint32` descriptors over a flat slice (`[]Node[K]`), avoiding pointer fragmentation and dramatically improving CPU cache locality.
- **Zero Heap Allocations on Hot Paths:** Query operations such as `ExactMatch` and `Covers` execute with **0 B/op and 0 allocs/op**.
- **Slot Recycling (O(1) Free-List):** Deleted node slots are recycled via an internal free-index pool, avoiding heap reallocations during continuous insert/delete cycles.
- **Recursion-Free Algorithms:** All traversals utilize fixed-size stack buffers (`[64]uint32`), preventing stack overflows and eliminating runtime frame allocations.
- **Self-Balancing AVL:** Guarantees strict O(log N) search, insertion, and deletion times with dynamic subtree `MaxEnd` rebalancing.
- **Battle-Tested:** Validated against millions of randomized fuzzing iterations and stress-tested for extreme stability.

---

## Benchmarks

Tested on **AMD Ryzen 7 7730U** (Linux amd64, Go 1.24):

```text
BenchmarkInsert-16          1618212    1417 ns/op     44 B/op    0 allocs/op
BenchmarkExactMatch-16      7456974     308.4 ns/op    0 B/op    0 allocs/op
BenchmarkCovers-16          7246612     316.0 ns/op    0 B/op    0 allocs/op
BenchmarkFindOverlaps-16    2240947     951.2 ns/op  790 B/op    5 allocs/op
BenchmarkDelete-16          3336750    1110 ns/op      0 B/op    0 allocs/op
BenchmarkGCPause-16            8697  294530 ns/op      2 B/op    0 allocs/op (426 µs/GC)
```
![Image](assets/benchmark_output.png)

## Fuzz Testing Verification

The tree balance, index recycling, and overlap invariants were fuzzed over millions of operations without integrity failures or memory leaks:
![Image](assets/fuzz_output.png)

## Installation

```bash
go get github.com/arslanatayev/intervaltree
```

## Quick Start

```go
package main

import (
	"cmp"
	"fmt"

	"github.com/arslanatayev/intervaltree"
)

func main() {
	// Initialize an interval tree for integers with initial capacity
	tree := intervaltree.New[int](
		1024,
		cmp.Compare[int],     // Comparison function
		func(v int) int { return v - 1 }, // Prev function
		func(v int) int { return v + 1 }, // Next function
	)

	// Insert intervals [start, end]
	tree.Insert(10, 20)
	tree.Insert(15, 25)
	tree.Insert(30, 40)

	// 1. Point / Range Coverage (Zero-alloc)
	if tree.Covers(12, 18) {
		fmt.Println("Range [12, 18] is completely covered!")
	}

	// 2. Overlap Query
	starts, ends := tree.FindOverlaps(18, 35)
	for i := range starts {
		fmt.Printf("Overlapping interval: [%d, %d]\n", starts[i], ends[i])
	}

	// 3. Exact Match (Zero-alloc)
	exists := tree.ExactMatch(10, 20)
	fmt.Printf("Exact interval [10, 20] exists: %t\n", exists)

	// 4. Delete interval (Zero-alloc)
	tree.Delete(10, 20)
}
```

## Interval Computation Features

`intervaltree` includes discrete interval utilities to calculate range mutations prior to applying them:

## Interval Merging (`ComputeMerge`)

Computes merged bounding intervals for inserting adjacent or overlapping segments without redundant nodes:

```go
delStarts, delEnds, mergedStart, mergedEnd, ok := tree.ComputeMerge(21, 29)
if ok {
    // Delete obsolete intervals
    for i := range delStarts {
        tree.Delete(delStarts[i], delEnds[i])
    }
    // Insert single consolidated range
    tree.Insert(mergedStart, mergedEnd)
}
```

## Interval Splitting / Punch-Hole (`ComputeSplit`)

Computes necessary cuts and fragment ranges when punching an interval hole through existing sets:

```go
delStarts, delEnds, addStarts, addEnds, ok := tree.ComputeSplit(12, 18)
if ok {
    for i := range delStarts {
        tree.Delete(delStarts[i], delEnds[i])
    }
    for i := range addStarts {
        tree.Insert(addStarts[i], addEnds[i])
    }
}
```

## Architecture

Traditional balanced trees allocate each node independently on the heap (`*Node`), introducing pointer overhead and memory fragmentation:

```text
Traditional Tree:
Heap -> [*Node] ---> [*Node] ---> [*Node] (Cache misses, heavy GC work)

This Implementation:
Memory: [ Node 0 | Node 1 | Node 2 | Node 3 | ... ] (Flat, cache-local slice)
             ^         |
             +---------+ (Referenced by 32-bit array indices)
```

1. Flat Memory Layout: All nodes reside in a single contiguous slice.
2. Compact 32-bit Indices: Nodes reference children and parents via `uint32` indices instead of 64-bit machine pointers, saving 50% memory on pointer fields.
3. Internal Memory Pool: Deleted slots are placed on a LIFO free list (`FreeIndex`) and repurposed immediately on subsequent inserts.

## Concurrency Note

Like Go standard library container types (`container/list`, `map`), `IntervalTree` is not thread-safe by default. If concurrent read/write access is required, protect instances with a `sync.RWMutex`:

```go
type SafeTree[K any] struct {
    mu   sync.RWMutex
    tree *intervaltree.IntervalTree[K]
}
```

## LICENSE

This project is licensed under the Apache license, Version 2.0. See the [LICENSE](LICENSE) file for details.
