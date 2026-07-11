package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"cli-top/features"
	"cli-top/helpers"
	"cli-top/types"
)

const (
	plannerSemesterID = "SEM-PLANNER-E2E"
	plannerClassGroup = "COMB"
)

var plannerFeatureCookies = types.Cookies{
	SERVERID:   "planner-fake-server",
	CSRF:       "planner-fake-csrf",
	JSESSIONID: "planner-fake-session",
}

type plannerFeatureRequest struct {
	path string
	form url.Values
}

type plannerFeatureTransport struct {
	regNo     string
	responses map[string]func(url.Values) string

	mu       sync.Mutex
	requests []plannerFeatureRequest
	errors   []string
}

func (transport *plannerFeatureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s request body: %w", req.URL.Path, err)
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse %s request body: %w", req.URL.Path, err)
	}

	transport.mu.Lock()
	transport.requests = append(transport.requests, plannerFeatureRequest{path: req.URL.Path, form: form})
	if req.Method != http.MethodPost {
		transport.errors = append(transport.errors, fmt.Sprintf("%s used method %s", req.URL.Path, req.Method))
	}
	if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		transport.errors = append(transport.errors, fmt.Sprintf("%s used content type %q", req.URL.Path, got))
	}
	cookie := req.Header.Get("Cookie")
	if !strings.Contains(cookie, "SERVERID="+plannerFeatureCookies.SERVERID) ||
		!strings.Contains(cookie, "JSESSIONID="+plannerFeatureCookies.JSESSIONID) {
		transport.errors = append(transport.errors, fmt.Sprintf("%s used cookie %q", req.URL.Path, cookie))
	}
	if got := form.Get("authorizedID"); got != transport.regNo {
		transport.errors = append(transport.errors, fmt.Sprintf("%s authorizedID = %q", req.URL.Path, got))
	}
	if got := form.Get("_csrf"); got != plannerFeatureCookies.CSRF {
		transport.errors = append(transport.errors, fmt.Sprintf("%s _csrf = %q", req.URL.Path, got))
	}
	handler := transport.responses[req.URL.Path]
	transport.mu.Unlock()

	if handler == nil {
		return nil, fmt.Errorf("unexpected request path %q", req.URL.Path)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(handler(form))),
		Request:    req,
	}, nil
}

