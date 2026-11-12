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
	"math"
	"slices"
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

func (m *referenceModel) Overlaps(start, end int) [][2]int {
	var res [][2]int
	for _, it := range m.intervals {
		if it[0] <= end && start <= it[1] {
			res = append(res, it)
		}
	}
	return res
}

func (m *referenceModel) stabs(point int) [][2]int {
	var res [][2]int
	for _, it := range m.intervals {
		if it[0] <= point && point <= it[1] {
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

func (m *referenceModel) computeContiguousMerge(start, end int) (toDelete [][2]int, merged [2]int, isIntersect bool) {
	newStart := start
	newEnd := end
	for _, it := range m.intervals {
		if it[0] <= end+1 && start-1 <= it[1] {
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
		if next[0] <= current[1]+1 {
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

// pairsOf converts intervals to [start, end] pairs and drops the values.
func pairsOf[V any](intervals []Interval[int, V]) [][2]int {
	res := make([][2]int, len(intervals))
	for i, it := range intervals {
		res[i] = [2]int{it.Start, it.End}
	}
	return res
}

func sortedPairs(pairs [][2]int) [][2]int {
	res := make([][2]int, len(pairs))
	copy(res, pairs)
	sort.Slice(res, func(i, j int) bool {
		if res[i][0] != res[j][0] {
			return res[i][0] < res[j][0]
		}
		return res[i][1] < res[j][1]
	})
	return res
}

// samePairs fails the test unless got and want hold the same intervals.
// The order is ignored; duplicates must match in number.
func samePairs(t *testing.T, what string, got, want [][2]int) {
	t.Helper()
	samePairsInOrder(t, what, sortedPairs(got), sortedPairs(want))
}

// samePairsInOrder fails the test unless got and want are identical, including order.
func samePairsInOrder(t *testing.T, what string, got, want [][2]int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: count mismatch: Tree=%d, Reference=%d", what, len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: element mismatch at %d: Tree=%v, Reference=%v", what, i, got[i], want[i])
		}
	}
}

// verifyAgainstModel checks that the whole content of the tree equals the reference model.
func verifyAgainstModel(t *testing.T, tree *IntervalTree[int, struct{}], ref *referenceModel) {
	t.Helper()
	want := ref.all()

	samePairsInOrder(t, "AllSlice", pairsOf(tree.AllSlice()), want)
	samePairsInOrder(t, "All", pairsOf(slices.Collect(tree.All())), want)

	if tree.Len() != len(want) {
		t.Fatalf("Len() mismatch: Tree=%d, Reference=%d", tree.Len(), len(want))
	}
	if tree.IsEmpty() != (len(want) == 0) {
		t.Fatalf("IsEmpty() mismatch: Tree=%v, Reference count=%d", tree.IsEmpty(), len(want))
	}
}

// fuzzOp encodes one operation in the format the fuzz target reads:
// one op byte followed by two little-endian int16 values.
func fuzzOp(op byte, start, end int16) []byte {
	b := make([]byte, 5)
	b[0] = op
	binary.LittleEndian.PutUint16(b[1:], uint16(start))
	binary.LittleEndian.PutUint16(b[3:], uint16(end))
	return b
}

// FuzzIntervalTree tests memory integrity and algorithmic correctness of the library.
//
// Operation codes (op % 12): 0 Insert, 1 Delete, 2 ExactMatch, 3 Covers,
// 4 Overlaps, 5 Stabs, 6 ComputeMerge, 7 ComputeContiguousMerge, 8 MergeAll,
// 9 ComputeSplit, 10 Get, 11 Clone. An op byte of 250 or more resets the tree.
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

	// Seed corpus: scenarios that trigger rotations, duplicates and every operation.
	f.Add(bytes.Join([][]byte{
		fuzzOp(0, 10, 20), // Insert [10, 20]
		fuzzOp(0, 5, 15),  // Insert [5, 15]
		fuzzOp(0, 25, 30), // Insert [25, 30]
		fuzzOp(1, 5, 15),  // Delete [5, 15]
		fuzzOp(0, 10, 20), // Duplicate Insert [10, 20]
		fuzzOp(2, 10, 20), // ExactMatch
		fuzzOp(3, 12, 18), // Covers
		fuzzOp(4, 15, 25), // Overlaps
		fuzzOp(5, 12, 12), // Stabs
		fuzzOp(6, 12, 28), // ComputeMerge
		fuzzOp(7, 21, 24), // ComputeContiguousMerge
		fuzzOp(9, 12, 18), // ComputeSplit
		fuzzOp(10, 10, 20), // Get
		fuzzOp(11, 1, 2),  // Clone
		fuzzOp(8, 0, 0),   // MergeAll
		fuzzOp(250, 0, 0), // Reset
	}, nil))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 5 {
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

			if op >= 250 { // RESET
				tree.Reset()
				ref.intervals = nil
			} else {
				switch op % 12 {
				case 0: // INSERT
					existed := ref.exactMatch(s, e)
					inserted := tree.Insert(s, e, struct{}{})
					if inserted == existed {
						t.Fatalf("Insert(%d, %d) returned %v, but the interval already existed: %v", s, e, inserted, existed)
					}
					ref.insert(s, e)

				case 1: // DELETE
					existed := ref.exactMatch(s, e)
					deleted := tree.Delete(s, e)
					if deleted != existed {
						t.Fatalf("Delete(%d, %d) returned %v, but the interval existed: %v", s, e, deleted, existed)
					}
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

				case 4: // OVERLAPS (slice and iterator forms)
					want := ref.Overlaps(s, e)
					samePairs(t, "OverlapsSlice", pairsOf(tree.OverlapsSlice(s, e)), want)
					samePairs(t, "Overlaps", pairsOf(slices.Collect(tree.Overlaps(s, e))), want)

				case 5: // STABS (slice and iterator forms)
					want := ref.stabs(s)
					samePairs(t, "StabsSlice", pairsOf(tree.StabsSlice(s)), want)
					samePairs(t, "Stabs", pairsOf(slices.Collect(tree.Stabs(s))), want)

				case 6: // COMPUTE MERGE
					treeDel, treeMerged, treeIntersect := tree.ComputeMerge(s, e, struct{}{})
					refDel, refMerged, refIntersect := ref.computeMerge(s, e)

					if treeIntersect != refIntersect {
						t.Fatalf("ComputeMerge intersect mismatch at (%d, %d): Tree=%v, Ref=%v", s, e, treeIntersect, refIntersect)
					}
					if treeMerged.Start != refMerged[0] || treeMerged.End != refMerged[1] {
						t.Fatalf("ComputeMerge merged range mismatch at (%d, %d): Tree=[%d,%d], Ref=%v", s, e, treeMerged.Start, treeMerged.End, refMerged)
					}
					samePairs(t, "ComputeMerge toDelete", pairsOf(treeDel), refDel)

				case 7: // COMPUTE CONTIGUOUS MERGE
					treeDel, treeMerged, treeIntersect := tree.ComputeContiguousMerge(s, e, struct{}{}, prev, next)
					refDel, refMerged, refIntersect := ref.computeContiguousMerge(s, e)

					if treeIntersect != refIntersect {
						t.Fatalf("ComputeContiguousMerge intersect mismatch at (%d, %d): Tree=%v, Ref=%v", s, e, treeIntersect, refIntersect)
					}
					if treeMerged.Start != refMerged[0] || treeMerged.End != refMerged[1] {
						t.Fatalf("ComputeContiguousMerge merged range mismatch at (%d, %d): Tree=[%d,%d], Ref=%v", s, e, treeMerged.Start, treeMerged.End, refMerged)
					}
					samePairs(t, "ComputeContiguousMerge toDelete", pairsOf(treeDel), refDel)

				case 8: // MERGE ALL
					tree.MergeAll(next)
					ref.mergeAll()

				case 9: // COMPUTE SPLIT
					treeDel, treeAdd, treeIntersect := tree.ComputeSplit(s, e, prev, next)
					refDel, refAdd, refIntersect := ref.computeSplit(s, e)

					if treeIntersect != refIntersect {
						t.Fatalf("ComputeSplit intersect mismatch at (%d, %d): Tree=%v, Ref=%v", s, e, treeIntersect, refIntersect)
					}
					samePairs(t, "ComputeSplit toDelete", pairsOf(treeDel), refDel)
					samePairs(t, "ComputeSplit toAdd", pairsOf(treeAdd), refAdd)

				case 10: // GET
					_, ok := tree.Get(s, e)
					refOk := ref.exactMatch(s, e)
					if ok != refOk {
						t.Fatalf("Get(%d, %d) mismatch: Tree=%v, Reference=%v", s, e, ok, refOk)
					}

				case 11: // CLONE
					clone := tree.Clone()
					verifyTreeInvariants(t, clone)
					samePairsInOrder(t, "Clone", pairsOf(clone.AllSlice()), ref.all())

					// Changing the clone must not affect the original. The check
					// of the original against the model below catches shared state.
					clone.Reset()
					_ = clone.Insert(s, e, struct{}{})
					verifyTreeInvariants(t, clone)
				}
			}

			// CHECK ALL TREE INVARIANTS AND THE FULL CONTENT AFTER EVERY OPERATION
			verifyTreeInvariants(t, tree)
			verifyAgainstModel(t, tree, ref)
		}
	})
}

// verifyTreeInvariants validates the tree's AVL rules, maxEnd calculations, and parent-child indices.
func verifyTreeInvariants(t *testing.T, tree *IntervalTree[int, struct{}]) {
	const minInt = math.MinInt

	if tree.rootIdx == noIdx {
		activeCount := len(tree.nodes) - len(tree.freeIndex)
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

		node := tree.nodes[idx]

		// 1. Parent verification
		if node.parentIdx != expectedParent {
			t.Fatalf("Incorrect parentIdx for node %d. Expected: %d, Actual: %d", idx, expectedParent, node.parentIdx)
		}

		// 2. BST Ordering verification
		if node.leftIdx != noIdx {
			left := tree.nodes[node.leftIdx]
			if tree.cmpIntervals(left.Interval, node.Interval) >= 0 {
				t.Fatalf("BST rule violation: Left child [%d, %d] >= Root [%d, %d]", left.Start, left.End, node.Start, node.End)
			}
		}
		if node.rightIdx != noIdx {
			right := tree.nodes[node.rightIdx]
			if tree.cmpIntervals(right.Interval, node.Interval) <= 0 {
				t.Fatalf("BST rule violation: Right child [%d, %d] <= Root [%d, %d]", right.Start, right.End, node.Start, node.End)
			}
		}

		// Recursively validate subtrees
		leftH, leftMax := validateNode(node.leftIdx, idx)
		rightH, rightMax := validateNode(node.rightIdx, idx)

		// 3. AVL Height and Balance Verification
		expectedH := 1 + max(leftH, rightH)
		if node.height != expectedH {
			t.Fatalf("Node %d invalid height. Expected: %d, Got: %d", idx, expectedH, node.height)
		}

		balance := int(leftH) - int(rightH)
		if balance < -1 || balance > 1 {
			t.Fatalf("AVL balance factor violation! Node %d, Balance: %d", idx, balance)
		}

		// 4. maxEnd verification
		expectedMax := node.End
		if node.leftIdx != noIdx && leftMax > expectedMax {
			expectedMax = leftMax
		}
		if node.rightIdx != noIdx && rightMax > expectedMax {
			expectedMax = rightMax
		}
		if node.maxEnd != expectedMax {
			t.Fatalf("Node %d invalid maxEnd. Expected: %d, Got: %d", idx, expectedMax, node.maxEnd)
		}

		return node.height, node.maxEnd
	}

	validateNode(tree.rootIdx, noIdx)

	// 5. Memory Leak / Unreachable Node Check
	activeCount := len(tree.nodes) - len(tree.freeIndex)
	if len(visited) != activeCount {
		t.Fatalf("Active node count mismatch! Visited in tree: %d, Calculated (nodes - freeIndex): %d", len(visited), activeCount)
	}
}
