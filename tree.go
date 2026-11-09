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

// Package intervaltree implements an AVL interval tree for closed intervals
// [Start, End], generic over key and value types.
//
// Nodes are stored in one slice and linked by uint32 indices instead of
// pointers. If K and V contain no pointers, the garbage collector does not
// scan the node slice.
//
// An IntervalTree is not safe for concurrent use and must not be copied by
// value; use Clone. Iterator methods require Go 1.23.
package intervaltree

import (
	"cmp"
	"math"
)

// noIdx represents a sentinel index used to denote a null or non-existent node reference.
const noIdx uint32 = math.MaxUint32

// defaultCapacity defines the initial capacity allocated for slices or buffers to avoid frequent allocations.
const defaultCapacity = 16

// node represents a single interval element stored within a contiguous slice backing the tree.
type node[K any, V any] struct {
	Interval[K, V]
	maxEnd    K
	leftIdx   uint32
	rightIdx  uint32
	parentIdx uint32
	height    uint8
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

// IntervalTree represents slice-backed AVL interval tree.
type IntervalTree[K any, V any] struct {
	nodes     []node[K, V]
	freeIndex []uint32
	rootIdx   uint32
	cmp       CompareFunc[K]
}

// New constructs and initializes a new IntervalTree with a pre-allocated memory capacity.
func New[K any, V any](capacity int, cmp CompareFunc[K]) *IntervalTree[K, V] {
	return &IntervalTree[K, V]{
		nodes:     make([]node[K, V], 0, capacity),
		freeIndex: make([]uint32, 0, capacity),
		rootIdx:   noIdx,
		cmp:       cmp,
	}
}

// NewOrdered initializes and returns a new empty IntervalTree pre-allocated with the
// specified capacity for key types that satisfy cmp.Ordered (all integer, float, and string types).
// It automatically configures cmp.Compare as the comparison function.
func NewOrdered[K cmp.Ordered, V any](capacity int) *IntervalTree[K, V] {
	return New[K, V](capacity, cmp.Compare[K])
}

// Reset clears all active nodes and free indices while retaining the underlying slice allocations.
func (t *IntervalTree[K, V]) Reset() {
	clear(t.nodes)
	t.nodes = t.nodes[:0]
	t.freeIndex = t.freeIndex[:0]
	t.rootIdx = noIdx
}

// cmpIntervals compares two intervals 'a' and 'b'.
func (t *IntervalTree[K, V]) cmpIntervals(a, b Interval[K, V]) int {
	if cmp := t.cmp(a.Start, b.Start); cmp != 0 {
		return cmp
	}
	return t.cmp(a.End, b.End)
}

// maxKey compares two keys 'a' and 'b' and returns the greater key based on the configured comparison function.
func (t *IntervalTree[K, V]) maxKey(a, b K) K {
	if t.cmp(a, b) >= 0 {
		return a
	}
	return b
}

// updateNodeStats recalculates and updates the height and maxEnd values for the node at idx.
func (t *IntervalTree[K, V]) updateNodeStats(idx uint32) {
	if idx == noIdx {
		return
	}

	leftIdx := t.nodes[idx].leftIdx
	rightIdx := t.nodes[idx].rightIdx

	var leftH, rightH uint8
	currMax := t.nodes[idx].End

	if leftIdx != noIdx {
		leftH = t.nodes[leftIdx].height
		currMax = t.maxKey(currMax, t.nodes[leftIdx].maxEnd)
	}
	if rightIdx != noIdx {
		rightH = t.nodes[rightIdx].height
		currMax = t.maxKey(currMax, t.nodes[rightIdx].maxEnd)
	}

	if leftH > rightH {
		t.nodes[idx].height = 1 + leftH
	} else {
		t.nodes[idx].height = 1 + rightH
	}
	t.nodes[idx].maxEnd = currMax
}

// getBalance calculates and returns the AVL balance factor (LeftHeight - RightHeight) of the node at idx.
func (t *IntervalTree[K, V]) getBalance(idx uint32) int {
	if idx == noIdx {
		return 0
	}

	var leftH, rightH int
	if t.nodes[idx].leftIdx != noIdx {
		leftH = int(t.nodes[t.nodes[idx].leftIdx].height)
	}
	if t.nodes[idx].rightIdx != noIdx {
		rightH = int(t.nodes[t.nodes[idx].rightIdx].height)
	}

	return leftH - rightH
}

// rotateLeft performs a standard AVL left rotation on the subtree rooted at xIdx.
func (t *IntervalTree[K, V]) rotateLeft(xIdx uint32) uint32 {
	yIdx := t.nodes[xIdx].rightIdx

	t.nodes[xIdx].rightIdx = t.nodes[yIdx].leftIdx
	if t.nodes[yIdx].leftIdx != noIdx {
		t.nodes[t.nodes[yIdx].leftIdx].parentIdx = xIdx
	}

	parentIdx := t.nodes[xIdx].parentIdx
	t.nodes[yIdx].parentIdx = parentIdx

	if parentIdx == noIdx {
		t.rootIdx = yIdx
	} else if t.nodes[parentIdx].leftIdx == xIdx {
		t.nodes[parentIdx].leftIdx = yIdx
	} else {
		t.nodes[parentIdx].rightIdx = yIdx
	}

	t.nodes[yIdx].leftIdx = xIdx
	t.nodes[xIdx].parentIdx = yIdx

	t.updateNodeStats(xIdx)
	t.updateNodeStats(yIdx)

	return yIdx
}

// rotateRight performs a standard AVL right rotation on the subtree rooted at yIdx.
func (t *IntervalTree[K, V]) rotateRight(yIdx uint32) uint32 {
	xIdx := t.nodes[yIdx].leftIdx

	t.nodes[yIdx].leftIdx = t.nodes[xIdx].rightIdx
	if t.nodes[xIdx].rightIdx != noIdx {
		t.nodes[t.nodes[xIdx].rightIdx].parentIdx = yIdx
	}

	parentIdx := t.nodes[yIdx].parentIdx
	t.nodes[xIdx].parentIdx = parentIdx

	if parentIdx == noIdx {
		t.rootIdx = xIdx
	} else if t.nodes[parentIdx].leftIdx == yIdx {
		t.nodes[parentIdx].leftIdx = xIdx
	} else {
		t.nodes[parentIdx].rightIdx = xIdx
	}

	t.nodes[xIdx].rightIdx = yIdx
	t.nodes[yIdx].parentIdx = xIdx

	t.updateNodeStats(yIdx)
	t.updateNodeStats(xIdx)

	return xIdx
}

// rebalance checks the balance factor of xIdx and executes necessary single or double rotations.
func (t *IntervalTree[K, V]) rebalance(xIdx uint32, balance int) uint32 {
	if balance > 1 {
		leftIdx := t.nodes[xIdx].leftIdx
		if t.getBalance(leftIdx) < 0 {
			t.rotateLeft(leftIdx)
		}
		return t.rotateRight(xIdx)
	}

	if balance < -1 {
		rightIdx := t.nodes[xIdx].rightIdx
		if t.getBalance(rightIdx) > 0 {
			t.rotateRight(rightIdx)
		}
		return t.rotateLeft(xIdx)
	}
	return xIdx
}

// Clone returns an independent copy of the tree. Intervals are copied by
// assignment, so if K or V contain pointers, the data they point to is shared.
func (t *IntervalTree[K, V]) Clone() *IntervalTree[K, V] {
	nodes := make([]node[K, V], len(t.nodes), cap(t.nodes))
	copy(nodes, t.nodes)

	freeIdx := make([]uint32, len(t.freeIndex), cap(t.freeIndex))
	copy(freeIdx, t.freeIndex)

	return &IntervalTree[K, V]{
		nodes:     nodes,
		freeIndex: freeIdx,
		rootIdx:   t.rootIdx,
		cmp:       t.cmp,
	}
}