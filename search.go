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

import "iter"

// ExactMatch reports whether an interval matching [start, end] exactly exists in the tree.
func (t *IntervalTree[K, V]) ExactMatch(start, end K) bool {
    _, ok := t.Get(start, end)
    return ok
}

// Covers reports whether a single stored interval contains all of [start, end].
// The union of several intervals is not considered.
func (t *IntervalTree[K, V]) Covers(start, end K) bool {
	if t.rootIdx == noIdx {
		return false
	}

	var stack [64]uint32
	stack[0] = t.rootIdx
	stackIdx := 1

	for stackIdx > 0 {
		stackIdx--
		currentIdx := stack[stackIdx]
		node := &t.nodes[currentIdx]

		if t.cmp(node.maxEnd, end) < 0 {
			continue
		}

		if t.cmp(node.Start, start) <= 0 && t.cmp(end, node.End) <= 0 {
			return true
		}

		if t.cmp(node.Start, start) <= 0 && node.rightIdx != noIdx {
			stack[stackIdx] = node.rightIdx
			stackIdx++
		}

		if node.leftIdx != noIdx {
			stack[stackIdx] = node.leftIdx
			stackIdx++
		}
	}

	return false
}

// Overlaps searches the tree and returns an iterator over all intervals that overlap with the query range [start, end].
// The tree must not be modified while the iterator is in use.
// To modify the tree based on the results, use OverlapsSlice.
func (t *IntervalTree[K, V]) Overlaps(start, end K) iter.Seq[Interval[K, V]] {
	return func(yield func(Interval[K, V]) bool) {
		if t.rootIdx == noIdx {
			return
		}

		var stack [64]uint32
		stack[0] = t.rootIdx
		stackIdx := 1

		for stackIdx > 0 {
			stackIdx--
			currentIdx := stack[stackIdx]
			node := &t.nodes[currentIdx]

			if t.cmp(node.maxEnd, start) < 0 {
				continue
			}

			if t.cmp(node.Start, end) <= 0 && t.cmp(start, node.End) <= 0 {
				if !yield(node.Interval) {
					return
				}
			}

			if t.cmp(node.Start, end) <= 0 && node.rightIdx != noIdx {
				stack[stackIdx] = node.rightIdx
				stackIdx++
			}

			if node.leftIdx != noIdx {
				stack[stackIdx] = node.leftIdx
				stackIdx++
			}
		}
	}
}

// OverlapsSlice returns all intervals that overlap with the query range [start, end] as a slice.
// The slice is a snapshot: it is safe to modify the tree while using it.
func (t *IntervalTree[K, V]) OverlapsSlice(start, end K) []Interval[K, V] {
	intervals := make([]Interval[K, V], 0, 8)

	for interval := range t.Overlaps(start, end) {
		intervals = append(intervals, interval)
	}

	return intervals
}

// All traverses the tree in-order and returns an iterator over all intervals.
// The tree must not be modified while the iterator is in use.
// To modify the tree based on the results, use AllSlice.
func (t *IntervalTree[K, V]) All() iter.Seq[Interval[K, V]] {
	return func(yield func(Interval[K, V]) bool) {
		if t.rootIdx == noIdx {
			return
		}

		var stack [64]uint32
		stackIdx := 0
		currentIdx := t.rootIdx

		for currentIdx != noIdx || stackIdx > 0 {
			for currentIdx != noIdx {
				stack[stackIdx] = currentIdx
				stackIdx++
				currentIdx = t.nodes[currentIdx].leftIdx
			}

			stackIdx--
			currentIdx = stack[stackIdx]

			if !yield(t.nodes[currentIdx].Interval) {
				return
			}

			currentIdx = t.nodes[currentIdx].rightIdx
		}
	}
}

// AllSlice returns all intervals as an in-order slice.
// The slice is a snapshot: it is safe to modify the tree while using it.
func (t *IntervalTree[K, V]) AllSlice() (intervals []Interval[K, V]) {
	capacity := t.Len()

	intervals = make([]Interval[K, V], 0, capacity)

	for interval := range t.All() {
		intervals = append(intervals, interval)
	}

	return intervals
}

// IsEmpty returns true if the tree contains no intervals.
func (t *IntervalTree[K, V]) IsEmpty() bool {
	return t.rootIdx == noIdx
}

// Len returns the number of intervals currently stored in the tree.
func (t *IntervalTree[K, V]) Len() int {
	return len(t.nodes) - len(t.freeIndex)
}

// Stabs returns an iterator over all intervals that contain the given point.
// The tree must not be modified while the iterator is in use.
// To modify the tree based on the results, use StabsSlice.
func (t *IntervalTree[K, V]) Stabs(point K) iter.Seq[Interval[K, V]] {
	return t.Overlaps(point, point)
}

// StabsSlice returns all intervals that contain the given point as a slice.
// The slice is a snapshot: it is safe to modify the tree while using it.
func (t *IntervalTree[K, V]) StabsSlice(point K) []Interval[K, V] {
	return t.OverlapsSlice(point, point)
}

// Get searches the tree for an interval matching [start, end] exactly and returns its associated
// value and true if found, or zero value and false otherwise.
func (t *IntervalTree[K, V]) Get(start, end K) (value V, ok bool) {
	target := Interval[K, V]{
		Start: start,
		End:   end,
	}
	currentIdx := t.rootIdx

	for currentIdx != noIdx {
		node := &t.nodes[currentIdx]
		cmp := t.cmpIntervals(target, node.Interval)
		if cmp < 0 {
			currentIdx = node.leftIdx
		} else if cmp > 0 {
			currentIdx = node.rightIdx
		} else {
			return node.Value, true
		}
	}

	return value, false
}
