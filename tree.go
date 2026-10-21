// Package intervaltree implements a high-performance, memory-efficient,
// pointer-free AVL interval tree backed by a flat contiguous slice.
package intervaltree

import "math"

// noIdx represents a sentinel index used to denote a null or non-existent node reference.
const noIdx uint32 = math.MaxUint32

// Node represents a single interval element stored within a contiguous slice backing the tree.
type Node[K any] struct {
	Start     K
	End       K
	MaxEnd    K
	LeftIdx   uint32
	RightIdx  uint32
	ParentIdx uint32
	Height    uint8
}

// CompareFunc defines a 3-way comparison logic between two keys 'a' and 'b'.
// It returns a negative integer if a < b, zero if a == b, or a positive integer if a > b.
type CompareFunc[K any] func(a, b K) int

// NextFunc returns the immediate successor of a key (e.g., a + 1 for integers).
type NextFunc[K any] func(a K) K

// PrevFunc returns the immediate predecessor of a key (e.g., a - 1 for integers).
type PrevFunc[K any] func(a K) K

// IntervalTree represents a memory-efficient, slice-backed AVL interval tree.
type IntervalTree[K any] struct {
	Nodes     []Node[K]
	FreeIndex []uint32
	RootIdx   uint32
	Cmp       CompareFunc[K]
	Next      NextFunc[K]
	Prev      PrevFunc[K]
}

// New constructs and initializes a new IntervalTree with a pre-allocated memory capacity.
func New[K any](capacity int, cmp CompareFunc[K], prev PrevFunc[K], next NextFunc[K]) *IntervalTree[K] {
	return &IntervalTree[K]{
		Nodes:     make([]Node[K], 0, capacity),
		FreeIndex: make([]uint32, 0, capacity),
		RootIdx:   noIdx,
		Cmp:       cmp,
		Next:      next,
		Prev:      prev,
	}
}

// Reset clears all active nodes and free indices while retaining the underlying slice allocations.
func (t *IntervalTree[K]) Reset() {
	t.Nodes = t.Nodes[:0]
	t.FreeIndex = t.FreeIndex[:0]
	t.RootIdx = noIdx
}

// cmpInterval compares two intervals [s1, e1] and [s2, e2].
func (t *IntervalTree[K]) cmpInterval(s1, e1, s2, e2 K) int {
	if cmp := t.Cmp(s1, s2); cmp != 0 {
		return cmp
	}
	return t.Cmp(e1, e2)
}

// maxKey compares two keys 'a' and 'b' and returns the greater key based on the configured comparison function.
func (t *IntervalTree[K]) maxKey(a, b K) K {
	if t.Cmp(a, b) >= 0 {
		return a
	}
	return b
}

// updateNodeStats recalculates and updates the Height and MaxEnd values for the node at idx.
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

// getBalance calculates and returns the AVL balance factor (LeftHeight - RightHeight) of the node at idx.
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

// rotateLeft performs a standard AVL left rotation on the subtree rooted at xIdx.
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

// rotateRight performs a standard AVL right rotation on the subtree rooted at yIdx.
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

// rebalance checks the balance factor of xIdx and executes necessary single or double rotations.
func (t *IntervalTree[K]) rebalance(xIdx uint32, balance int) uint32 {
	if balance > 1 {
		leftIdx := t.Nodes[xIdx].LeftIdx
		if t.getBalance(leftIdx) < 0 {
			t.rotateLeft(leftIdx)
		}
		return t.rotateRight(xIdx)
	}

	if balance < -1 {
		rightIdx := t.Nodes[xIdx].RightIdx
		if t.getBalance(rightIdx) > 0 {
			t.rotateRight(rightIdx)
		}
		return t.rotateLeft(xIdx)
	}
	return xIdx
}

// Clone creates a fast, deep copy of the IntervalTree, duplicating the underlying
// node and free-index slices while preserving the structural state and function pointers.
func (t *IntervalTree[K]) Clone() *IntervalTree[K] {
	nodes := make([]Node[K], len(t.Nodes), cap(t.Nodes))
	copy(nodes, t.Nodes)

	freeIndex := make([]uint32, len(t.FreeIndex), cap(t.FreeIndex))
	copy(freeIndex, t.FreeIndex)

	return &IntervalTree[K]{
		Nodes:     nodes,
		FreeIndex: freeIndex,
		RootIdx:   t.RootIdx,
		Cmp:       t.Cmp,
		Next:      t.Next,
		Prev:      t.Prev,
	}
}
