package intervaltree

// ComputeMerge finds overlapping intervals to remove and computes a single merged range for the new addition.
func (t *IntervalTree[K]) ComputeMerge(start, end K) (toDelete []Interval[K], merged Interval[K], isIntersect bool) {
	toDelete = t.FindOverlaps(t.Prev(start), t.Next(end))

	if len(toDelete) == 0 {
		return nil, merged, false
	}

	newStart := start
	newEnd := end

	for _, interval := range toDelete {
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

	return toDelete, merged, true
}

// ComputeSplit detects intervals to delete and calculates any remaining fragment parts to be re-added.
func (t *IntervalTree[K]) ComputeSplit(start, end K) (toDelete, toAdd []Interval[K], isIntersect bool) {
	toDelete = t.FindOverlaps(start, end)

	if len(toDelete) == 0 {
		return nil, nil, false
	}

	toAdd = make([]Interval[K], 0, len(toDelete)*2)

	for _, interval := range toDelete {
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

	return toDelete, toAdd, true
}
