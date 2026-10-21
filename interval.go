package intervaltree

// ComputeMerge finds overlapping intervals to remove and computes a single merged range for the new addition.
func (t *IntervalTree[K]) ComputeMerge(start, end K) (delStarts, delEnds []K, mergedStart, mergedEnd K, isIntersect bool) {
	starts, ends := t.FindOverlaps(t.Prev(start), t.Next(end))

	if len(starts) == 0 {
		return nil, nil, start, end, false
	}

	newStart := start
	newEnd := end

	for i := range starts {
		if t.Cmp(starts[i], newStart) < 0 {
			newStart = starts[i]
		}
		if t.Cmp(ends[i], newEnd) > 0 {
			newEnd = ends[i]
		}
	}

	return starts, ends, newStart, newEnd, true
}

// ComputeSplit detects intervals to delete and calculates any remaining fragment parts to be re-added.
func (t *IntervalTree[K]) ComputeSplit(start, end K) (delStarts, delEnds, addStarts, addEnds []K, isIntersect bool) {
	starts, ends := t.FindOverlaps(start, end)

	if len(starts) == 0 {
		return nil, nil, nil, nil, false
	}

	l := len(starts)
	addStarts = make([]K, 0, l*2)
	addEnds = make([]K, 0, l*2)

	for i := range starts {
		// left fragment: [starts[i], prev(start)]
		if t.Cmp(starts[i], start) < 0 {
			addStarts = append(addStarts, starts[i])
			addEnds = append(addEnds, t.Prev(start))
		}

		// right fragment: [next(end), ends[i]]
		if t.Cmp(ends[i], end) > 0 {
			addStarts = append(addStarts, t.Next(end))
			addEnds = append(addEnds, ends[i])
		}
	}

	return starts, ends, addStarts, addEnds, true
}
