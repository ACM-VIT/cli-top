package tests

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"cli-top/features"
	"cli-top/helpers"
	"cli-top/types"
)

const (
	academicSemesterPath = "/vtop/academics/common/StudentAttendance"
	academicExamPath     = "/vtop/examinations/doSearchExamScheduleForStudent"
	selectedSemesterID   = "SEM-SELECTED"
)

var academicFeatureCookies = types.Cookies{
	SERVERID:   "fake-server",
	CSRF:       "fake-csrf",
	JSESSIONID: "fake-session",
}

type academicFeatureRoundTripper struct {
	featurePath      string
	featureHTML      string
	semesterRequests int
	featureRequests  int
	selectedSemester string
	authorizedID     string
	formParseError   error
	uploadRequests   int
	uploadedICS      string
	uploadParseError error
}

func (rt *academicFeatureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return nil, fmt.Errorf("unexpected %s request for %s", req.Method, req.URL.Path)
	}

	var body string
	switch req.URL.Path {
	case academicSemesterPath:
		rt.semesterRequests++
		body = `
			<select id="semesterSubId">
				<option value="SEM-SELECTED">Selected Semester</option>
				<option value="SEM-OTHER">Other Semester</option>
			</select>`
	case rt.featurePath:
		rt.featureRequests++
		if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
			rt.formParseError = req.ParseMultipartForm(1 << 20)
		} else {
			rt.formParseError = req.ParseForm()
		}
		if rt.formParseError == nil {
			rt.selectedSemester = req.Form.Get("semesterSubId")
			rt.authorizedID = req.Form.Get("authorizedID")
		}
		body = rt.featureHTML
	case "/upload":
		rt.uploadRequests++
		rt.uploadParseError = req.ParseMultipartForm(1 << 20)
		if rt.uploadParseError == nil {
			file, _, err := req.FormFile("file")
			if err != nil {
				rt.uploadParseError = err
			} else {
				uploaded, readErr := io.ReadAll(file)
				_ = file.Close()
				if readErr != nil {
					rt.uploadParseError = readErr
				} else {
					rt.uploadedICS = string(uploaded)
				}
			}
		}
		body = `{"url":"https://calendar.test/exams.ics"}`
	default:
		return nil, fmt.Errorf("unexpected request path %q", req.URL.Path)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestAcademicFeaturesEndToEnd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	t.Setenv("SEMESTER_CACHE", "")

	t.Run("marks", func(t *testing.T) {
		tables := exerciseAcademicFeature(t, "E2E-MARKS-0001", "/vtop/examinations/doStudentMarkView", `
			<table>
				<tr class="tableContent">
					<td>1</td><td>-</td><td>CSE1001</td><td>Data Structures</td>
					<td>Theory Only</td><td>-</td><td>Dr Ada</td><td>A1</td>
				</tr>
			</table>
			<table class="customTable-level1">
				<tbody>
					<tr><td>1</td><td>CAT 1</td><td>50</td><td>15</td><td>Present</td><td>45</td><td>13.50</td></tr>
				</tbody>
			</table>`, features.GetMarks)

		assertAcademicTables(t, tables, []helpers.TableSnapshot{{
			Headers: []string{"TITLE", "MAX MARKS", "WEIGHTAGE %", "STATUS", "SCORED MARK", "WEIGHTAGE MARK"},
			Rows:    [][]string{{"CAT 1", "50", "15", "Present", "45", "13.50"}},
		}})
	})

	t.Run("grades", func(t *testing.T) {
		tables := exerciseAcademicFeature(t, "E2E-GRADES-0002", "/vtop/examinations/examGradeView/doStudentGradeView", `
			<table class="table-hover table-bordered">
				<tbody>
					<tr>
						<td>1</td><td>CSE2001</td><td>Algorithms</td><td>Theory Only</td>
						<td>-</td><td>-</td><td>-</td><td>4.0</td><td>RG</td><td>87</td><td>A</td>
					</tr>
				</tbody>
			</table>`, features.GetGrades)

		assertAcademicTables(t, tables, []helpers.TableSnapshot{{
			Headers:  []string{"INDEX", "COURSE CODE", "COURSE TITLE", "COURSE TYPE", "CREDITS", "TOTAL", "GRADING", "GRADE"},
			Rows:     [][]string{{"1", "CSE2001", "Algorithms", "Theory Only", "4.0", "87", "Relative", "A"}},
			HasIndex: true,
		}})
	})

	t.Run("attendance", func(t *testing.T) {
		tables := exerciseAcademicFeature(t, "E2E-ATTENDANCE-0003", "/vtop/processViewStudentAttendance", `
			<table id="AttendanceDetailDataTable">
				<tbody>
					<tr>
						<td>1</td><td>-</td><td><span>CSE2001 - Algorithms - Theory Only</span></td><td>-</td>
						<td><span>DR ADA LOVELACE - 1001</span></td><td><span>8</span></td><td><span>10</span></td>
						<td><span><span>80%</span></span></td>
					</tr>
				</tbody>
			</table>`, features.GetAttendance)

		assertAcademicTables(t, tables, []helpers.TableSnapshot{{
			Headers:  []string{"INDEX", "SUBJECT", "TYPE", "FACULTY NAME", "CLASSES ATTENDED", "PERCENTAGE", "75% ALERT"},
			Rows:     [][]string{{"1", "Algorithms", "Theory Only", "Dr Ada Lovelace", "8/10", "80%", "Can miss 0 class(es)"}},
			HasIndex: true,
		}})
	})

	t.Run("exam schedule", func(t *testing.T) {
		tables := exerciseAcademicFeature(t, "E2E-EXAMS-0004", academicExamPath, `
			<table class="customTable">
				<tbody>
					<tr><td class="panelHead-secondary" colspan="13">CAT1</td></tr>
					<tr>
						<td>1</td><td>CSE3001</td><td>Compiler Design</td><td>-</td><td>-</td><td>D1</td>
						<td>31-Dec-2099</td><td>-</td><td>8:45 AM</td><td>9:00 AM - 12:00 PM</td><td>SJT-101</td><td>A</td><td>42</td>
					</tr>
				</tbody>
			</table>`, features.GetExamSchedule)

		if len(tables) != 1 {
			t.Fatalf("captured %d exam tables, want 1: %#v", len(tables), tables)
		}
		wantHeaders := []string{"INDEX", "CODE", "COURSE TITLE", "SLOT", "EXAM DATE", "EXAM TIME", "VENUE", "SEAT", "SEAT NO.", "DAYS LEFT"}
		if !reflect.DeepEqual(tables[0].Headers, wantHeaders) {
			t.Fatalf("exam headers = %#v, want %#v", tables[0].Headers, wantHeaders)
		}
		if !tables[0].HasIndex || len(tables[0].Rows) != 1 || len(tables[0].Rows[0]) != len(wantHeaders) {
			t.Fatalf("unexpected exam table snapshot: %#v", tables[0])
		}
		wantRowPrefix := []string{"1", "CSE3001", "Compiler Design", "D1", "31-Dec-2099", "9:00 AM - 12:00 PM (Report by: 8:45 AM)", "SJT-101", "A", "42"}
		if !reflect.DeepEqual(tables[0].Rows[0][:len(wantRowPrefix)], wantRowPrefix) {
			t.Fatalf("exam row = %#v, want prefix %#v", tables[0].Rows[0], wantRowPrefix)
		}
		if daysLeft, err := strconv.Atoi(tables[0].Rows[0][len(wantRowPrefix)]); err != nil || daysLeft <= 0 {
			t.Fatalf("exam days left = %q, want a positive integer", tables[0].Rows[0][len(wantRowPrefix)])
		}

		icsPath := filepath.Join(home, "Downloads", "CLI-TOP Downloads", "Other Downloads", "ICS File", "Exam_Schedule.ics")
		ics, err := os.ReadFile(icsPath)
		if err != nil {
			t.Fatalf("read generated exam ICS file: %v", err)
		}
		for _, want := range []string{"BEGIN:VCALENDAR", "DTSTART;TZID=Asia/Kolkata:20991231T090000", "SUMMARY:CAT1: D1 - Compiler Design", "END:VCALENDAR"} {
			if !strings.Contains(string(ics), want) {
				t.Errorf("generated ICS is missing %q:\n%s", want, ics)
			}
		}
	})
}

