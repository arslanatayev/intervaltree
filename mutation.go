package intervaltree

// Insert inserts a new interval [start, end] into the tree.
// It reuses a recycled node index from FreeIndex if available, otherwise appends a new node,
// and rebalances the tree bottom-up along the insertion path.
func (t *IntervalTree[K]) Insert(start, end K) {
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

// Delete removes the node that exactly matches the interval [start, end].
// It recycles the freed slot index into FreeIndex for memory reuse and rebalances the tree upwards.
func (t *IntervalTree[K]) Delete(start, end K) {
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