func TestPlannerFeaturesEndToEnd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	t.Setenv("SEMESTER_CACHE", "")

	t.Run("GetTimeTable", func(t *testing.T) {
		regNo := "E2E-PLANNER-TIMETABLE"
		locIndia := time.FixedZone("IST", 5*60*60+30*60)
		now := time.Now().In(locIndia)
		daysUntilSaturday := (int(time.Saturday) - int(now.Weekday()) + 7) % 7
		workingSaturday := now.AddDate(0, 0, daysUntilSaturday)
		calendarDate := plannerCalendarDate(workingSaturday)
		calendarHTML := plannerCalendarHTML(workingSaturday.Year(), workingSaturday.Month(), map[int][]string{
			workingSaturday.Day(): {"Instructional Day", "Monday Day Order"},
		})

		transport, tables, _ := runPlannerFeature(t, regNo, map[string]func(url.Values) string{
			"/vtop/academics/common/StudentAttendance": plannerStaticResponse(plannerSemesterHTML()),
			"/vtop/processViewTimeTable":               plannerStaticResponse(plannerTimetableHTML("A1")),
			"/vtop/getDateForSemesterPreview":          plannerStaticResponse(plannerClassGroupHTML()),
			"/vtop/getListForSemester":                 plannerStaticResponse(plannerDateListHTML(calendarDate)),
			"/vtop/processViewCalendar":                plannerStaticResponse(calendarHTML),
		}, func() {
			features.GetTimeTable(regNo, plannerFeatureCookies, 1)
		})

		assertPlannerPathCounts(t, transport, map[string]int{
			"/vtop/academics/common/StudentAttendance": 1,
			"/vtop/processViewTimeTable":               1,
			"/vtop/getDateForSemesterPreview":          1,
			"/vtop/getListForSemester":                 1,
			"/vtop/processViewCalendar":                1,
		})
		assertPlannerPostedValue(t, transport, "/vtop/processViewTimeTable", "semesterSubId", plannerSemesterID)
		assertPlannerPostedValue(t, transport, "/vtop/processViewCalendar", "calDate", calendarDate)
		assertPlannerTableContains(t, tables, []string{"TIME", "SUBJECT", "SLOT", "VENUE"}, []string{"08:00-08:50", "Algorithms", "A1", "SJT-101"})
	})

	t.Run("PrintCal", func(t *testing.T) {
		regNo := "E2E-PLANNER-CALENDAR"
		calendarDate := "01-JAN-2030"
		transport, tables, output := runPlannerFeature(t, regNo, map[string]func(url.Values) string{
			"/vtop/academics/common/StudentAttendance": plannerStaticResponse(plannerSemesterHTML()),
			"/vtop/getDateForSemesterPreview":          plannerStaticResponse(plannerClassGroupHTML()),
			"/vtop/getListForSemester":                 plannerStaticResponse(plannerDateListHTML(calendarDate)),
			"/vtop/processViewCalendar": plannerStaticResponse(plannerCalendarHTML(2030, time.January, map[int][]string{
				1: {"Holiday", "New Year's Day"},
				2: {"Exam Day"},
			})),
		}, func() {
			features.PrintCal(regNo, plannerFeatureCookies, 1, 1)
		})

		assertPlannerPathCounts(t, transport, map[string]int{
			"/vtop/academics/common/StudentAttendance": 1,
			"/vtop/getDateForSemesterPreview":          1,
			"/vtop/getListForSemester":                 1,
			"/vtop/processViewCalendar":                1,
		})
		assertPlannerPostedValue(t, transport, "/vtop/getListForSemester", "classGroupId", plannerClassGroup)
		if len(tables) != 0 {
			t.Fatalf("PrintCal captured unexpected tables: %#v", tables)
		}
		plainOutput := helpers.StripAnsiCodes(output)
		for _, want := range []string{"Your selected class group: Combined", "Red-Exam Day", "Blue-Holiday", "2030", "JAN", "Su Mo Tu We Th Fr Sa"} {
			if !strings.Contains(plainOutput, want) {
				t.Errorf("PrintCal output = %q, want %q", plainOutput, want)
			}
		}
	})

	t.Run("GetHolidayList", func(t *testing.T) {
		regNo := "E2E-PLANNER-HOLIDAYS"
		holiday := time.Date(2099, time.December, 31, 0, 0, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
		calendarDate := plannerCalendarDate(holiday)
		transport, tables, _ := runPlannerFeature(t, regNo, map[string]func(url.Values) string{
			"/vtop/academics/common/StudentAttendance": plannerStaticResponse(plannerSemesterHTML()),
			"/vtop/getDateForSemesterPreview":          plannerStaticResponse(plannerClassGroupHTML()),
			"/vtop/getListForSemester":                 plannerStaticResponse(plannerDateListHTML(calendarDate)),
			"/vtop/processViewCalendar": plannerStaticResponse(plannerCalendarHTML(holiday.Year(), holiday.Month(), map[int][]string{
				holiday.Day(): {"Holiday", "Founders Day"},
			})),
			"/vtop/processViewTimeTable": plannerStaticResponse(plannerTimetableHTML("B1")),
		}, func() {
			features.GetHolidayList(regNo, plannerFeatureCookies, 1, 1)
		})

		assertPlannerPathCounts(t, transport, map[string]int{
			"/vtop/academics/common/StudentAttendance": 1,
			"/vtop/getDateForSemesterPreview":          1,
			"/vtop/getListForSemester":                 1,
			"/vtop/processViewCalendar":                1,
			"/vtop/processViewTimeTable":               1,
		})
		if len(tables) != 1 {
			t.Fatalf("holiday table count = %d, want 1: %#v", len(tables), tables)
		}
		if !reflect.DeepEqual(tables[0].Headers, []string{"IN", "DATE", "DAY", "TYPE", "REASON"}) || len(tables[0].Rows) != 1 {
			t.Fatalf("unexpected holiday table: %#v", tables[0])
		}
		wantSuffix := []string{"31 Dec 2099", "Thursday", "Holiday", "Founders Day"}
		if !reflect.DeepEqual(tables[0].Rows[0][1:], wantSuffix) {
			t.Fatalf("holiday row = %#v, want suffix %#v", tables[0].Rows[0], wantSuffix)
		}
	})

	planningCases := []struct {
		name   string
		label  string
		offset int
		run    func(string)
	}{
		{name: "GetToday", label: "Today", offset: 0, run: func(regNo string) { features.GetToday(regNo, plannerFeatureCookies, 1, 1) }},
		{name: "GetTomorrow", label: "Tomorrow", offset: 1, run: func(regNo string) { features.GetTomorrow(regNo, plannerFeatureCookies, 1, 1) }},
		{name: "GetDayAfter", label: "Day After Tomorrow", offset: 2, run: func(regNo string) { features.GetDayAfter(regNo, plannerFeatureCookies, 1, 1) }},
	}

	for index, testCase := range planningCases {
		t.Run(testCase.name, func(t *testing.T) {
			regNo := fmt.Sprintf("E2E-PLANNER-DAY-%d", index)
			locIndia := time.FixedZone("IST", 5*60*60+30*60)
			target := features.ResolvePlanningDate(time.Now().In(locIndia), testCase.offset)
			calendarDate := plannerCalendarDate(target)
			transport, tables, output := runPlannerFeature(t, regNo, map[string]func(url.Values) string{
				"/vtop/academics/common/StudentAttendance": plannerStaticResponse(plannerSemesterHTML()),
				"/vtop/getDateForSemesterPreview":          plannerStaticResponse(plannerClassGroupHTML()),
				"/vtop/getListForSemester":                 plannerStaticResponse(plannerDateListHTML(calendarDate)),
				"/vtop/processViewCalendar": plannerStaticResponse(plannerCalendarHTML(target.Year(), target.Month(), map[int][]string{
					target.Day(): {"Instructional Day", "Monday Day Order", "E2E planning schedule"},
				})),
				"/vtop/examinations/doSearchExamScheduleForStudent": plannerStaticResponse(plannerExamHTML(target)),
				"/vtop/processViewTimeTable":                        plannerStaticResponse(plannerTimetableHTML("A1")),
				"/vtop/processViewStudentAttendance":                plannerStaticResponse(plannerAttendanceHTML()),
			}, func() {
				testCase.run(regNo)
			})

			assertPlannerPathCounts(t, transport, map[string]int{
				"/vtop/academics/common/StudentAttendance":          1,
				"/vtop/getDateForSemesterPreview":                   1,
				"/vtop/getListForSemester":                          1,
				"/vtop/processViewCalendar":                         1,
				"/vtop/examinations/doSearchExamScheduleForStudent": 1,
				"/vtop/processViewTimeTable":                        1,
				"/vtop/processViewStudentAttendance":                1,
			})
			assertPlannerPostedValue(t, transport, "/vtop/processViewStudentAttendance", "semesterSubId", plannerSemesterID)
			assertPlannerTableContains(t, tables, []string{"COURSE", "CATEGORY", "TIME", "VENUE"}, []string{"Algorithms", "CAT1", "09:00", "SJT-201"})
			assertPlannerTableContains(t, tables, []string{"TIME", "SUBJECT", "VENUE", "%", "LEVERAGE", "SKIP?"}, []string{"08:00-08:50", "Algorithms", "SJT-101", "90%", "Can miss 2 class(es)", "Yes"})

			plainOutput := helpers.StripAnsiCodes(output)
			for _, want := range []string{
				testCase.label + ": " + target.Format("Monday, 02 Jan 2006"),
				"Calendar: E2E planning schedule",
				fmt.Sprintf("Skippable %s: 1/1 classes.", strings.ToLower(testCase.label)),
			} {
				if !strings.Contains(plainOutput, want) {
					t.Errorf("%s output = %q, want %q", testCase.name, plainOutput, want)
				}
			}
		})
	}

	err := filepath.WalkDir(home, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".ics") {
			return fmt.Errorf("unexpected ICS file %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runPlannerFeature(t *testing.T, regNo string, responses map[string]func(url.Values) string, invoke func()) (*plannerFeatureTransport, []helpers.TableSnapshot, string) {
	t.Helper()

	helpers.InvalidateSemesterCache(regNo)
	t.Cleanup(func() { helpers.InvalidateSemesterCache(regNo) })

	transport := &plannerFeatureTransport{regNo: regNo, responses: responses}
	client := helpers.GetHTTPClient()
	previousTransport := client.Transport
	client.Transport = transport
	t.Cleanup(func() { client.Transport = previousTransport })

	var tables []helpers.TableSnapshot
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
	})
	t.Cleanup(restoreCapture)

	output := capturePlannerFeatureOutput(t, invoke)
	transport.mu.Lock()
	defer transport.mu.Unlock()
	if len(transport.errors) > 0 {
		t.Fatalf("request validation errors: %s", strings.Join(transport.errors, "; "))
	}
	return transport, tables, output
}

