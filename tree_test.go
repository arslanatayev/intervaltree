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
	"math/rand"
	"testing"
)

func TestInsertReplacesValue(t *testing.T) {
	tree := NewOrdered[int, string](4)

	if !tree.Insert(1, 5, "a") {
		t.Fatal("first Insert should report a new interval")
	}
	if tree.Insert(1, 5, "b") {
		t.Fatal("second Insert with the same bounds should report false")
	}
	if got, ok := tree.Get(1, 5); !ok || got != "b" {
		t.Fatalf("Get(1, 5) = %q, %v; want \"b\", true", got, ok)
	}
	if tree.Len() != 1 {
		t.Fatalf("Len() = %d; want 1", tree.Len())
	}
}

func TestMergeAllKeepsValueOfGreatestEnd(t *testing.T) {
	next := func(a int) int { return a + 1 }

	tests := []struct {
		name  string
		input []Interval[int, string]
		want  Interval[int, string]
	}{
		{
			name:  "later interval extends the group",
			input: []Interval[int, string]{{1, 5, "a"}, {3, 10, "b"}, {4, 6, "c"}},
			want:  Interval[int, string]{1, 10, "b"},
		},
		{
			name:  "tie on End keeps the first value",
			input: []Interval[int, string]{{1, 5, "a"}, {2, 5, "b"}},
			want:  Interval[int, string]{1, 5, "a"},
		},
		{
			name:  "contiguous intervals are merged",
			input: []Interval[int, string]{{1, 5, "a"}, {6, 9, "b"}},
			want:  Interval[int, string]{1, 9, "b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tree := NewOrdered[int, string](8)
			for _, iv := range tc.input {
				tree.Insert(iv.Start, iv.End, iv.Value)
			}

			tree.MergeAll(next)

			got := tree.AllSlice()
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("MergeAll result = %v; want [%v]", got, tc.want)
			}
		})
	}
}

// TestLargeTree checks the tree invariants and query results at one million intervals.
func TestLargeTree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large tree test in short mode")
	}

	const n = 1_000_000
	intervals := genIntervals(n, 1000)
	tree := defaultTree(n)

	for _, iv := range intervals {
		if !tree.Insert(iv[0], iv[1], struct{}{}) {
			t.Fatalf("Insert(%d, %d) reported a duplicate", iv[0], iv[1])
		}
	}
	if tree.Len() != n {
		t.Fatalf("Len() = %d; want %d", tree.Len(), n)
	}
	verifyTreeInvariants(t, tree)

	// Delete every other interval and keep the rest as the expected content.
	remaining := make([][2]int, 0, n/2)
	for i, iv := range intervals {
		if i%2 == 0 {
			if !tree.Delete(iv[0], iv[1]) {
				t.Fatalf("Delete(%d, %d) reported a missing interval", iv[0], iv[1])
			}
		} else {
			remaining = append(remaining, iv)
		}
	}
	if tree.Len() != len(remaining) {
		t.Fatalf("Len() = %d; want %d", tree.Len(), len(remaining))
	}
	verifyTreeInvariants(t, tree)

	// Compare random overlap queries with a brute-force scan.
	rng := rand.New(rand.NewSource(7))
	for q := 0; q < 200; q++ {
		s := rng.Intn(1_000_000)
		e := s + rng.Intn(1000)

		var want [][2]int
		for _, iv := range remaining {
			if iv[0] <= e && s <= iv[1] {
				want = append(want, iv)
			}
		}
		samePairs(t, "OverlapsSlice", pairsOf(tree.OverlapsSlice(s, e)), want)
	}
}
