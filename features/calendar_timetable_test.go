package features

import (
	"cli-top/types"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractClassGroupKeepsSingleCombinedOption(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<select id="classGroupId">
			<option value="COMB" selected="selected">All Class Group (Combined)</option>
		</select>
	`))
	if err != nil {
		t.Fatalf("failed to build document: %v", err)
	}

	got := extractclassgrp(doc)
	if len(got) != 1 {
		t.Fatalf("expected one class group, got %d: %#v", len(got), got)
	}
	if got[0][0] != "All Class Group (Combined)" || got[0][1] != "COMB" {
		t.Fatalf("unexpected class group: %#v", got[0])
	}
}

func TestReadMonthsIgnoresMalformedOnclick(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<a class="btn btn-md btn-primary" onclick="javascript:processViewCalendar('01-MAY-2026');">MAY-2026</a>
		<a class="btn btn-md btn-primary" onclick="javascript:processViewCalendar();">bad</a>
	`))
	if err != nil {
		t.Fatalf("failed to build document: %v", err)
	}

	got := readmonths(doc)
	if len(got) != 1 || got[0] != "01-MAY-2026" {
		t.Fatalf("expected only the valid month, got %#v", got)
	}
}

func TestProcessDatesHandlesEmptyDateList(t *testing.T) {
	semSection, month, year := processDates("", types.Cookies{}, types.Semester{}, "COMB", nil, 0)
	if semSection != nil || month != -1 || year != -1 {
		t.Fatalf("expected empty calendar result, got semSection=%#v month=%d year=%d", semSection, month, year)
	}
}

func TestResolveTimetableCalendarClassGroupPrefersCombined(t *testing.T) {
	got, ok := resolveTimetableCalendarClassGroupID([][]string{
		{"General (Semester)", "ALL"},
		{"All Class Group (Combined)", "COMB"},
	})
	if !ok || got != "COMB" {
		t.Fatalf("expected COMB, got %q ok=%v", got, ok)
	}
}

func TestExamCategoryActiveOnlyWithinCategoryWindow(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	exams := []types.ExamEvent{
		{Category: "CAT2", ExamDate: time.Date(2026, time.March, 15, 0, 0, 0, 0, loc)},
		{Category: "FAT", ExamDate: time.Date(2026, time.April, 22, 0, 0, 0, 0, loc)},
		{Category: "FAT", ExamDate: time.Date(2026, time.May, 13, 0, 0, 0, 0, loc)},
	}

	if !examCategoryActive(exams, "FAT", time.Date(2026, time.April, 23, 12, 0, 0, 0, loc)) {
		t.Fatal("expected FAT window to be active")
	}
	if examCategoryActive(exams, "CAT2", time.Date(2026, time.April, 23, 12, 0, 0, 0, loc)) {
		t.Fatal("did not expect CAT2 window to be active")
	}
	if examCategoryActive(exams, "FAT", time.Date(2026, time.May, 14, 12, 0, 0, 0, loc)) {
		t.Fatal("did not expect FAT window after the last FAT exam")
	}
}
