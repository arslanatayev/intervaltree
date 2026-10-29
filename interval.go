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

import "reflect"

// ComputeMerge finds overlapping intervals to remove and computes a single merged range for the new addition.
func (t *IntervalTree[K, V]) ComputeMerge(start, end K, value V) (toDelete []Interval[K, V], merged Interval[K, V], isIntersect bool) {
	var altValue V
	toDelete = make([]Interval[K, V], 0, defaultCapacity)

	newStart := start
	newEnd := end

	for interval := range t.FindOverlaps(start, end) {
		toDelete = append(toDelete, interval)

		if t.Cmp(interval.Start, newStart) < 0 {
			newStart = interval.Start
		}
		if t.Cmp(interval.End, newEnd) > 0 {
			newEnd = interval.End
			altValue = interval.Value
		}
	}

	if !reflect.ValueOf(value).IsZero() {
		altValue = value	
	}
	merged = Interval[K, V]{
		Start: newStart,
		End:   newEnd,
		Value: altValue,
	}

	isIntersect = len(toDelete) > 0

	return toDelete, merged, isIntersect
}

// MergeAll merges all overlapping or contiguous intervals in the tree.
// It uses an in-order slice traversal (which is already sorted by start keys),
// merges them in O(N) time, clears the tree, and re-inserts the merged intervals.
// For overlapping intervals, the value of the latest extending interval is preserved.
func (t *IntervalTree[K, V]) MergeAll(nextFunc func(a K) K) {
	if t.Len() <= 1 {
		return
	}

	intervals := t.Slice()
	if len(intervals) == 0 {
		return
	}

	merged := make([]Interval[K, V], 0, t.Len())
	current := intervals[0]

	for i := 1; i < len(intervals); i++ {
		next := intervals[i]

		if t.Cmp(nextFunc(current.End), next.Start) >= 0 {
			if t.Cmp(current.End, next.End) < 0 {
				current.End = next.End
				current.Value = next.Value
			}
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)

	t.Reset()

	for _, interval := range merged {
		t.Insert(interval.Start, interval.End, interval.Value)
	}
}

// ComputeSplit detects intervals to delete and calculates any remaining fragment parts to be re-added.
func (t *IntervalTree[K, V]) ComputeSplit(start, end K, prevFunc, nextFunc func(a K) K) (toDelete, toAdd []Interval[K, V], isIntersect bool) {
	isIntersect = false
	toDelete = make([]Interval[K, V], 0, defaultCapacity)
	toAdd = make([]Interval[K, V], 0, defaultCapacity*2)

	for interval := range t.FindOverlaps(start, end) {
		toDelete = append(toDelete, interval)

		// left fragment: [interval.Start, prev(start)]
		if t.Cmp(interval.Start, start) < 0 {
			toAdd = append(toAdd, Interval[K, V]{
				Start: interval.Start,
				End:   prevFunc(start),
				Value: interval.Value,
			})
		}

		// right fragment: [next(end), interval.End]
		if t.Cmp(interval.End, end) > 0 {
			toAdd = append(toAdd, Interval[K, V]{
				Start: nextFunc(end),
				End:   interval.End,
				Value: interval.Value,
			})
		}
	}

	if len(toDelete) > 0 {
		isIntersect = true
	}

	return toDelete, toAdd, isIntersect
}
