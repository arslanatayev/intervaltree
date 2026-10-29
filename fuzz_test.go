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

import (
	"bytes"
	"encoding/binary"
	"sort"
	"testing"
)

// referenceModel is a brute-force reference model that cannot make errors.
type referenceModel struct {
	intervals [][2]int
}

func (m *referenceModel) insert(start, end int) {
	for _, it := range m.intervals {
		if it[0] == start && it[1] == end {
			return // duplicate
		}
	}
	m.intervals = append(m.intervals, [2]int{start, end})
}

func (m *referenceModel) delete(start, end int) {
	for i, it := range m.intervals {
		if it[0] == start && it[1] == end {
			m.intervals = append(m.intervals[:i], m.intervals[i+1:]...)
			return
		}
	}
}

func (m *referenceModel) exactMatch(start, end int) bool {
	for _, it := range m.intervals {
		if it[0] == start && it[1] == end {
			return true
		}
	}
	return false
}

func (m *referenceModel) covers(start, end int) bool {
	for _, it := range m.intervals {
		if it[0] <= start && end <= it[1] {
			return true
		}
	}
	return false
}

func (m *referenceModel) findOverlaps(start, end int) [][2]int {
	var res [][2]int
	for _, it := range m.intervals {
		if it[0] <= end && start <= it[1] {
			res = append(res, it)
		}
	}
	return res
}

func (m *referenceModel) all() [][2]int {
	res := make([][2]int, len(m.intervals))
	copy(res, m.intervals)
	sort.Slice(res, func(i, j int) bool {
		if res[i][0] != res[j][0] {
			return res[i][0] < res[j][0]
		}
		return res[i][1] < res[j][1]
	})
	return res
}

func (m *referenceModel) computeMerge(start, end int) (toDelete [][2]int, merged [2]int, isIntersect bool) {
	newStart := start
	newEnd := end
	for _, it := range m.intervals {
		if it[0] <= end && start <= it[1] {
			toDelete = append(toDelete, it)
			if it[0] < newStart {
				newStart = it[0]
			}
			if it[1] > newEnd {
				newEnd = it[1]
			}
		}
	}
	merged = [2]int{newStart, newEnd}
	isIntersect = len(toDelete) > 0
	return toDelete, merged, isIntersect
}

