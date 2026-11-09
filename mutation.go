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

package intervaltree

// Insert adds the interval [start, end] with the given value. It returns true
// if a new interval was added. If an interval with the same start and end
// already exists, Insert replaces its value and returns false.
func (t *IntervalTree[K, V]) Insert(start, end K, value V) bool {
	if t.cmp(start, end) > 0 {
		return false
	}

	var newIndex uint32
	var isLeft bool
	target := Interval[K, V]{
		Start: start,
		End:   end,
		Value: value,
	}

	newNode := node[K, V]{
		Interval:  target,
		maxEnd:    end,
		height:    1,
		leftIdx:   noIdx,
		rightIdx:  noIdx,
		parentIdx: noIdx,
	}

	// Root control
	if t.rootIdx == noIdx {
		t.rootIdx = t.allocateNode(newNode)
		return true
	}

	// BST Traversal
	currentIdx := t.rootIdx
	for {
		cmp := t.cmpIntervals(target, t.nodes[currentIdx].Interval)
		if cmp < 0 {
			if t.nodes[currentIdx].leftIdx == noIdx {
				isLeft = true
				break
			}
			currentIdx = t.nodes[currentIdx].leftIdx
		} else if cmp > 0 {
			if t.nodes[currentIdx].rightIdx == noIdx {
				isLeft = false
				break
			}
			currentIdx = t.nodes[currentIdx].rightIdx
		} else {
			t.nodes[currentIdx].Value = value
			return false // Same bounds already exist: replace the value, tree shape is unchanged
		}
	}

	newIndex = t.allocateNode(newNode)

	if isLeft {
		t.nodes[currentIdx].leftIdx = newIndex
	} else {
		t.nodes[currentIdx].rightIdx = newIndex
	}

	t.nodes[newIndex].parentIdx = currentIdx

	// Bottom-Up maxEnd Update And Balance
	updateIdx := newIndex
	for updateIdx != noIdx {
		t.updateNodeStats(updateIdx)

		balance := t.getBalance(updateIdx)

		if balance > 1 || balance < -1 {
			updateIdx = t.rebalance(updateIdx, balance)
		}

		updateIdx = t.nodes[updateIdx].parentIdx
	}

	return true
}

// allocateNode allocates a new node either by reusing an index from the freeIndex pool
// or by appending to the nodes slice if no recycled indices are available.
func (t *IntervalTree[K, V]) allocateNode(newNode node[K, V]) uint32 {
	if len(t.freeIndex) > 0 {
		lastIdx := len(t.freeIndex) - 1
		idx := t.freeIndex[lastIdx]
		t.freeIndex = t.freeIndex[:lastIdx]
		t.nodes[idx] = newNode
		return idx
	}
	idx := uint32(len(t.nodes))
	t.nodes = append(t.nodes, newNode)
	return idx
}

// Delete removes the interval with exactly these bounds. It returns false if
// there is no such interval.
func (t *IntervalTree[K, V]) Delete(start, end K) bool {
	var freeIdx uint32

	target := Interval[K, V]{
		Start: start,
		End:   end,
	}

	currentIdx := t.rootIdx
	for currentIdx != noIdx {
		cmp := t.cmpIntervals(target, t.nodes[currentIdx].Interval)
		if cmp < 0 {
			currentIdx = t.nodes[currentIdx].leftIdx
		} else if cmp > 0 {
			currentIdx = t.nodes[currentIdx].rightIdx
		} else {
			break
		}
	}

	if currentIdx == noIdx {
		return false // Interval not found
	}

	parentIdx := t.nodes[currentIdx].parentIdx

	if t.nodes[currentIdx].leftIdx == noIdx && t.nodes[currentIdx].rightIdx == noIdx { // Node has no child
		if parentIdx != noIdx {
			if t.nodes[parentIdx].leftIdx == currentIdx {
				t.nodes[parentIdx].leftIdx = noIdx
			} else {
				t.nodes[parentIdx].rightIdx = noIdx
			}
		} else {
			t.rootIdx = noIdx
		}

		freeIdx = currentIdx
	} else if t.nodes[currentIdx].leftIdx != noIdx && t.nodes[currentIdx].rightIdx != noIdx { // Node has two child
		current := t.nodes[currentIdx].rightIdx

		for t.nodes[current].leftIdx != noIdx {
			current = t.nodes[current].leftIdx
		}

		successorIdx := current
		parentIdx = t.nodes[successorIdx].parentIdx

		t.nodes[currentIdx].Interval = t.nodes[successorIdx].Interval

		if t.nodes[parentIdx].leftIdx == successorIdx {
			t.nodes[parentIdx].leftIdx = t.nodes[successorIdx].rightIdx
		} else {
			t.nodes[parentIdx].rightIdx = t.nodes[successorIdx].rightIdx
		}

		if t.nodes[successorIdx].rightIdx != noIdx {
			t.nodes[t.nodes[successorIdx].rightIdx].parentIdx = parentIdx
		}

		freeIdx = successorIdx
	} else { // Node has one child
		var childIdx uint32

		if t.nodes[currentIdx].leftIdx != noIdx {
			childIdx = t.nodes[currentIdx].leftIdx
		} else {
			childIdx = t.nodes[currentIdx].rightIdx
		}

		if parentIdx != noIdx {
			t.nodes[childIdx].parentIdx = parentIdx

			if t.nodes[parentIdx].leftIdx == currentIdx {
				t.nodes[parentIdx].leftIdx = childIdx
			} else {
				t.nodes[parentIdx].rightIdx = childIdx
			}
		} else {
			t.nodes[childIdx].parentIdx = noIdx
			t.rootIdx = childIdx
		}

		freeIdx = currentIdx
	}

	var noVal K

	t.nodes[freeIdx] = node[K, V]{
		Interval: Interval[K, V]{
			Start: noVal,
			End:   noVal,
		},
		maxEnd:    noVal,
		leftIdx:   noIdx,
		rightIdx:  noIdx,
		parentIdx: noIdx,
		height:    0,
	}
	t.freeIndex = append(t.freeIndex, freeIdx)

	updateIdx := parentIdx
	for updateIdx != noIdx {
		t.updateNodeStats(updateIdx)

		balance := t.getBalance(updateIdx)

		if balance > 1 || balance < -1 {
			updateIdx = t.rebalance(updateIdx, balance)
		}

		updateIdx = t.nodes[updateIdx].parentIdx
	}

	return true
}
