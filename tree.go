// Copyright 2026 Arslan Atayev
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package intervaltree implements a high-performance, memory-efficient,
// pointer-free AVL interval tree backed by a flat contiguous slice.
package intervaltree

import "math"

// noIdx represents a sentinel index used to denote a null or non-existent node reference.
const noIdx uint32 = math.MaxUint32

// defaultCapacity defines the initial capacity allocated for slices or buffers to avoid frequent allocations.
const defaultCapacity = 16

// minInt represents the platform-dependent minimum integer value, used for boundary initialization in tree validations.
const minInt = math.MinInt

// Node represents a single interval element stored within a contiguous slice backing the tree.
type Node[K any, V any] struct {
	Interval[K, V]
	MaxEnd    K
	LeftIdx   uint32
	RightIdx  uint32
	ParentIdx uint32
	Height    uint8
}

// CompareFunc defines a 3-way comparison logic between two keys 'a' and 'b'.
// It returns a negative integer if a < b, zero if a == b, or a positive integer if a > b.
type CompareFunc[K any] func(a, b K) int

// Interval represents a closed bounded range [Start, End] where Start <= End.
type Interval[K any, V any] struct {
	Start K
	End   K
	Value V
}

// IntervalTree represents a memory-efficient, slice-backed AVL interval tree.
type IntervalTree[K any, V any] struct {
	Nodes     []Node[K, V]
	FreeIndex []uint32
	RootIdx   uint32
	Cmp       CompareFunc[K]
}

// New constructs and initializes a new IntervalTree with a pre-allocated memory capacity.
func New[K any, V any](capacity int, cmp CompareFunc[K]) *IntervalTree[K, V] {
	return &IntervalTree[K, V]{
		Nodes:     make([]Node[K, V], 0, capacity),
		FreeIndex: make([]uint32, 0, capacity),
		RootIdx:   noIdx,
		Cmp:       cmp,
	}
}

// Reset clears all active nodes and free indices while retaining the underlying slice allocations.
func (t *IntervalTree[K, V]) Reset() {
	clear(t.Nodes)
	t.Nodes = t.Nodes[:0]
	t.FreeIndex = t.FreeIndex[:0]
	t.RootIdx = noIdx
}

// cmpIntervals compares two intervals 'a' and 'b'.
func (t *IntervalTree[K, V]) cmpIntervals(a, b Interval[K, V]) int {
	if cmp := t.Cmp(a.Start, b.Start); cmp != 0 {
		return cmp
	}
	return t.Cmp(a.End, b.End)
}

// maxKey compares two keys 'a' and 'b' and returns the greater key based on the configured comparison function.
func (t *IntervalTree[K, V]) maxKey(a, b K) K {
	if t.Cmp(a, b) >= 0 {
		return a
	}
	return b
}

// updateNodeStats recalculates and updates the Height and MaxEnd values for the node at idx.
func (t *IntervalTree[K, V]) updateNodeStats(idx uint32) {
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
func (t *IntervalTree[K, V]) getBalance(idx uint32) int {
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
func (t *IntervalTree[K, V]) rotateLeft(xIdx uint32) uint32 {
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
func (t *IntervalTree[K, V]) rotateRight(yIdx uint32) uint32 {
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
func (t *IntervalTree[K, V]) rebalance(xIdx uint32, balance int) uint32 {
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
func (t *IntervalTree[K, V]) Clone() *IntervalTree[K, V] {
	nodes := make([]Node[K, V], len(t.Nodes), cap(t.Nodes))
	copy(nodes, t.Nodes)

	freeIndex := make([]uint32, len(t.FreeIndex), cap(t.FreeIndex))
	copy(freeIndex, t.FreeIndex)

	return &IntervalTree[K, V]{
		Nodes:     nodes,
		FreeIndex: freeIndex,
		RootIdx:   t.RootIdx,
		Cmp:       t.Cmp,
	}
}
