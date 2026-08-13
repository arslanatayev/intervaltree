package intervaltree

import "math"

const noIdx uint32 = math.MaxUint32

type Node[K any] struct {
	Start     K
	End       K
	MaxEnd    K
    LeftIdx   uint32
    RightIdx  uint32
	ParentIdx uint32
	Height    uint8
}

type CompareFunc[K any] func(a, b K) int

type IntervalTree[K any] struct {
	Nodes     []Node[K]
	FreeIndex []uint32
	RootIdx   uint32
	Cmp       CompareFunc[K]
}

func New[K any](capacity int, cmp CompareFunc[K]) IntervalTree[K] {
	return IntervalTree[K]{
		Nodes:     make([]Node[K], 0, capacity),
		FreeIndex: make([]uint32, 0, capacity),
		RootIdx:   noIdx,
		Cmp:       cmp,   
	}
}

func (t *IntervalTree[K]) Reset() {
	t.Nodes = t.Nodes[:0]
	t.FreeIndex = t.FreeIndex[:0]
	t.RootIdx = noIdx
}

func (t *IntervalTree[K]) maxKey(a, b K) K {
	if t.Cmp(a, b) >= 0 {
		return a
	}
	return b
}

func (t *IntervalTree[K]) updateNodeStats(idx uint32) {
	if idx == noIdx {
		return
	}

	leftIdx := t.Nodes[idx].LeftIdx
	rightIdx := t.Nodes[idx].RightIdx

	var leftH, rightH uint8
	currMax := t.Nodes[idx].End

	if leftIdx != noIdx {
		leftH = t.Nodes[leftIdx].Height
		currMax = t.maxKey(currMax, t.Nodes[leftIdx].MaxEnd)
	}
	if rightIdx != noIdx {
		rightH = t.Nodes[rightIdx].Height
		currMax = t.maxKey(currMax, t.Nodes[rightIdx].MaxEnd)
	}

	if leftH > rightH {
		t.Nodes[idx].Height = 1 + leftH
	} else {
		t.Nodes[idx].Height = 1 + rightH
	}
	t.Nodes[idx].MaxEnd = currMax
}

func (t *IntervalTree[K]) getBalance(idx uint32) int {
	if idx == noIdx {
		return 0
	}

	var leftH, rightH int
	if t.Nodes[idx].LeftIdx != noIdx {
		leftH = int(t.Nodes[t.Nodes[idx].LeftIdx].Height)
	}
	if t.Nodes[idx].RightIdx != noIdx {
		rightH = int(t.Nodes[t.Nodes[idx].RightIdx].Height)
	}

	return leftH - rightH
}

// Rotate left
func (t *IntervalTree[K]) rotateLeft(xIdx uint32) uint32 {
	yIdx := t.Nodes[xIdx].RightIdx

	t.Nodes[xIdx].RightIdx = t.Nodes[yIdx].LeftIdx
	if t.Nodes[yIdx].LeftIdx != noIdx {
		t.Nodes[t.Nodes[yIdx].LeftIdx].ParentIdx = xIdx
	}

	parentIdx := t.Nodes[xIdx].ParentIdx
	t.Nodes[yIdx].ParentIdx = parentIdx

	if parentIdx == noIdx {
		t.RootIdx = yIdx
	} else if t.Nodes[parentIdx].LeftIdx == xIdx {
		t.Nodes[parentIdx].LeftIdx = yIdx
	} else {
		t.Nodes[parentIdx].RightIdx = yIdx
	}

	t.Nodes[yIdx].LeftIdx = xIdx
	t.Nodes[xIdx].ParentIdx = yIdx

	t.updateNodeStats(xIdx)
	t.updateNodeStats(yIdx)

	return yIdx
}

