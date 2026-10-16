package intervaltree

// IntersectForAdd finds overlapping intervals to remove and computes a single merged range for the new addition.
func (t *IntervalTree[K]) IntersectForAdd(start, end K) (delStarts, delEnds []K, mergedStart, mergedEnd K, isIntersect bool) {
	starts, ends := t.FindOverlaps(t.Prev(start), t.Next(end))

	if len(starts) == 0 {
 	   return nil, nil, start, end, false
	}

	l := len(starts)
	delStarts = make([]K, 0, l)
	delEnds = make([]K, 0, l)

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

// IntersectForDelete detects intervals to delete and calculates any remaining fragment parts to be re-added.
func (t *IntervalTree[K]) IntersectForDelete(start, end K) (delStarts, delEnds, addStarts, addEnds []K, isIntersect bool) {
	starts, ends := t.FindOverlaps(start, end)

	if len(starts) == 0 {
		return nil, nil, nil, nil, false
	}

	l := len(starts)
	delStarts = make([]K, 0, l)
	delEnds = make([]K, 0, l)
	addStarts = make([]K, 0, l*2)
	addEnds = make([]K, 0, l*2)
	
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