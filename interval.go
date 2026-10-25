package intervaltree

// ComputeMerge finds overlapping intervals to remove and computes a single merged range for the new addition.
func (t *IntervalTree[K]) ComputeMerge(start, end K) (toDelete []Interval[K], merged Interval[K], isIntersect bool) {
	isIntersect = false
	capacity := (len(t.Nodes) - len(t.FreeIndex)) / 4
	toDelete = make([]Interval[K], 0, capacity)

	newStart := start
	newEnd := end

	for interval := range t.FindOverlaps(t.Prev(start), t.Next(end)) {
		toDelete = append(toDelete, interval)

		if t.Cmp(interval.Start, newStart) < 0 {
			newStart = interval.Start
		}
		if t.Cmp(interval.End, newEnd) > 0 {
			newEnd = interval.End
		}
	}

	merged = Interval[K]{
		Start: newStart,
		End:   newEnd,
	}

	if len(toDelete) > 0 {
		isIntersect = true
	}

	return toDelete, merged, isIntersect
}

// ComputeSplit detects intervals to delete and calculates any remaining fragment parts to be re-added.
func (t *IntervalTree[K]) ComputeSplit(start, end K) (toDelete, toAdd []Interval[K], isIntersect bool) {
	isIntersect = false
	capacity := (len(t.Nodes) - len(t.FreeIndex)) / 4
	toDelete = make([]Interval[K], 0, capacity)
	toAdd = make([]Interval[K], 0, capacity*2)

	for interval := range t.FindOverlaps(start, end) {
		toDelete = append(toDelete, interval)

		// left fragment: [interval.Start, prev(start)]
		if t.Cmp(interval.Start, start) < 0 {
			toAdd = append(toAdd, Interval[K]{
				Start: interval.Start,
				End:   t.Prev(start),
			})
		}

		// right fragment: [next(end), interval.End]
		if t.Cmp(interval.End, end) > 0 {
			toAdd = append(toAdd, Interval[K]{
				Start: t.Next(end),
				End:   interval.End,
			})
		}
	}

	if len(toDelete) > 0 {
		isIntersect = true
	}

	return toDelete, toAdd, isIntersect
}