// Rotate Right
func (t *IntervalTree[K]) rotateRight(yIdx uint32) uint32 {
	xIdx := t.Nodes[yIdx].LeftIdx

	t.Nodes[yIdx].LeftIdx = t.Nodes[xIdx].RightIdx
	if t.Nodes[xIdx].RightIdx != noIdx {
		t.Nodes[t.Nodes[xIdx].RightIdx].ParentIdx = yIdx
	}

	parentIdx := t.Nodes[yIdx].ParentIdx
	t.Nodes[xIdx].ParentIdx = parentIdx

	if parentIdx == noIdx {
		t.RootIdx = xIdx
	} else if t.Nodes[parentIdx].LeftIdx == yIdx {
		t.Nodes[parentIdx].LeftIdx = xIdx
	} else {
		t.Nodes[parentIdx].RightIdx = xIdx
	}

	t.Nodes[xIdx].RightIdx = yIdx
	t.Nodes[yIdx].ParentIdx = xIdx

	t.updateNodeStats(yIdx)
	t.updateNodeStats(xIdx)

	return xIdx
}

func (t *IntervalTree[K]) rebalance(xIdx uint32, balance int) uint32 {
	// Left tree is heavy (LL or LR)
	if balance > 1 {
		leftIdx := t.Nodes[xIdx].LeftIdx
		// LR: Firstly rotate left
		if t.getBalance(leftIdx) < 0 {
			t.rotateLeft(leftIdx)
		}
		// Rotate right
		return t.rotateRight(xIdx)
	}

	// Right tree is heavy (RR or RL)
	rightIdx := t.Nodes[xIdx].RightIdx
	// RL: Firstly rotate right
	if t.getBalance(rightIdx) > 0 {
		t.rotateRight(rightIdx)
	}
	// Rotate left
	return t.rotateLeft(xIdx)
}

func (t *IntervalTree[K]) CreateNodeAndUpdate(start, end K) {
	var newIndex uint32
	newNode := Node[K]{
		Start:     start,
		End:       end,
		MaxEnd:    end,
		Height:    1,
		LeftIdx:   noIdx,
		RightIdx:  noIdx,
		ParentIdx: noIdx,
	}

	if len(t.FreeIndex) > 0 {
		lastIdx := len(t.FreeIndex) - 1
		newIndex = t.FreeIndex[lastIdx]
		t.FreeIndex = t.FreeIndex[:lastIdx]
		t.Nodes[newIndex] = newNode
	} else {
		newIndex = uint32(len(t.Nodes))
		t.Nodes = append(t.Nodes, newNode)
	}

	// Root control
	if t.RootIdx == noIdx {
		t.RootIdx = newIndex
		return
	}

	// BST Traversal
	currentIdx := t.RootIdx
	for {
		if t.Cmp(start, t.Nodes[currentIdx].Start) < 0 { // start < node.Start
			if t.Nodes[currentIdx].LeftIdx == noIdx {
				t.Nodes[currentIdx].LeftIdx = newIndex
				t.Nodes[newIndex].ParentIdx = currentIdx
				break
			}
			currentIdx = t.Nodes[currentIdx].LeftIdx
		} else { // start >= node.Start
			if t.Nodes[currentIdx].RightIdx == noIdx {
				t.Nodes[currentIdx].RightIdx = newIndex
				t.Nodes[newIndex].ParentIdx = currentIdx
				break
			}
			currentIdx = t.Nodes[currentIdx].RightIdx
		}
	}

	// Bottom-Up MaxEnd Update And Balance
	updateIdx := newIndex
	for updateIdx != noIdx {
		t.updateNodeStats(updateIdx)

		balance := t.getBalance(updateIdx)

		if balance > 1 || balance < -1 {
			updateIdx = t.rebalance(updateIdx, balance)
		}

		updateIdx = t.Nodes[updateIdx].ParentIdx
	}
}

