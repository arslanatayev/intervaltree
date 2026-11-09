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

// ComputeMerge computes the result of adding [start, end] with the given value
// without modifying the tree.
//
// toDelete holds the stored intervals that overlap [start, end]; merged covers
// [start, end] and all of toDelete and carries value. The values of toDelete
// are not carried over. To apply: delete toDelete, then insert merged.
func (t *IntervalTree[K, V]) ComputeMerge(start, end K, value V) (toDelete []Interval[K, V], merged Interval[K, V], isIntersect bool) {
	toDelete = make([]Interval[K, V], 0, defaultCapacity)

	newStart := start
	newEnd := end

	for interval := range t.Overlaps(start, end) {
		toDelete = append(toDelete, interval)

		if t.cmp(interval.Start, newStart) < 0 {
			newStart = interval.Start
		}
		if t.cmp(interval.End, newEnd) > 0 {
			newEnd = interval.End
		}
	}

	merged = Interval[K, V]{
		Start: newStart,
		End:   newEnd,
		Value: value,
	}

	isIntersect = len(toDelete) > 0

	return toDelete, merged, isIntersect
}

// ComputeContiguousMerge is like ComputeMerge, but also merges intervals that
// are directly adjacent, such as [10, 20] and [21, 30].
//
// prevFunc and nextFunc return the key before and after a given key (a-1 and
// a+1 for integers). They are not bounds-checked, so guard against overflow at
// the ends of your key domain.
func (t *IntervalTree[K, V]) ComputeContiguousMerge(start, end K, value V, prevFunc, nextFunc func(a K) K) (toDelete []Interval[K, V], merged Interval[K, V], isIntersect bool) {
	toDelete = make([]Interval[K, V], 0, defaultCapacity)

	newStart := start
	newEnd := end

	for interval := range t.Overlaps(prevFunc(start), nextFunc(end)) {
		toDelete = append(toDelete, interval)

		if t.cmp(interval.Start, newStart) < 0 {
			newStart = interval.Start
		}
		if t.cmp(interval.End, newEnd) > 0 {
			newEnd = interval.End
		}
	}

	merged = Interval[K, V]{
		Start: newStart,
		End:   newEnd,
		Value: value,
	}

	isIntersect = len(toDelete) > 0

	return toDelete, merged, isIntersect
}

// MergeAll merges all overlapping or contiguous intervals in the tree.
// Intervals are contiguous if the start of one equals nextFunc(end of the
// other); nextFunc is not bounds-checked.
//
// MergeAll rebuilds the tree, so it invalidates any active iterator.
func (t *IntervalTree[K, V]) MergeAll(nextFunc func(a K) K) {
	if t.Len() <= 1 {
		return
	}

	intervals := t.AllSlice()
	merged := make([]Interval[K, V], 0, t.Len())
	current := intervals[0]

	for i := 1; i < len(intervals); i++ {
		next := intervals[i]

		if t.cmp(nextFunc(current.End), next.Start) >= 0 {
			if t.cmp(current.End, next.End) < 0 {
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

// ComputeSplit computes the result of removing [start, end] from the stored
// intervals without modifying the tree.
//
// toDelete holds the overlapping intervals; toAdd holds their remaining
// fragments outside [start, end], each with the value of the interval it was
// cut from. To apply: delete toDelete, then insert toAdd. prevFunc and
// nextFunc are not bounds-checked.
func (t *IntervalTree[K, V]) ComputeSplit(start, end K, prevFunc, nextFunc func(a K) K) (toDelete, toAdd []Interval[K, V], isIntersect bool) {
	isIntersect = false
	toDelete = make([]Interval[K, V], 0, defaultCapacity)
	toAdd = make([]Interval[K, V], 0, defaultCapacity*2)

	for interval := range t.Overlaps(start, end) {
		toDelete = append(toDelete, interval)

		// left fragment: [interval.Start, prev(start)]
		if t.cmp(interval.Start, start) < 0 {
			toAdd = append(toAdd, Interval[K, V]{
				Start: interval.Start,
				End:   prevFunc(start),
				Value: interval.Value,
			})
		}

		// right fragment: [next(end), interval.End]
		if t.cmp(interval.End, end) > 0 {
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