func exerciseAcademicFeature(
	t *testing.T,
	regNo string,
	featurePath string,
	featureHTML string,
	run func(string, types.Cookies, int),
) []helpers.TableSnapshot {
	t.Helper()

	helpers.InvalidateSemesterCache(regNo)
	t.Cleanup(func() { helpers.InvalidateSemesterCache(regNo) })

	transport := &academicFeatureRoundTripper{
		featurePath: featurePath,
		featureHTML: featureHTML,
	}
	client := helpers.GetHTTPClient()
	previousTransport := client.Transport
	client.Transport = transport
	t.Cleanup(func() { client.Transport = previousTransport })

	var tables []helpers.TableSnapshot
	restoreTableCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
	})
	t.Cleanup(restoreTableCapture)

	run(regNo, academicFeatureCookies, 2)

	if transport.formParseError != nil {
		t.Fatalf("parse feature form: %v", transport.formParseError)
	}
	if transport.uploadParseError != nil {
		t.Fatalf("parse calendar upload: %v", transport.uploadParseError)
	}
	if featurePath == academicExamPath {
		if transport.uploadRequests != 1 {
			t.Fatalf("calendar upload count = %d, want 1", transport.uploadRequests)
		}
		if !strings.Contains(transport.uploadedICS, "SUMMARY:CAT1: D1 - Compiler Design") {
			t.Fatalf("uploaded calendar did not contain the exam event:\n%s", transport.uploadedICS)
		}
	}
	if transport.semesterRequests != 1 {
		t.Fatalf("semester request count = %d, want 1", transport.semesterRequests)
	}
	if transport.featureRequests != 1 {
		t.Fatalf("feature request count = %d, want 1", transport.featureRequests)
	}
	if transport.selectedSemester != selectedSemesterID {
		t.Fatalf("posted semester = %q, want %q", transport.selectedSemester, selectedSemesterID)
	}
	if transport.authorizedID != regNo {
		t.Fatalf("posted authorizedID = %q, want %q", transport.authorizedID, regNo)
	}

	return tables
}

func assertAcademicTables(t *testing.T, got, want []helpers.TableSnapshot) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("captured tables = %#v, want %#v", got, want)
	}
}
