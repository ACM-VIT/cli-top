package tests

import (
	appfeatures "cli-top/features"
	"cli-top/internal/timetableutil"
	"cli-top/types"
	"testing"
	"time"
)

func TestShouldPromptForTimetableSemesterSelection(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "prompts in june",
			now:  time.Date(2026, time.June, 10, 9, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "prompts in july",
			now:  time.Date(2026, time.July, 10, 9, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "prompts in december",
			now:  time.Date(2026, time.December, 10, 9, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "skips prompt outside rollover months",
			now:  time.Date(2026, time.March, 10, 9, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timetableutil.ShouldPromptSemesterSelection(tt.now); got != tt.want {
				t.Fatalf("ShouldPromptSemesterSelection(%v) = %v, want %v", tt.now.Month(), got, tt.want)
			}
		})
	}
}

func TestLatestSemesterReturnsNewestSemesterForTimetableDefault(t *testing.T) {
	semesters := []types.Semester{
		{SemID: "VL20242501", SemName: "Fall Semester 2024-25 - VLR"},
		{SemID: "VL20242505", SemName: "Winter Semester 2024-25 - VLR"},
		{SemID: "VL20252601", SemName: "Fall Semester 2025-26 - VLR"},
	}

	got := appfeatures.LatestSemester(semesters)
	if got.SemID != "VL20252601" {
		t.Fatalf("expected latest semester to be selected, got %q", got.SemID)
	}
}