func assertPlannerPathCounts(t *testing.T, transport *plannerFeatureTransport, want map[string]int) {
	t.Helper()
	transport.mu.Lock()
	defer transport.mu.Unlock()

	got := make(map[string]int)
	for _, request := range transport.requests {
		got[request.path]++
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request path counts = %#v, want %#v", got, want)
	}
}

func assertPlannerPostedValue(t *testing.T, transport *plannerFeatureTransport, path, key, want string) {
	t.Helper()
	transport.mu.Lock()
	defer transport.mu.Unlock()

	for _, request := range transport.requests {
		if request.path == path {
			if got := request.form.Get(key); got != want {
				t.Fatalf("%s %s = %q, want %q", path, key, got, want)
			}
			return
		}
	}
	t.Fatalf("no request captured for %s", path)
}

func assertPlannerTableContains(t *testing.T, tables []helpers.TableSnapshot, headers, row []string) {
	t.Helper()
	for _, table := range tables {
		if !reflect.DeepEqual(table.Headers, headers) {
			continue
		}
		for _, candidate := range table.Rows {
			if reflect.DeepEqual(candidate, row) {
				return
			}
		}
	}
	t.Fatalf("captured tables %#v do not contain headers %#v and row %#v", tables, headers, row)
}

func plannerStaticResponse(body string) func(url.Values) string {
	return func(url.Values) string { return body }
}

