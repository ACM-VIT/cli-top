package tests

import (
	featurespkg "cli-top/features"
	types "cli-top/types"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractAttendanceRecordsUsesLabSessionsForLeverage(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<table id="AttendanceDetailDataTable">
			<tbody>
				<tr>
					<td></td>
					<td></td>
					<td><span>ECE3001 - Microcontrollers and Interfacing - Embedded Lab</span></td>
					<td></td>
					<td><span>DOE - JANE</span></td>
					<td><span>8</span></td>
					<td><span>10</span></td>
					<td><span><span>80</span></span></td>
				</tr>
			</tbody>
		</table>
	`))
	if err != nil {
		t.Fatalf("failed to build attendance HTML: %v", err)
	}

	records := featurespkg.ExtractAttendanceRecords(doc)
	if len(records) != 1 {
		t.Fatalf("expected 1 attendance record, got %d", len(records))
	}

	if !records[0].IsLab {
		t.Fatal("expected record to be marked as lab")
	}

	if records[0].CanMiss != 0 {
		t.Fatalf("expected lab leverage to be computed in session units, got %d", records[0].CanMiss)
	}
}

func TestBuildTomorrowClassAdvicesConsumesSkipBudgetPerOccurrence(t *testing.T) {
	classes := []types.Class{
		{Subject: "Engineering Optimization", Slot: "A2", StartTime: "14:00", EndTime: "14:50"},
		{Subject: "Engineering Optimization", Slot: "TA2", StartTime: "16:00", EndTime: "16:50"},
		{Subject: "Industrial Robotics( Embedded Lab )", Slot: "L29+L30", StartTime: "11:40", EndTime: "13:20"},
	}

	attendance := []featurespkg.AttendanceRecord{
		{
			Subject:    "Engineering Optimization",
			Percentage: "82",
			Alert:      "Can miss 1 class(es)",
			CanMiss:    1,
		},
		{
			Subject:    "Industrial Robotics",
			Percentage: "91",
			Alert:      "Can miss 2 lab(s)",
			CanMiss:    2,
			IsLab:      true,
		},
	}

	advices := featurespkg.BuildTomorrowClassAdvices(classes, attendance)
	if len(advices) != 3 {
		t.Fatalf("expected 3 class advices, got %d", len(advices))
	}

	if advices[0].Class.Slot != "L29+L30" || advices[0].Skip != "Yes" {
		t.Fatalf("expected first advice to be the lab and skippable, got %+v", advices[0])
	}

	if advices[1].Class.Slot != "A2" || advices[1].Skip != "Yes" {
		t.Fatalf("expected second advice to use the single theory skip budget, got %+v", advices[1])
	}

	if advices[2].Class.Slot != "TA2" || advices[2].Skip != "No" {
		t.Fatalf("expected final theory class to consume the remaining theory budget, got %+v", advices[2])
	}

	if advices[0].Attendance != "91%" || advices[1].Attendance != "82%" {
		t.Fatalf("expected attendance percentages to be carried into the advice rows, got %+v", advices)
	}
}

func TestResolvePlanningDateForToday(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	now := time.Date(2026, time.March, 26, 3, 0, 0, 0, loc)

	date := featurespkg.ResolvePlanningDate(now, 0)
	want := time.Date(2026, time.March, 26, 0, 0, 0, 0, loc)
	if !date.Equal(want) {
		t.Fatalf("expected %v, got %v", want, date)
	}
}

func TestResolvePlanningDateForTomorrow(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	now := time.Date(2026, time.March, 26, 8, 0, 0, 0, loc)

	date := featurespkg.ResolvePlanningDate(now, 1)
	want := time.Date(2026, time.March, 27, 0, 0, 0, 0, loc)
	if !date.Equal(want) {
		t.Fatalf("expected %v, got %v", want, date)
	}
}

func TestResolvePlanningDateForDayAfterTomorrow(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	now := time.Date(2026, time.March, 26, 8, 0, 0, 0, loc)

	date := featurespkg.ResolvePlanningDate(now, 2)
	want := time.Date(2026, time.March, 28, 0, 0, 0, 0, loc)
	if !date.Equal(want) {
		t.Fatalf("expected %v, got %v", want, date)
	}
}

func TestFilterExamsForDateMatchesExactDay(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	target := time.Date(2026, time.March, 27, 0, 0, 0, 0, loc)
	exams := []types.ExamEvent{
		{
			CourseTitle: "Engineering Optimization",
			Category:    "CAT2",
			ExamDate:    time.Date(2026, time.March, 27, 0, 0, 0, 0, loc),
			ExamTime:    "2:00 PM - 3:30 PM",
			Venue:       "GDN122",
		},
		{
			CourseTitle: "Control Systems",
			Category:    "CAT2",
			ExamDate:    time.Date(2026, time.March, 28, 0, 0, 0, 0, loc),
			ExamTime:    "9:00 AM - 10:30 AM",
			Venue:       "GDN122",
		},
	}

	filtered := featurespkg.FilterExamsForDate(exams, target)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 exam on target date, got %d", len(filtered))
	}

	if filtered[0].CourseTitle != "Engineering Optimization" {
		t.Fatalf("unexpected exam returned: %+v", filtered[0])
	}
}

func TestFilterExamsForDateIgnoresTimezoneOnDateOnlyExams(t *testing.T) {
	ist := time.FixedZone("IST", 5*3600+1800)
	nzdt := time.FixedZone("NZDT", 13*3600)
	target := time.Date(2026, time.March, 27, 0, 0, 0, 0, ist)

	exams := []types.ExamEvent{
		{
			CourseTitle: "Engineering Optimization",
			Category:    "CAT2",
			ExamDate:    time.Date(2026, time.March, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			CourseTitle: "Signal Processing",
			Category:    "CAT2",
			ExamDate:    time.Date(2026, time.March, 27, 0, 0, 0, 0, nzdt),
		},
		{
			CourseTitle: "Control Systems",
			Category:    "CAT2",
			ExamDate:    time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC),
		},
	}

	filtered := featurespkg.FilterExamsForDate(exams, target)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 exams on target date, got %d", len(filtered))
	}

	if filtered[0].CourseTitle != "Engineering Optimization" || filtered[1].CourseTitle != "Signal Processing" {
		t.Fatalf("unexpected exams returned: %+v", filtered)
	}
}
