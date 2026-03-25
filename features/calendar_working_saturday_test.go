package features

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractTypeOfDayRecognizesLowercaseDayOrder(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<table>
			<tr>
				<td>
					<span>28</span>
					<span>Instructional Day</span>
					<span>(Friday Day order)</span>
				</td>
			</tr>
		</table>
	`))
	if err != nil {
		t.Fatalf("failed to build document: %v", err)
	}

	arr := make([]int, 31)
	typeOfDay := extractTypeOfDay(doc, arr)

	if len(typeOfDay) != 1 || typeOfDay[0] != 3 {
		t.Fatalf("expected instructional-day color classification, got %v", typeOfDay)
	}

	if got := arr[0]; got != int(time.Friday) {
		t.Fatalf("expected the parsed date to be Friday day order, got %d", got)
	}
}

func TestWorkingSaturdaysFromSemSection(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	semSection := [][]int{make([]int, 31)}
	semSection[0][27] = int(time.Friday)

	got := workingSaturdaysFromSemSection(semSection, 2, 2026, loc)
	if len(got) != 1 {
		t.Fatalf("expected 1 working Saturday, got %d", len(got))
	}

	if got[0].DayOrder != "Friday" {
		t.Fatalf("expected Friday day order, got %q", got[0].DayOrder)
	}

	if got[0].Date.Format("2006-01-02") != "2026-03-28" {
		t.Fatalf("expected working Saturday on 2026-03-28, got %s", got[0].Date.Format("2006-01-02"))
	}
}
