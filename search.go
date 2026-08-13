package intervaltree

// cmpInterval compares two intervals [s1, e1] and [s2, e2].
func (t *IntervalTree[K]) cmpInterval(s1, e1, s2, e2 K) int {
	if cmp := t.Cmp(s1, s2); cmp != 0 {
		return cmp
	}
	return t.Cmp(e1, e2)
}

// ExactMatch traverses the tree to determine if an interval matching [start, end] exactly exists.
func (t *IntervalTree[K]) ExactMatch(start, end K) bool {
	currentIdx := t.RootIdx
	for currentIdx != noIdx {
		cmp := t.cmpInterval(start, end, t.Nodes[currentIdx].Start, t.Nodes[currentIdx].End)
		if cmp < 0 {
			currentIdx = t.Nodes[currentIdx].LeftIdx
		} else if cmp > 0 {
			currentIdx = t.Nodes[currentIdx].RightIdx
		} else {
			return true
		}
	}

	return false
}

// Contains checks if there exists any node in the tree that completely encompasses the given [start, end] range.
func (t *IntervalTree[K]) Contains(start, end K) bool {
	return t.containsHelper(t.RootIdx, start, end)
}

func (t *IntervalTree[K]) containsHelper(idx uint32, start, end K) bool {
	if idx == noIdx {
		return false
	}

	if t.Cmp(t.Nodes[idx].MaxEnd, end) < 0 {
		return false
	}

	if t.Cmp(t.Nodes[idx].Start, start) <= 0 && t.Cmp(end, t.Nodes[idx].End) <= 0 {
		return true
	}

	if t.containsHelper(t.Nodes[idx].LeftIdx, start, end) {
		return true
	}

	if t.Cmp(t.Nodes[idx].Start, start) <= 0 {
		return t.containsHelper(t.Nodes[idx].RightIdx, start, end)
	}

	return false
}

// ContainsOrCovers returns true if the interval [start, end] matches an existing node exactly or is fully encompassed by one.
func (t *IntervalTree[K]) ContainsOrCovers(start, end K) bool {
	return t.ExactMatch(start, end) || t.Contains(start, end)
}

// FindOverlaps searches the tree and returns all intervals that overlap with the query range [start, end].
func (t *IntervalTree[K]) FindOverlaps(start, end K) (starts []K, ends []K) {
	// TODO: Range search for all overlapping intervals
	return starts, ends
}

// AllElements traverses the tree in-order and returns two parallel slices containing the start and end keys of all active intervals.
func (t *IntervalTree[K]) AllElements() (starts []K, ends []K) {
	if t.RootIdx == noIdx {
		return nil, nil
	}

	capacity := len(t.Nodes) - len(t.FreeIndex)
	starts = make([]K, 0, capacity)
	ends = make([]K, 0, capacity)

	var inOrder func(idx uint32)
	inOrder = func(idx uint32) {
		if idx == noIdx {
			return
		}
		inOrder(t.Nodes[idx].LeftIdx)

		starts = append(starts, t.Nodes[idx].Start)
		ends = append(ends, t.Nodes[idx].End)

		inOrder(t.Nodes[idx].RightIdx)
	}

	inOrder(t.RootIdx)
	return starts, ends
}