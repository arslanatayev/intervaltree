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

// ExactMatch traverses the tree to determine if an interval matching [start, end] exactly exists.
func (t *IntervalTree[K, V]) ExactMatch(start, end K) bool {
	target := Interval[K, V]{
		Start: start,
		End:   end,
	}
	currentIdx := t.RootIdx

	for currentIdx != noIdx {
		node := &t.Nodes[currentIdx]
		cmp := t.cmpIntervals(target, node.Interval)
		if cmp < 0 {
			currentIdx = node.LeftIdx
		} else if cmp > 0 {
			currentIdx = node.RightIdx
		} else {
			return true
		}
	}

	return false
}

// Covers checks if there exists any node in the tree that completely encompasses the given [start, end] range.
// It uses an iterative stack with MaxEnd pruning for maximum performance.
func (t *IntervalTree[K, V]) Covers(start, end K) bool {
	if t.RootIdx == noIdx {
		return false
	}

	var stack [64]uint32
	stack[0] = t.RootIdx
	stackIdx := 1

	for stackIdx > 0 {
		stackIdx--
		currentIdx := stack[stackIdx]
		node := &t.Nodes[currentIdx]

		if t.Cmp(node.MaxEnd, end) < 0 {
			continue
		}

		if t.Cmp(node.Start, start) <= 0 && t.Cmp(end, node.End) <= 0 {
			return true
		}

		if t.Cmp(node.Start, start) <= 0 && node.RightIdx != noIdx {
			stack[stackIdx] = node.RightIdx
			stackIdx++
		}

		if node.LeftIdx != noIdx {
			stack[stackIdx] = node.LeftIdx
			stackIdx++
		}
	}

	return false
}

// Overlaps searches the tree and returns iterator over all intervals that overlap with the query range [start, end].
func (t *IntervalTree[K, V]) Overlaps(start, end K) iter.Seq[Interval[K, V]] {
	return func(yield func(Interval[K, V]) bool) {
		if t.RootIdx == noIdx {
			return
		}

		var stack [64]uint32
		stack[0] = t.RootIdx
		stackIdx := 1

		for stackIdx > 0 {
			stackIdx--
			currentIdx := stack[stackIdx]
			node := &t.Nodes[currentIdx]

			if t.Cmp(node.MaxEnd, start) < 0 {
				continue
			}

			if t.Cmp(node.Start, end) <= 0 && t.Cmp(start, node.End) <= 0 {
				if !yield(node.Interval) {
					return
				}
			}

			if t.Cmp(node.Start, end) <= 0 && node.RightIdx != noIdx {
				stack[stackIdx] = node.RightIdx
				stackIdx++
			}

			if node.LeftIdx != noIdx {
				stack[stackIdx] = node.LeftIdx
				stackIdx++
			}
		}
	}
}

// OverlapsSlice returns all intervals that overlap with the query range [start, end] as a slice.
// This is convenient when you need a concrete slice instead of an iterator, or when performing
// tree mutations (Insert/Delete) based on search results to avoid concurrent iterator modification issues.
func (t *IntervalTree[K, V]) OverlapsSlice(start, end K) []Interval[K, V] {
	intervals := make([]Interval[K, V], 0, 8)

	for interval := range t.Overlaps(start, end) {
		intervals = append(intervals, interval)
	}

	return intervals
}

// All traverses the tree in-order and returns iterator over all active intervals containing the start and end keys.
// It uses an iterative stack to achieve zero-allocation traversal without recursion overhead.
func (t *IntervalTree[K, V]) All() iter.Seq[Interval[K, V]] {
	return func(yield func(Interval[K, V]) bool) {
		if t.RootIdx == noIdx {
			return
		}

		var stack [64]uint32
		stackIdx := 0
		currentIdx := t.RootIdx

		for currentIdx != noIdx || stackIdx > 0 {
			for currentIdx != noIdx {
				stack[stackIdx] = currentIdx
				stackIdx++
				currentIdx = t.Nodes[currentIdx].LeftIdx
			}

			stackIdx--
			currentIdx = stack[stackIdx]

			if !yield(t.Nodes[currentIdx].Interval) {
				return
			}

			currentIdx = t.Nodes[currentIdx].RightIdx
		}
	}
}

// AllSlice returns all active intervals as an in-order slice.
// This is convenient when you need a concrete slice instead of an iterator, or when performing
// tree mutations (Insert/Delete) based on search results to avoid concurrent iterator modification issues.
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
	return t.RootIdx == noIdx
}

// Len returns the number of active intervals currently stored in the tree.
func (t *IntervalTree[K, V]) Len() int {
	return len(t.Nodes) - len(t.FreeIndex)
}

// Stabs returns iterator over all intervals that contain the given point.
func (t *IntervalTree[K, V]) Stabs(point K) iter.Seq[Interval[K, V]] {
	return t.Overlaps(point, point)
}

// StabsSlice returns all intervals that contain the given point as a slice.
// This is convenient when you need a concrete slice instead of an iterator, or when performing
// tree mutations (Insert/Delete) based on search results to avoid concurrent iterator modification issues.
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
	currentIdx := t.RootIdx

	for currentIdx != noIdx {
		node := &t.Nodes[currentIdx]
		cmp := t.cmpIntervals(target, node.Interval)
		if cmp < 0 {
			currentIdx = node.LeftIdx
		} else if cmp > 0 {
			currentIdx = node.RightIdx
		} else {
			return node.Value, true
		}
	}

	return value, false
}