func (m *referenceModel) mergeAll() {
	if len(m.intervals) <= 1 {
		return
	}

	sort.Slice(m.intervals, func(i, j int) bool {
		if m.intervals[i][0] != m.intervals[j][0] {
			return m.intervals[i][0] < m.intervals[j][0]
		}
		return m.intervals[i][1] < m.intervals[j][1]
	})

	var merged [][2]int
	current := m.intervals[0]

	for i := 1; i < len(m.intervals); i++ {
		next := m.intervals[i]
		if next[0] <= current[1] + 1 {
			if next[1] > current[1] {
				current[1] = next[1]
			}
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)
	m.intervals = merged
}

func (m *referenceModel) computeSplit(start, end int) (toDelete, toAdd [][2]int, isIntersect bool) {
	for _, it := range m.intervals {
		if it[0] <= end && start <= it[1] {
			toDelete = append(toDelete, it)
			if it[0] < start {
				toAdd = append(toAdd, [2]int{it[0], start - 1})
			}
			if it[1] > end {
				toAdd = append(toAdd, [2]int{end + 1, it[1]})
			}
		}
	}
	isIntersect = len(toDelete) > 0
	return toDelete, toAdd, isIntersect
}

// FuzzIntervalTree tests memory integrity and algorithmic correctness of the library.
func FuzzIntervalTree(f *testing.F) {
	cmp := func(a, b int) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
	prev := func(a int) int { return a - 1 }
	next := func(a int) int { return a + 1 }

	// Seed Corpus: Scenarios to trigger rotations and edge cases
	f.Add([]byte{
		0, 10, 20, // Insert [10, 20]
		0, 5, 15,  // Insert [5, 15]
		0, 25, 30, // Insert [25, 30]
		1, 5, 15,  // Delete [5, 15]
		0, 10, 20, // Duplicate Insert [10, 20]
		2, 10, 20, // ExactMatch
		3, 12, 18, // Covers
		4, 15, 25, // FindOverlaps
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 3 {
			return
		}

		tree := New[int, struct{}](16, cmp)
		ref := &referenceModel{}
		reader := bytes.NewReader(data)

		for reader.Len() >= 5 {
			var op byte
			var rawStart, rawEnd int16

			_ = binary.Read(reader, binary.LittleEndian, &op)
			_ = binary.Read(reader, binary.LittleEndian, &rawStart)
			_ = binary.Read(reader, binary.LittleEndian, &rawEnd)

			// Constrain values (-200..200) to increase collision/overlap probability
			s := int(rawStart % 200)
			e := int(rawEnd % 200)

			// Precondition guarantee: start <= end
			if s > e {
				s, e = e, s
			}

			switch op % 9 {
			case 0: // INSERT
				_ = tree.Insert(s, e, struct{}{})
				ref.insert(s, e)

			case 1: // DELETE
				_ = tree.Delete(s, e)
				ref.delete(s, e)

			case 2: // EXACT MATCH
				treeRes := tree.ExactMatch(s, e)
				refRes := ref.exactMatch(s, e)
				if treeRes != refRes {
					t.Fatalf("ExactMatch(%d, %d) mismatch: Tree=%v, Reference=%v", s, e, treeRes, refRes)
				}

			case 3: // COVERS
				treeRes := tree.Covers(s, e)
				refRes := ref.covers(s, e)
				if treeRes != refRes {
					t.Fatalf("Covers(%d, %d) mismatch: Tree=%v, Reference=%v", s, e, treeRes, refRes)
				}

			case 4: // FIND OVERLAPS
				overlaps := tree.FindOverlapsSlice(s, e)
				refOverlaps := ref.findOverlaps(s, e)

				if len(overlaps) != len(refOverlaps) {
					t.Fatalf("FindOverlaps(%d, %d) size error: Tree=%d, Reference=%d", s, e, len(overlaps), len(refOverlaps))
				}

				// Sort overlapping intervals and verify
				treeIntervals := make([][2]int, len(overlaps))
				for i := range overlaps {
					treeIntervals[i] = [2]int{overlaps[i].Start, overlaps[i].End}
				}
				sort.Slice(treeIntervals, func(i, j int) bool {
					if treeIntervals[i][0] != treeIntervals[j][0] {
						return treeIntervals[i][0] < treeIntervals[j][0]
					}
					return treeIntervals[i][1] < treeIntervals[j][1]
				})
				sort.Slice(refOverlaps, func(i, j int) bool {
					if refOverlaps[i][0] != refOverlaps[j][0] {
						return refOverlaps[i][0] < refOverlaps[j][0]
					}
					return refOverlaps[i][1] < refOverlaps[j][1]
				})

				for i := range treeIntervals {
					if treeIntervals[i] != refOverlaps[i] {
						t.Fatalf("FindOverlaps element mismatch: Tree=%v, Reference=%v", treeIntervals[i], refOverlaps[i])
					}
				}

			case 5: // Slice
				overlaps := tree.Slice()
				refAll := ref.all()

				if len(overlaps) != len(refAll) {
					t.Fatalf("Slice() element count mismatch: Tree=%d, Reference=%d", len(overlaps), len(refAll))
				}
				for i := range overlaps {
					if overlaps[i].Start != refAll[i][0] || overlaps[i].End != refAll[i][1] {
						t.Fatalf("Slice() sorting/content mismatch at %d: Tree=[%d,%d], Reference=%v", i, overlaps[i].Start, overlaps[i].End, refAll[i])
					}
				}

			case 6: // COMPUTE MERGE
				treeDel, treeMerged, treeIntersect := tree.ComputeMerge(s, e, struct{}{})
				refDel, refMerged, refIntersect := ref.computeMerge(s, e)

				if treeIntersect != refIntersect {
					t.Fatalf("ComputeMerge intersect mismatch at (%d, %d): Tree=%v, Ref=%v", s, e, treeIntersect, refIntersect)
				}
				if treeMerged.Start != refMerged[0] || treeMerged.End != refMerged[1] {
					t.Fatalf("ComputeMerge merged range mismatch at (%d, %d): Tree=[%d,%d], Ref=%v", s, e, treeMerged.Start, treeMerged.End, refMerged)
				}
				if len(treeDel) != len(refDel) {
					t.Fatalf("ComputeMerge toDelete count mismatch: Tree=%d, Ref=%d", len(treeDel), len(refDel))
				}

			case 7: // MERGE ALL
				tree.MergeAll(next)
				ref.mergeAll()

			case 8: // COMPUTE SPLIT
				treeDel, treeAdd, treeIntersect := tree.ComputeSplit(s, e, prev, next)
				refDel, refAdd, refIntersect := ref.computeSplit(s, e)

				if treeIntersect != refIntersect {
					t.Fatalf("ComputeSplit intersect mismatch at (%d, %d): Tree=%v, Ref=%v", s, e, treeIntersect, refIntersect)
				}
				if len(treeDel) != len(refDel) {
					t.Fatalf("ComputeSplit toDelete count mismatch: Tree=%d, Ref=%d", len(treeDel), len(refDel))
				}
				if len(treeAdd) != len(refAdd) {
					t.Fatalf("ComputeSplit toAdd count mismatch: Tree=%d, Ref=%d", len(treeAdd), len(refAdd))
				}
			}

			// CHECK ALL TREE INVARIANTS AFTER EVERY OPERATION
			verifyTreeInvariants(t, tree)
		}
	})
}

// verifyTreeInvariants validates the tree's AVL rules, MaxEnd calculations, and parent-child pointers.
func verifyTreeInvariants(t *testing.T, tree *IntervalTree[int, struct{}]) {
	if tree.RootIdx == noIdx {
		activeCount := len(tree.Nodes) - len(tree.FreeIndex)
		if activeCount != 0 {
			t.Fatalf("Root is empty but active node count is not 0: %d", activeCount)
		}
		return
	}

	visited := make(map[uint32]bool)
	var validateNode func(idx, expectedParent uint32) (height uint8, maxEnd int)

	validateNode = func(idx, expectedParent uint32) (uint8, int) {
		if idx == noIdx {
			return 0, minInt
		}

		if visited[idx] {
			t.Fatalf("Cycle detected in the tree! Node index: %d", idx)
		}
		visited[idx] = true

		node := tree.Nodes[idx]

		// 1. Parent verification
		if node.ParentIdx != expectedParent {
			t.Fatalf("Incorrect ParentIdx for node %d. Expected: %d, Actual: %d", idx, expectedParent, node.ParentIdx)
		}

		// 2. BST Ordering verification
		if node.LeftIdx != noIdx {
			left := tree.Nodes[node.LeftIdx]
			if tree.cmpIntervals(left.Interval, node.Interval) >= 0 {
				t.Fatalf("BST rule violation: Left child [%d, %d] >= Root [%d, %d]", left.Start, left.End, node.Start, node.End)
			}
		}
		if node.RightIdx != noIdx {
			right := tree.Nodes[node.RightIdx]
			if tree.cmpIntervals(right.Interval, node.Interval) <= 0 {
				t.Fatalf("BST rule violation: Right child [%d, %d] <= Root [%d, %d]", right.Start, right.End, node.Start, node.End)
			}
		}

		// Recursively validate subtrees
		leftH, leftMax := validateNode(node.LeftIdx, idx)
		rightH, rightMax := validateNode(node.RightIdx, idx)

		// 3. AVL Height and Balance Verification
		expectedH := 1 + max(leftH, rightH)
		if node.Height != expectedH {
			t.Fatalf("Node %d invalid height. Expected: %d, Got: %d", idx, expectedH, node.Height)
		}

		balance := int(leftH) - int(rightH)
		if balance < -1 || balance > 1 {
			t.Fatalf("AVL balance factor violation! Node %d, Balance: %d", idx, balance)
		}

		// 4. MaxEnd verification
		expectedMax := node.End
		if node.LeftIdx != noIdx && leftMax > expectedMax {
			expectedMax = leftMax
		}
		if node.RightIdx != noIdx && rightMax > expectedMax {
			expectedMax = rightMax
		}
		if node.MaxEnd != expectedMax {
			t.Fatalf("Node %d invalid MaxEnd. Expected: %d, Got: %d", idx, expectedMax, node.MaxEnd)
		}

		return node.Height, node.MaxEnd
	}

	validateNode(tree.RootIdx, noIdx)

	// 5. Memory Leak / Unreachable Node Check
	activeCount := len(tree.Nodes) - len(tree.FreeIndex)
	if len(visited) != activeCount {
		t.Fatalf("Active node count mismatch! Visited in tree: %d, Calculated (Nodes - FreeIndex): %d", len(visited), activeCount)
	}
}