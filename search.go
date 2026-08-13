package intervaltree

// ExactMatch traverses the tree to determine if an interval matching [start, end] exactly exists.
func (t *IntervalTree[K]) ExactMatch(start, end K) bool {
	// TODO: Tree traversal for exact equality check
	return false
}

// Contains checks if there exists any node in the tree that completely encompasses the given [start, end] range.
func (t *IntervalTree[K]) Contains(start, end K) bool {
	// TODO: Tree traversal for containment
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