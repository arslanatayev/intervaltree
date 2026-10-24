package intervaltree

// ExactMatch traverses the tree to determine if an interval matching [start, end] exactly exists.
func (t *IntervalTree[K]) ExactMatch(start, end K) bool {
	target := Interval[K]{
		Start: start,
		End:   end,
	}
	currentIdx := t.RootIdx
	nodesLen := uint32(len(t.Nodes))

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
func (t *IntervalTree[K]) Covers(start, end K) bool {
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

// FindOverlaps searches the tree and returns all intervals that overlap with the query range [start, end].
func (t *IntervalTree[K]) FindOverlaps(start, end K) (intervals []Interval[K]) {
	if t.RootIdx == noIdx {
		return intervals
	}

	var stack [64]uint32
	stack[0] = t.RootIdx
	stackIdx := 1

	intervals = make([]Interval[K], 0, 8)

	for stackIdx > 0 {
		stackIdx--
		currentIdx := stack[stackIdx]
		node := &t.Nodes[currentIdx]

		if t.Cmp(node.MaxEnd, start) < 0 {
			continue
		}

		if t.Cmp(node.Start, end) <= 0 && t.Cmp(start, node.End) <= 0 {
			intervals = append(intervals, node.Interval)
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

	return intervals
}

// All traverses the tree in-order and returns all active intervals containing the start and end keys.
// It uses an iterative stack to achieve zero-allocation traversal without recursion overhead.
func (t *IntervalTree[K]) All() (intervals []Interval[K]) {
	if t.RootIdx == noIdx {
		return nil
	}

	capacity := len(t.Nodes) - len(t.FreeIndex)
	intervals = make([]Interval[K], 0, capacity)

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

		intervals = append(intervals, t.Nodes[currentIdx].Interval)

		currentIdx = t.Nodes[currentIdx].RightIdx
	}

	return intervals
}

// IsEmpty returns true if the tree contains no intervals.
func (t *IntervalTree[K]) IsEmpty() bool {
	return t.RootIdx == noIdx
}

// Len returns the number of active intervals currently stored in the tree.
func (t *IntervalTree[K]) Len() int {
	return len(t.Nodes) - len(t.FreeIndex)
}

// Stabs returns all intervals that contain the given point.
func (t *IntervalTree[K]) Stabs(point K) (intervals []Interval[K]) {
	return t.FindOverlaps(point, point)
}