func (t *IntervalTree[K]) DeleteNodeAndUpdate(start, end K) {
	var freeIdx uint32

	currentIdx := t.RootIdx
	for currentIdx != noIdx {
		if t.Cmp(start, t.Nodes[currentIdx].Start) < 0 {
			currentIdx = t.Nodes[currentIdx].LeftIdx
			continue
		} else if t.Cmp(start, t.Nodes[currentIdx].Start) > 0 {
			currentIdx = t.Nodes[currentIdx].RightIdx
			continue
		} else {
			if t.Cmp(end, t.Nodes[currentIdx].End) < 0 {
				currentIdx = t.Nodes[currentIdx].LeftIdx
				continue
			} else if t.Cmp(end, t.Nodes[currentIdx].End) > 0 {
				currentIdx = t.Nodes[currentIdx].RightIdx
				continue
			} else {
				break
			}
		}
	}

	if currentIdx == noIdx {
   		return
	}

	parentIdx := t.Nodes[currentIdx].ParentIdx

	if t.Nodes[currentIdx].LeftIdx == noIdx && t.Nodes[currentIdx].RightIdx == noIdx { // Node has no child
		if parentIdx != noIdx {
			if t.Nodes[parentIdx].LeftIdx == currentIdx {
				t.Nodes[parentIdx].LeftIdx = noIdx
			} else {
				t.Nodes[parentIdx].RightIdx = noIdx
			}
		} else {
			t.RootIdx = noIdx
		}

		freeIdx = currentIdx
	} else if t.Nodes[currentIdx].LeftIdx != noIdx && t.Nodes[currentIdx].RightIdx != noIdx { // Node has two child
		current := t.Nodes[currentIdx].RightIdx

		for t.Nodes[current].LeftIdx != noIdx {
			current = t.Nodes[current].LeftIdx
		}

		successorIdx := current
		parentIdx = t.Nodes[successorIdx].ParentIdx

		t.Nodes[currentIdx].Start = t.Nodes[successorIdx].Start
		t.Nodes[currentIdx].End = t.Nodes[successorIdx].End

		if t.Nodes[parentIdx].LeftIdx == successorIdx {
			t.Nodes[parentIdx].LeftIdx = t.Nodes[successorIdx].RightIdx
		} else {
			t.Nodes[parentIdx].RightIdx = t.Nodes[successorIdx].RightIdx
		}

		if t.Nodes[successorIdx].RightIdx != noIdx {
			t.Nodes[t.Nodes[successorIdx].RightIdx].ParentIdx = parentIdx
		}

		freeIdx = successorIdx
	} else { // Node has one child
		var childIdx uint32

		if t.Nodes[currentIdx].LeftIdx != noIdx {
			childIdx = t.Nodes[currentIdx].LeftIdx
		} else {
			childIdx = t.Nodes[currentIdx].RightIdx
		}

		if parentIdx != noIdx {
			t.Nodes[childIdx].ParentIdx = parentIdx

			if t.Nodes[parentIdx].LeftIdx == currentIdx {
				t.Nodes[parentIdx].LeftIdx = childIdx
			} else {
				t.Nodes[parentIdx].RightIdx = childIdx
			}
		} else {
			t.Nodes[childIdx].ParentIdx = noIdx
			t.RootIdx = childIdx
		}

		freeIdx = currentIdx
	}

	t.FreeIndex = append(t.FreeIndex, freeIdx)

	updateIdx := parentIdx
	for updateIdx != noIdx {
		t.updateNodeStats(updateIdx)

		balance := t.getBalance(updateIdx)

		if balance > 1 || balance < -1 {
			updateIdx = t.rebalance(updateIdx, balance)
		}

		updateIdx = t.Nodes[updateIdx].ParentIdx
	}
}

func (t *IntervalTree[K]) ExactMatch(start, end K) bool {
	// TODO: Tree traversal for exact equality check
	return false
}

func (t *IntervalTree[K]) Contains(start, end K) bool {
	// TODO: Tree traversal for containment
	return false
}

func (t *IntervalTree[K]) ContainsOrCovers(start, end K) bool {
	return t.ExactMatch(start, end) || t.Contains(start, end)
}

func (t *IntervalTree[K]) FindOverlaps(start, end K) (starts []K, ends []K) {
	// TODO: Range search for all overlapping intervals
	return starts, ends
}

func (t *IntervalTree[K]) IntersectForAdd(start, end K) (delStarts, delEnds []K, mergedStart, mergedEnd K, isIntersect bool) {
	// TODO
	return delStarts, delEnds, mergedStart, mergedEnd, false
}

func (t *IntervalTree[K]) IntersectForDelete(start, end K) (delStarts, delEnds []K, addStarts, addEnds []K, isIntersect bool) {
	// TODO
	return delStarts, delEnds, addStarts, addEnds, false
}

// AllElements traverses the tree in-order and returns two parallel slices for all active ranges (starts, ends).
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