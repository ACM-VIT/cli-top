package tests

import (
	featurespkg "cli-top/features"
	types "cli-top/types"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestParseCalendarEntryCapturesHolidayReasonAndDayOrderCaseInsensitive(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<table class="calendar-table">
			<tr>
				<td>
					<span>28</span>
					<span>No Instructional Day</span>
					<span>(Friday Day order)</span>
					<span>(Festival)</span>
				</td>
			</tr>
		</table>
	`))
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	loc := time.FixedZone("IST", 5*3600+1800)
	entry, ok := featurespkg.ParseCalendarEntry(doc.Find("td").First(), time.March, 2026, loc)
	if !ok {
		t.Fatal("expected calendar entry to be parsed")
	}

	if entry.Status != "No Instructional Day" {
		t.Fatalf("expected no instructional day, got %q", entry.Status)
	}

	if entry.DayOrder != "Friday" {
		t.Fatalf("expected friday day order, got %q", entry.DayOrder)
	}

	if entry.Reason != "Festival" {
		t.Fatalf("expected Festival reason, got %q", entry.Reason)
	}
}

func TestFilterUpcomingClassImpactingHolidaysUsesScheduleAndSkipsIrrelevantWeekend(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	now := time.Date(2026, time.March, 1, 9, 0, 0, 0, loc)

	entries := []featurespkg.CalendarDayEntry{
		{
			Date:   time.Date(2026, time.March, 2, 0, 0, 0, 0, loc),
			Status: "Holiday",
			Reason: "Monday Holiday",
		},
		{
			Date:   time.Date(2026, time.March, 7, 0, 0, 0, 0, loc),
			Status: "Holiday",
			Reason: "Saturday Holiday",
		},
		{
			Date:     time.Date(2026, time.March, 14, 0, 0, 0, 0, loc),
			Status:   "No Instructional Day",
			Reason:   "Working Saturday Off",
			DayOrder: "Friday",
		},
	}

	timetable := map[string][]types.Class{
		"Monday": {
			{Subject: "Engineering Optimization"},
		},
		"Friday": {
			{Subject: "Control Systems"},
		},
	}

	got := featurespkg.FilterUpcomingClassImpactingHolidays(entries, timetable, now)
	if len(got) != 2 {
		t.Fatalf("expected 2 relevant holidays, got %d", len(got))
	}

	if got[0].Reason != "Monday Holiday" || got[0].Schedule != "Monday" {
		t.Fatalf("unexpected first holiday: %+v", got[0])
	}

	if got[1].Reason != "Working Saturday Off" || got[1].Schedule != "Friday" {
		t.Fatalf("unexpected second holiday: %+v", got[1])
	}
}

func TestResolveHolidaySemesterDefaultsToLatest(t *testing.T) {
	semesters := []types.Semester{
		{SemID: "VL20242501", SemName: "Fall Semester 2024-25 - VLR"},
		{SemID: "VL20242505", SemName: "Winter Semester 2024-25 - VLR"},
		{SemID: "VL20252605", SemName: "Winter Semester 2025-26 - VLR"},
	}

	got := featurespkg.LatestSemester(semesters)
	if got.SemID != "VL20252605" {
		t.Fatalf("expected latest semester to be selected, got %q", got.SemID)
	}
}
