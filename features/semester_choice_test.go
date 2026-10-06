package features

import (
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
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

func TestChooseCourseSemesterDefaultsToLatest(t *testing.T) {
	t.Setenv("AUTO_SELECT_SEMESTER", "true")
	t.Setenv("CLI_TOP_PROXY_MODE", "")
	semesters := []string{"Fall Semester 2025-26", "Winter Semester 2025-26"}

	got, err := chooseCourseSemester(semesters, 0)
	if err != nil {
		t.Fatalf("chooseCourseSemester returned an error: %v", err)
	}
	if got != "Winter Semester 2025-26" {
		t.Fatalf("got %q, want latest semester", got)
	}
}

func TestChooseCourseSemesterExplicitAndInvalidChoices(t *testing.T) {
	t.Setenv("AUTO_SELECT_SEMESTER", "true")
	semesters := []string{"Fall Semester 2025-26", "Winter Semester 2025-26"}

	got, err := chooseCourseSemester(semesters, 1)
	if err != nil || got != "Fall Semester 2025-26" {
		t.Fatalf("chooseCourseSemester(1) = %q, %v; want first semester", got, err)
	}
	if _, err := chooseCourseSemester(semesters, -1); err == nil {
		t.Fatal("expected a negative choice to be rejected")
	}
	if _, err := chooseCourseSemester(nil, 0); err == nil {
		t.Fatal("expected an empty semester list to be rejected")
	}
}

func TestChooseCourseSemesterListsWhenRequested(t *testing.T) {
	semesters := []string{"Fall Semester 2025-26", "Winter Semester 2025-26"}

	for _, tc := range []struct {
		name string
		env  string
		list bool
	}{
		{name: "list flag", env: "true", list: true},
		{name: "auto-select off", env: "false", list: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTO_SELECT_SEMESTER", tc.env)
			t.Setenv("CLI_TOP_PROXY_MODE", "")
			helpers.ListSemesters = tc.list
			t.Cleanup(func() { helpers.ListSemesters = false })
			withStdin(t, "1\n")

			got, err := chooseCourseSemester(semesters, 0)
			if err != nil {
				t.Fatalf("chooseCourseSemester returned an error: %v", err)
			}
			if got != "Fall Semester 2025-26" {
				t.Fatalf("got %q, want the semester picked from the list", got)
			}
		})
	}
}

func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	w.Close()
	original := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = original
		r.Close()
	})
}
