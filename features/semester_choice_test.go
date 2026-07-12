package features

import (
	"cli-top/types"
	"fmt"
	"testing"
)

func TestSemesterCandidatesAutomaticNewestFirst(t *testing.T) {
	semesters := []types.Semester{
		{SemID: "oldest", SemName: "Oldest"},
		{SemID: "middle", SemName: "Middle"},
		{SemID: "newest", SemName: "Newest"},
	}

	got, err := semesterCandidates(semesters, 0)
	if err != nil {
		t.Fatalf("semesterCandidates returned an error: %v", err)
	}

	wantIDs := []string{"newest", "middle", "oldest"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d candidates, want %d", len(got), len(wantIDs))
	}
	for i, wantID := range wantIDs {
		if got[i].SemID != wantID {
			t.Fatalf("candidate %d has ID %q, want %q", i, got[i].SemID, wantID)
		}
	}
}

func TestSemesterCandidatesPositiveChoiceSelectsOnlyDisplayedSemester(t *testing.T) {
	semesters := []types.Semester{
		{SemID: "first", SemName: "First displayed"},
		{SemID: "second", SemName: "Second displayed"},
		{SemID: "third", SemName: "Third displayed"},
	}

	got, err := semesterCandidates(semesters, 2)
	if err != nil {
		t.Fatalf("semesterCandidates returned an error: %v", err)
	}
	if len(got) != 1 || got[0].SemID != "second" {
		t.Fatalf("got %#v, want only the second displayed semester", got)
	}
}

func TestSemesterCandidatesRejectsInvalidChoice(t *testing.T) {
	semesters := []types.Semester{{SemID: "only", SemName: "Only"}}

	for _, choice := range []int{-1, 2} {
		t.Run(fmt.Sprintf("choice_%d", choice), func(t *testing.T) {
			if _, err := semesterCandidates(semesters, choice); err == nil {
				t.Fatalf("expected choice %d to be rejected", choice)
			}
		})
	}
}
