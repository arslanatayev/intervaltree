# intervaltree

[![Go Reference](https://pkg.go.dev/badge/github.com/arslanatayev/intervaltree.svg)](https://pkg.go.dev/github.com/arslanatayev/intervaltree)
[![Go Version](https://img.shields.io/github/go-mod/go-version/arslanatayev/intervaltree)](https://github.com/arslanatayev/intervaltree)
[![Release](https://img.shields.io/github/v/release/arslanatayev/intervaltree)](https://github.com/arslanatayev/intervaltree/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A generic, index-based **AVL-augmented interval tree** for Go. Nodes live in a single flat slice and link to each other with `uint32` indices instead of pointers.

The goal is a small allocation footprint and a quiet garbage collector. See [Design](#design) for how it works.

> **Status:** new library, not yet widely used. The API may still change.

---

## Features

- **Index-based nodes:** nodes are stored in a flat slice (`[]node[K, V]`) and reference each other via `uint32` indices. Links take 4 bytes instead of 8.
- **GC-friendly when K and V have no pointers:** if `K` and `V` contain no pointers (integers, floats, `struct{}`, fixed-size arrays), the node slice contains no pointers and the garbage collector does not need to scan it. With `string` keys or pointer-containing values the slice must still be scanned, so this advantage largely disappears.
- **No allocations in lookups and deletes:** in the benchmarks below, `ExactMatch`, `Covers`, `Get`, `Overlaps` (iterator form) and `Delete` report 0 allocs/op. The slice-returning variants (`OverlapsSlice`, etc.) and the `Compute*` helpers allocate by design.
- **Slot recycling:** deleted slots go onto an internal free list and are reused by later inserts.
- **No recursion:** traversals use fixed-size stack buffers (`[64]uint32`) rather than recursive calls.
- **Self-balancing AVL:** insert, delete and exact lookup are O(log N). Each node keeps the maximum `End` of its subtree, which lets overlap queries skip whole subtrees.
- **Generic:** works with any key type via `NewOrdered` (ordered types) or `New` with your own comparison function.
- **Fuzz and stress tested:** fuzz tests check tree balance, index recycling and overlap invariants (sample output below). Stress-tested with N million intervals.

---

## Benchmarks

Absolute numbers for this library only. No comparison with other interval tree implementations was made.

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

## Fuzz Testing

Sample output of the fuzz tests that check tree balance, index recycling and overlap invariants:
![Image](assets/fuzz_output.png)

## Installation

```bash
go get github.com/arslanatayev/intervaltree
```

**Requires Go 1.23 or higher.** The package uses `iter.Seq` for its iterator-based methods (`All`, `Overlaps`, `Stabs`), so the whole package needs Go 1.23, including methods that do not return iterators.

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/arslanatayev/intervaltree"
)

func main() {
	intervals := []intervaltree.Interval[int, struct{}]{
		{Start: 10, End: 20, Value: struct{}{}},
		{Start: 15, End: 25, Value: struct{}{}},
		{Start: 30, End: 40, Value: struct{}{}},
	}

	// NewOrdered works for ordered types (integers, floats, strings).
	// The argument is the initial capacity (number of nodes).
	tree := intervaltree.NewOrdered[int, struct{}](1024)

	// For custom key types or explicit comparison control use New:
	// tree := intervaltree.New[int, struct{}](1024, cmp.Compare[int])

	// Insert returns true if a new interval was added, or false if an
	// interval with the same bounds already existed and its value was replaced.
	for _, interval := range intervals {
		if tree.Insert(interval.Start, interval.End, interval.Value) {
			fmt.Printf("Inserted [%d, %d] successfully!\n", interval.Start, interval.End)
		}
	}

	// 1. Coverage: does a single stored interval fully contain [12, 18]?
	if tree.Covers(12, 18) {
		fmt.Println("Range [12, 18] is covered by a single interval!")
	}

	// 2. Overlap query (iterator)
	for interval := range tree.Overlaps(18, 35) {
		fmt.Printf("Overlapping interval: [%d, %d]\n", interval.Start, interval.End)
	}

	// 3. Exact match
	exists := tree.ExactMatch(10, 20)
	fmt.Printf("Exact interval [10, 20] exists: %t\n", exists)

	// 4. Delete
	if tree.Delete(10, 20) {
		fmt.Println("Deleted [10, 20] successfully!")
	}
}
```

## Semantics

- Intervals are closed: both `Start` and `End` are inclusive.
- `Start` must not be greater than `End`. This is not checked; passing a reversed interval is undefined behavior.
- An interval is identified by its `(Start, End)` pair. Inserting a pair that already exists replaces the stored value and returns `false`, so one `(Start, End)` pair holds exactly one value.
- `Covers(start, end)` is true when **one stored interval** fully contains `[start, end]`. It does not check whether the union of several intervals covers the range. For example, with `[10, 20]` and `[21, 30]` stored, `Covers(15, 25)` is `false` unless you merge them first (see `MergeAll`).
- The tree does not merge intervals automatically. `Insert` treats every interval independently.

## Interval Computation Features

For discrete domains (such as IP addresses or port numbers) the library offers helpers that compute range changes before you apply them. They return what to delete and what to insert, and you apply the result yourself. The computation and the application are separate steps, so they are not atomic: do not modify the tree between them.

### Interval Merging (`ComputeMerge`)

Computes the merged interval for adding a segment that overlaps existing ones:

```go
toDelete, merged, ok := tree.ComputeMerge(21, 29, struct{}{})
if ok {
	for _, interval := range toDelete {
		tree.Delete(interval.Start, interval.End)
	}
	tree.Insert(merged.Start, merged.End, struct{}{})
}
```

The merged interval carries the value you pass in. The values of the replaced intervals are not carried over, but you can read them from `toDelete` before deleting.

### Contiguous Interval Merging (`ComputeContiguousMerge`)

Also merges intervals that touch but do not overlap (for example `[10, 20]` and `[21, 30]`). Because the library cannot know what "adjacent" means for your key type, you supply `prev` and `next` functions:

```go
prev := func(a int) int { return a - 1 }
next := func(a int) int { return a + 1 }

toDelete, merged, ok := tree.ComputeContiguousMerge(21, 29, struct{}{}, prev, next)
if ok {
	for _, interval := range toDelete {
		tree.Delete(interval.Start, interval.End)
	}
	tree.Insert(merged.Start, merged.End, struct{}{})
}
```

### Bulk Interval Merging (`MergeAll`)

Consolidates all overlapping or contiguous intervals in the tree in one pass:

```go
next := func(a int) int { return a + 1 }
tree.MergeAll(next)
```

**Values are not combined.** A merged interval keeps the value of the interval with the greatest `End` in its group (the first one if tied); the other values are discarded. `MergeAll` is meant for trees whose values do not matter, such as `struct{}`. If values matter, merge one interval at a time with `ComputeContiguousMerge`, where you choose the merged value.

### Interval Splitting / Punch-Hole (`ComputeSplit`)

Computes the cuts and remaining fragments when removing a range from existing intervals:

```go
prev := func(a int) int { return a - 1 }
next := func(a int) int { return a + 1 }

toDelete, toAdd, ok := tree.ComputeSplit(12, 18, prev, next)
if ok {
	for _, interval := range toDelete {
		tree.Delete(interval.Start, interval.End)
	}
	for _, interval := range toAdd {
		// Fragments keep the value of the interval they were cut from.
		tree.Insert(interval.Start, interval.End, interval.Value)
	}
}
```

## Design

Pointer-based trees allocate each node separately on the heap. This library keeps all nodes in one slice:

```text
Pointer-based tree:
Heap -> [*node] ---> [*node] ---> [*node]   (one allocation per node)

This implementation:
Memory: [ node 0 | node 1 | node 2 | node 3 | ... ]   (one slice)
             ^         |
             +---------+  (linked by 32-bit indices)
```

1. **Flat layout:** all nodes are in a single contiguous slice.
2. **32-bit indices:** nodes reference children and parents by `uint32` index instead of a 64-bit pointer.
3. **Free list:** deleted slots are pushed onto a LIFO free list (`freeIndex`) and reused by the next insert. The backing slice does not shrink after deletions.

This means far fewer allocations than a pointer-per-node tree. If `K` and `V` contain no pointers, the garbage collector does not scan the node slice, however large the tree is. With `string` keys or pointer-containing values it still does.

When the slice is full it grows and the nodes are copied, so if you know the approximate size in advance, pass it as the initial capacity to `New` or `NewOrdered`.

## Important Notes

### Boundary Safety in Custom Next/Prev Functions
`ComputeContiguousMerge` and `ComputeSplit` rely on the `prev` and `next` functions you pass in, and `MergeAll` relies on `next`. **The library does not check for overflow or underflow inside them.**

To avoid wrap-around at the extremes of your domain (for example `math.MaxInt` or `math.MinInt`), handle the boundaries in your own callbacks, for example by returning `v` when `v == math.MaxInt`.

### Mutation During Iteration
Iterator-based methods (`All()`, `Overlaps()`, `Stabs()`) yield elements lazily using `iter.Seq`. **Do not call `Insert`, `Delete`, `MergeAll` or `Reset` while iterating.** Modifying the tree can trigger rotations that invalidate the traversal state, which leads to undefined behavior.

If you need to change the tree based on query results, collect first and mutate afterwards. The slice-returning methods (`AllSlice()`, `OverlapsSlice()`, `StabsSlice()`) exist for this:

```go
// UNSAFE: mutating the tree inside the iterator loop
for interval := range tree.Overlaps(start, end) {
	tree.Delete(interval.Start, interval.End) // do not do this
}

// SAFE: collect first, then mutate
overlaps := tree.OverlapsSlice(start, end) // or AllSlice(), StabsSlice(point)

for _, interval := range overlaps {
	tree.Delete(interval.Start, interval.End)
}
tree.Insert(start, end, struct{}{})
```

### Copying
Do not copy an `IntervalTree` by value. A copy shares the same backing slices as the original, so changes to one silently affect the other. Use `Clone()` to get an independent copy.

### Concurrency
Like the standard library containers (`container/list`, `map`), `IntervalTree` is not safe for concurrent use. If you need concurrent reads and writes, protect the tree with a `sync.RWMutex`.

## License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.
