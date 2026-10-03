package intervaltree

// IntersectForAdd evaluates an incoming interval [start, end] against existing rules in the tree.
// It identifies overlapping intervals to be deleted and calculates a single merged interval [mergedStart, mergedEnd] for insertion.
func (t *IntervalTree[K]) IntersectForAdd(start, end K) (delStarts, delEnds []K, mergedStart, mergedEnd K, isIntersect bool) {
	starts, ends := t.FindOverlaps(t.Prev(start), t.Next(end))

	if len(starts) == 0 {
 	   return nil, nil, start, end, false
	}

	newStart := start
	newEnd := end

	for i := range starts {
    	delStarts = append(delStarts, starts[i])
    	delEnds = append(delEnds, ends[i])

    	if t.Cmp(starts[i], newStart) < 0 {
    	    newStart = starts[i]
    	}
    	if t.Cmp(ends[i], newEnd) > 0 {
    	    newEnd = ends[i]
    	}
	}

	return delStarts, delEnds, newStart, newEnd, true
}

// IntersectForDelete evaluates a deletion target [start, end] against overlapping intervals in the tree.
// It determines which existing intervals must be removed and calculates residual range fragments to be added back.
func (t *IntervalTree[K]) IntersectForDelete(start, end K) (delStarts, delEnds, addStarts, addEnds []K, isIntersect bool) {
	starts, ends := t.FindOverlaps(start, end)

	if len(starts) == 0 {
		return nil, nil, nil, nil, false
	}

	for i := range starts {
    	delStarts = append(delStarts, starts[i])
    	delEnds = append(delEnds, ends[i])

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

	return delStarts, delEnds, addStarts, addEnds, true
}