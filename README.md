# intervaltree

[![Go Reference](https://pkg.go.dev/badge/github.com/arslanatayev/intervaltree.svg)](https://pkg.go.dev/github.com/arslanatayev/intervaltree)
[![Go Version](https://img.shields.io/github/go-mod/go-version/arslanatayev/intervaltree)](https://github.com/arslanatayev/intervaltree)
[![Release](https://img.shields.io/github/v/release/arslanatayev/intervaltree)](https://github.com/arslanatayev/intervaltree/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A high-performance, memory-efficient, and pointer-free **AVL-augmented Interval Tree** written in Go using generics.

Designed with **Data-Oriented Design (DOD)** principles, `intervaltree` eliminates pointer chasing and minimizes GC overhead by storing nodes in contiguous memory backed by flat slices.

---

## Key Highlights

- **Cache-Friendly & Pointer-Free:** Nodes are indexed via `uint32` descriptors over a flat slice (`[]Node[K, V]`), avoiding pointer fragmentation and dramatically improving CPU cache locality.
- **Zero Heap Allocations on Hot Paths:** Query operations such as `ExactMatch` and `Covers` execute with **0 B/op and 0 allocs/op**.
- **Slot Recycling (O(1) Free-List):** Deleted node slots are recycled via an internal free-index pool, avoiding heap reallocations during continuous insert/delete cycles.
- **Recursion-Free Algorithms:** All traversals utilize fixed-size stack buffers (`[64]uint32`), preventing stack overflows and eliminating runtime frame allocations.
- **Self-Balancing AVL:** Guarantees strict O(log N) search, insertion, and deletion times with dynamic subtree `MaxEnd` rebalancing.
- **Battle-Tested:** Validated against millions of randomized fuzzing iterations and stress-tested for extreme stability.

---

## Benchmarks

Tested on **AMD Ryzen 7 7730U** (Linux amd64, Go 1.24):

```text
BenchmarkInsert-16                  1463577     1475 ns/op       52 B/op    0 allocs/op
BenchmarkExactMatch-16              7317874    313.6 ns/op        0 B/op    0 allocs/op
BenchmarkCovers-16                  6945052    323.5 ns/op        0 B/op    0 allocs/op
BenchmarkOverlaps-16                3624873    567.8 ns/op        0 B/op    0 allocs/op
BenchmarkOverlapsSlice-16           2165779    970.6 ns/op     1185 B/op    2 allocs/op
BenchmarkGet-16                     7119218    711.8 ns/op        0 B/op    0 allocs/op
BenchmarkComputeMerge-16            3748068    630.9 ns/op      405 B/op    1 allocs/op
BenchmarkComputeContiguousMerge-16  3267121    634.7 ns/op      408 B/op    1 allocs/op
BenchmarkMergeAll-16                    680  3514667 ns/op  1015809 B/op    4 allocs/op
BenchmarkComputeSplit-16            1593070     1345 ns/op     1776 B/op    2 allocs/op
BenchmarkDelete-16                  2997571     1066 ns/op        0 B/op    0 allocs/op
BenchmarkGCPause-16                    8691   309375 ns/op        3 B/op    0 allocs/op    395.0 µs/GC
```
![Image](assets/benchmark_output.png)

## Fuzz Testing Verification

The tree balance, index recycling, and overlap invariants were fuzzed over millions of operations without integrity failures or memory leaks:
![Image](assets/fuzz_output.png)

## Installation

```bash
go get github.com/arslanatayev/intervaltree
```

Note: Iterator-based traversal methods (such as Overlaps and All) leverage native range-over-func iterators (iter.Seq), requiring Go 1.23 or higher. Core operations like Insert, Delete, ExactMatch, and Covers work independently.

## Quick Start

```go
package main

import (
	"cmp"
	"fmt"

	"github.com/arslanatayev/intervaltree"
)

func main() {
	intervals := []intervaltree.Interval[int, struct{}]{
		{Start: 10, End: 20, Value: struct{}{}},
		{Start: 15, End: 25, Value: struct{}{}},
		{Start: 30, End: 40, Value: struct{}{}},
	}

    // Initialize an interval tree using NewOrdered for ordered types (integers, floats, strings) with initial capacity
	tree := intervaltree.NewOrdered[int, struct{}](1024)

	// OR: For custom types or explicit comparison control, use New:
	// tree := intervaltree.New[int, struct{}](1024, cmp.Compare[int])

	// Insert intervals and check bool return status using a loop
	for _, interval := range intervals {
		if tree.Insert(interval.Start, interval.End, interval.Value) {
			fmt.Printf("Inserted [%d, %d] successfully!\n", interval.Start, interval.End)
		}
	}

	// 1. Point / Range Coverage (Zero-alloc)
	if tree.Covers(12, 18) {
		fmt.Println("Range [12, 18] is completely covered!")
	}

	// 2. Overlap Query (Zero-alloc)
	for interval := range tree.Overlaps(18, 35) {
		fmt.Printf("Overlapping interval: [%d, %d]\n", interval.Start, interval.End)
	}

	// 3. Exact Match (Zero-alloc)
	exists := tree.ExactMatch(10, 20)
	fmt.Printf("Exact interval [10, 20] exists: %t\n", exists)

	// 4. Delete interval with success check
	if tree.Delete(10, 20) {
		fmt.Println("Deleted [10, 20] successfully!")
	}
}
```

## Interval Computation Features

`intervaltree` includes discrete interval utilities to calculate range mutations prior to applying them:

### Interval Merging (`ComputeMerge`)

Computes merged bounding intervals for inserting adjacent or overlapping segments without redundant nodes:

```go
toDelete, merged, ok := tree.ComputeMerge(21, 29, struct{}{})
if ok {
    // Delete obsolete intervals
    for _, interval := range toDelete {
        tree.Delete(interval.Start, interval.End)
    }
    // Insert single consolidated range
    tree.Insert(merged.Start, merged.End, struct{}{})
}
```

### Contiguous Interval Merging (`ComputeContiguousMerge`)

If you need to merge both overlapping and contiguous intervals for a new addition using custom boundary functions (`prev` and `next`), use `ComputeContiguousMerge`:

```go
prev := func(a int) int { return a - 1 }
next := func(a int) int { return a + 1 }

toDelete, merged, ok := tree.ComputeContiguousMerge(21, 29, struct{}{}, prev, next)
if ok {
    // Delete obsolete overlapping or contiguous intervals
    for _, interval := range toDelete {
        tree.Delete(interval.Start, interval.End)
    }
    // Insert single consolidated range
    tree.Insert(merged.Start, merged.End, struct{}{})
}
```

### Bulk Interval Merging (`MergeAll`)

If you want to consolidate all overlapping or contiguous intervals across the entire tree in a single pass without managing individual computations manually, use `MergeAll`:

```go
next := func(a int) int { return a + 1 } // Next function for int
tree.MergeAll(next)
```

### Interval Splitting / Punch-Hole (`ComputeSplit`)

Computes necessary cuts and fragment ranges when punching an interval hole through existing sets:

```go
prev := func(a int) int { return a - 1 }
next := func(a int) int { return a + 1 }

toDelete, toAdd, ok := tree.ComputeSplit(12, 18)
if ok {
    for _, interval := range toDelete {
        tree.Delete(interval.Start, interval.End)
    }
    for _, interval := range toAdd {
		// Fragments preserve their original associated values automatically
        tree.Insert(interval.Start, interval.End, interval.Value)
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


## Important Notes

### Boundary Safety in Custom Next/Prev Functions
The tree relies on your custom `Next` and `Prev` functions passed to methods like `ComputeSplit` or `MergeAll` for discrete interval logic. **The library does not perform boundary checks on arithmetic operations inside these functions.**

To avoid integer overflow/underflow or panics when reaching domain extremes (e.g., `math.MaxInt` or `math.MinInt`), you must handle boundary checks within your supplied callbacks:

```go
// Example: Safe Prev/Next functions for int
prev := func(v int) int {
    if v == math.MinInt {
        return v // Prevent underflow
    }
    return v - 1
}

next := func(v int) int {
    if v == math.MaxInt {
        return v // Prevent overflow
    }
    return v + 1
}
```

### Mutation During Iteration
Iterator-based methods (`All()`, `Overlaps()`, `Stabs()`) yield elements lazily using Go 1.23 iterators (`iter.Seq`). **Never mutate the tree (`Insert` or `Delete`) while actively iterating over it.** 

Modifying the tree structure during iteration invalidates internal stack states, breaks AVL tree topology due to rotations, and leads to undefined behavior. If you need to modify the tree based on search or traversal results, use the **"Collect-then-Mutate"** pattern. 

For convenience, the library provides dedicated slice-returning methods (`AllSlice()`, `OverlapsSlice()`, `StabsSlice()`) specifically designed for scenarios where you need to collect elements before performing mutations:

```go
// ❌ UNSAFE: Mutating the tree inside the active iterator loop
for interval := range tree.Overlaps(start, end) {
    tree.Delete(interval.Start, interval.End) // DO NOT DO THIS
}

// ✅ SAFE: Use slice-returning methods or slices.Collect first, then perform mutations
overlaps := tree.OverlapsSlice(start, end) // Or tree.AllSlice() for All(), tree.StabsSlice(point) for Stabs()

for _, interval := range overlaps {
    tree.Delete(interval.Start, interval.End)
}
tree.Insert(start, end, struct{}{})
```

### Concurrency

Like Go standard library container types (`container/list`, `map`), `IntervalTree` is not thread-safe by default. If concurrent read/write access is required, protect instances with a `sync.RWMutex`:

```go
type SafeTree[K any, V any] struct {
    mu   sync.RWMutex
    tree *intervaltree.IntervalTree[K, V]
}
```

### Automatic Merging and MergeAll
`intervaltree` **does not perform automatic merging**. The `Insert` method treats each interval independently, meaning it does not automatically merge overlapping ranges and returns `false` if an exact duplicate already exists. 

If you want to consolidate all overlapping or contiguous intervals across the entire tree in a single pass, you can use the **`MergeAll()`** method:

```go
next := func(a int) int { return a + 1 }
tree.MergeAll(next)
```

## LICENSE

This project is licensed under the Apache license, Version 2.0. See the [LICENSE](LICENSE) file for details.