func plannerSemesterHTML() string {
	return `<select id="semesterSubId"><option value="` + plannerSemesterID + `">Fall Semester 2026-27</option></select>`
}

func plannerClassGroupHTML() string {
	return `<select id="classGroupId"><option value="` + plannerClassGroup + `">Combined</option></select>`
}

func plannerDateListHTML(calendarDate string) string {
	return `<a class="btn btn-md btn-primary" onclick="load('` + calendarDate + `')">Month</a>`
}

func plannerCalendarDate(date time.Time) string {
	return strings.ToUpper(date.Format("01-Jan-2006"))
}

func plannerCalendarHTML(year int, month time.Month, overrides map[int][]string) string {
	var html strings.Builder
	html.WriteString(`<table class="calendar-table"><tbody><tr>`)
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	for day := 1; day <= daysInMonth; day++ {
		labels := overrides[day]
		if len(labels) == 0 {
			labels = []string{"Instructional Day"}
		}
		fmt.Fprintf(&html, "<td><span>%d</span>", day)
		for _, label := range labels {
			fmt.Fprintf(&html, "<span>%s</span>", label)
		}
		html.WriteString("</td>")
	}
	html.WriteString(`</tr></tbody></table>`)
	return html.String()
}

func plannerTimetableHTML(slot string) string {
	return `<table class="table"><tbody><tr>` +
		`<td>1</td><td>-</td><td>CSE1001 - Algorithms</td><td>-</td>` +
		`<td>-</td><td>-</td><td>-</td><td>` + slot + ` - SJT-101</td>` +
		`</tr></tbody></table>`
}

func plannerAttendanceHTML() string {
	return `<table id="AttendanceDetailDataTable"><tbody><tr>` +
		`<td>1</td><td>-</td><td><span>CSE1001 - Algorithms - Theory Only</span></td><td>-</td>` +
		`<td><span>DR ADA LOVELACE - 1001</span></td><td><span>9</span></td><td><span>10</span></td>` +
		`<td><span><span>90%</span></span></td>` +
		`</tr></tbody></table>`
}

func plannerExamHTML(date time.Time) string {
	return `<table class="customTable"><tbody>` +
		`<tr><td class="panelHead-secondary" colspan="13">CAT1</td></tr>` +
		`<tr><td>1</td><td>CSE1001</td><td>Algorithms</td><td>-</td><td>-</td><td>A1</td>` +
		`<td>` + date.Format("02-Jan-2006") + `</td><td>-</td><td>-</td><td>09:00</td>` +
		`<td>SJT-201</td><td>A</td><td>21</td></tr>` +
		`</tbody></table>`
}

func capturePlannerFeatureOutput(t *testing.T, invoke func()) (output string) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	previousStdout := os.Stdout
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		var captured bytes.Buffer
		_, _ = io.Copy(&captured, reader)
		done <- captured.String()
	}()

	defer func() {
		os.Stdout = previousStdout
		_ = writer.Close()
		output = <-done
		_ = reader.Close()
	}()

	invoke()
	return output
}
