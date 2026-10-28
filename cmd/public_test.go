package cmd

import (
	"bytes"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/types"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type contractTransport func(*http.Request) (*http.Response, error)

func (f contractTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fixtureTransport(t *testing.T, fn func(*http.Request) (string, int)) {
	t.Helper()
	client := helpers.GetHTTPClient()
	old := client.Transport
	client.Transport = contractTransport(func(r *http.Request) (*http.Response, error) {
		body, status := fn(r)
		return &http.Response{StatusCode: status, Status: fmt.Sprint(status), Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	t.Cleanup(func() { client.Transport = old })
}
func fixtureSession(t *testing.T) features.Session {
	return features.Session{Cookies: types.Cookies{CSRF: "secret-csrf", JSESSIONID: "secret-session", SERVERID: "secret-server"}, RegNo: t.Name()}
}
func invoke(t *testing.T, deps Dependencies, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	status := RunWithDependencies(args, strings.NewReader(""), &out, &errOut, deps)
	return status, out.String(), errOut.String()
}
func decodeReply(t *testing.T, raw string) contract.Result {
	t.Helper()
	var reply contract.Result
	if err := json.Unmarshal([]byte(raw), &reply); err != nil {
		t.Fatalf("not a single JSON document: %v\n%s", err, raw)
	}
	return reply
}
func TestOfflineDiscoveryAndSyntax(t *testing.T) {
	calls := 0
	deps := Dependencies{Authenticate: func() (features.Session, error) { calls++; return features.Session{}, errors.New("unexpected auth") }, CheckPolicy: func() int { calls++; return 0 }}
	for _, args := range [][]string{{"--help"}, {"marks", "--help"}, {"--version"}, {"completion", "bash"}, {"schema"}, {"schema", "marks"}, {"schema", "course-page"}} {
		status, _, stderr := invoke(t, deps, args...)
		if status != 0 || stderr != "" {
			t.Fatalf("%v = %d %s", args, status, stderr)
		}
	}
	for _, args := range [][]string{{"marks", "--semester", "wat", "--output", "json"}, {"marks", "--semester-id", "S", "--semester", "1", "--output", "json"}, {"profile", "--wat", "--output", "json"}, {"profile", "extra", "--output", "json"}, {"courses", "--source", "wrong", "--output", "json"}, {"faculties", "--source", "syllabus", "--output", "json"}, {"events", "--confirm", "--output", "json"}, {"leave", "--apply", "--output", "json"}, {"marks", "--semester", "-1", "--output", "json"}, {"leave", "--reason", "x", "--output", "json"}} {
		status, raw, stderr := invoke(t, deps, args...)
		reply := decodeReply(t, raw)
		if status != 2 || reply.Error == nil || reply.Error.Kind != "invalid_arguments" || stderr != "" {
			t.Fatalf("%v => %d %s %s", args, status, raw, stderr)
		}
	}
	if calls != 0 {
		t.Fatalf("offline and invalid commands invoked services %d times", calls)
	}
}
func TestSchemaReflectsFlagsAndDomainTypes(t *testing.T) {
	_, raw, _ := invoke(t, Dependencies{}, "schema", "marks")
	var schema commandSchema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		t.Fatal(err)
	}
	if !schema.AuthRequired || schema.Effect != "read" {
		t.Fatalf("bad metadata %+v", schema)
	}
	found := false
	for _, f := range schema.Flags {
		if f.Name == "semester-id" {
			found = true
			if f.Type != "string" || len(f.Conflicts) != 1 || f.Conflicts[0] != "--semester" {
				t.Fatalf("bad flag %+v", f)
			}
		}
	}
	if !found {
		t.Fatal("missing ID flag")
	}
	r := schema.Result.(map[string]any)
	props := r["properties"].(map[string]any)
	data := props["data"].(map[string]any)["anyOf"].([]any)[0].(map[string]any)
	course := data["properties"].(map[string]any)["courses"].(map[string]any)["items"].(map[string]any)
	assessment := course["properties"].(map[string]any)["assessments"].(map[string]any)["items"].(map[string]any)
	score := assessment["properties"].(map[string]any)["score"].(map[string]any)["anyOf"].([]any)
	if score[0].(map[string]any)["type"] != "number" || score[1].(map[string]any)["type"] != "null" {
		t.Fatal("score must be nullable numeric")
	}
}
func TestSelectionAndExactID(t *testing.T) {
	session := fixtureSession(t)
	featureCalls := 0
	fixtureTransport(t, func(r *http.Request) (string, int) {
		switch r.URL.Path {
		case "/vtop/academics/common/StudentAttendance":
			return `<select id="semesterSubId"><option value="S2">Winter</option><option value="S1">Fall</option></select>`, 200
		case "/vtop/examinations/doStudentMarkView":
			featureCalls++
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("semesterSubId") != "S1" {
				t.Fatal("exact semester ID lost")
			}
			return `<table><tr class="tableContent"><td>1</td><td>-</td><td>CSE1001</td><td>Data Structures</td><td>Theory Only</td><td>-</td><td>Ada</td><td>A1</td></tr></table><table class="customTable-level1"><tbody><tr><td>1</td><td>CAT 1</td><td>50</td><td>15</td><td>Present</td><td>45</td><td>13.5</td></tr></tbody></table>`, 200
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return "", 500
		}
	})
	deps := Dependencies{Authenticate: func() (features.Session, error) { return session, nil }, CheckPolicy: func() int { return 0 }}
	status, raw, stderr := invoke(t, deps, "marks", "--output", "json")
	reply := decodeReply(t, raw)
	if status != 4 || reply.Error.Kind != "selection_required" || len(reply.Error.Choices) != 2 || len(reply.Error.Next) == 0 || featureCalls != 0 || stderr != "" {
		t.Fatalf("bad selection response %d %s %s", status, raw, stderr)
	}
	status, raw, _ = invoke(t, deps, "marks", "--semester-id", "Fall", "--output", "json")
	reply = decodeReply(t, raw)
	if status != 2 || reply.Error.Kind != "invalid_selection" || featureCalls != 0 {
		t.Fatalf("ID fell back to label: %s", raw)
	}
	status, raw, stderr = invoke(t, deps, "marks", "--semester-id", "S1", "--output", "json", "--debug")
	reply = decodeReply(t, raw)
	if status != 0 || !reply.Success || stderr != "" || featureCalls != 1 {
		t.Fatalf("marks failed: %d %s %s", status, raw, stderr)
	}
	data := reply.Data.(map[string]any)
	course := data["courses"].([]any)[0].(map[string]any)
	score := course["assessments"].([]any)[0].(map[string]any)["score"]
	if score != float64(45) || course["code"] != "CSE1001" {
		t.Fatalf("bad domain data %+v", data)
	}
	if strings.Contains(raw, "secret-") || strings.Contains(raw, "\x1b") {
		t.Fatal("leaked auth or terminal formatting")
	}
	// A fresh command tree must not retain the previous explicit selector.
	status, raw, _ = invoke(t, deps, "marks", "--output", "json")
	if status != 4 {
		t.Fatalf("flags leaked between invocations: %s", raw)
	}
}
func TestDomainFailureExitCodes(t *testing.T) {
	for _, test := range []struct {
		kind   string
		status int
	}{{"auth_required", 3}, {"auth_failed", 3}, {"selection_required", 4}, {"network_error", 5}, {"parse_error", 5}, {"partial_failure", 7}, {"canceled", 130}, {"io_error", 1}} {
		t.Run(test.kind, func(t *testing.T) {
			deps := Dependencies{CheckPolicy: func() int { return 0 }, Authenticate: func() (features.Session, error) {
				return features.Session{}, contract.New(test.kind, "fixture failure")
			}}
			status, raw, stderr := invoke(t, deps, "profile", "--output", "json")
			reply := decodeReply(t, raw)
			if status != test.status || reply.Success || reply.Error.Kind != test.kind || stderr != "" {
				t.Fatalf("got %d %s %s", status, raw, stderr)
			}
		})
	}
	for _, kill := range []int{2, 3} {
		deps := Dependencies{CheckPolicy: func() int { return kill }, Authenticate: func() (features.Session, error) { t.Fatal("auth after policy block"); return features.Session{}, nil }}
		status, raw, _ := invoke(t, deps, "profile", "--output", "json")
		if status != 6 {
			t.Fatalf("policy did not fail: %s", raw)
		}
	}
}
func TestReceiptsAndUpstreamFailures(t *testing.T) {
	session := fixtureSession(t)
	statusCode := 200
	body := `<table class="table-bordered"><tbody><tr><td>Invoice Number</td><td>Receipt Number</td><td>Date</td><td>Amount</td></tr><tr><td>I1</td><td>R1</td><td>2026-01-01</td><td>1,200.50</td></tr></tbody></table>`
	fixtureTransport(t, func(r *http.Request) (string, int) { return body, statusCode })
	deps := Dependencies{CheckPolicy: func() int { return 0 }, Authenticate: func() (features.Session, error) { return session, nil }}
	status, raw, _ := invoke(t, deps, "receipts", "--output", "json")
	reply := decodeReply(t, raw)
	if status != 0 {
		t.Fatal(raw)
	}
	receipt := reply.Data.(map[string]any)["receipts"].([]any)[0].(map[string]any)
	if receipt["amount"] != 1200.5 || receipt["invoice_number"] != "I1" || receipt["receipt_number"] != "R1" || receipt["date"] != "2026-01-01" {
		t.Fatalf("wrong receipt mapping %+v", receipt)
	}
	statusCode = 503
	status, raw, _ = invoke(t, deps, "receipts", "--output", "json")
	reply = decodeReply(t, raw)
	if status != 5 || reply.Error.Status != 503 || !reply.Error.Retryable {
		t.Fatalf("status lost %d %s", status, raw)
	}
	statusCode = 200
	body = "<html>unexpected login form</html>"
	status, raw, _ = invoke(t, deps, "receipts", "--output", "json")
	if status != 5 || decodeReply(t, raw).Error.Kind != "parse_error" {
		t.Fatal(raw)
	}
}
func TestProxyRejectsInvalidInputsBeforeNetwork(t *testing.T) {
	deps := Dependencies{CheckPolicy: func() int { t.Fatal("policy/network reached for invalid proxy input"); return 0 }}
	for _, test := range []struct {
		args []string
		want int
	}{{[]string{"proxy", "", "", "profile"}, 3}, {[]string{"proxy", "user", "password", "marks", "--semester", "wrong"}, 2}, {[]string{"proxy", "user", "password", "profile", "--wat"}, 2}, {[]string{"proxy", "user", "password", "unsupported"}, 2}} {
		status, raw, stderr := invoke(t, deps, test.args...)
		var response proxyResponse
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			t.Fatal(err)
		}
		if status != test.want || response.Success || response.Error == nil || stderr != "" {
			t.Fatalf("bad proxy response %d %s %s", status, raw, stderr)
		}
	}
}

func TestCourseDiscoveryAndDownload(t *testing.T) {
	t.Chdir(t.TempDir())
	session := fixtureSession(t)
	downloads := 0
	fixtureTransport(t, func(r *http.Request) (string, int) {
		switch r.URL.Path {
		case "/vtop/academics/common/CoursePageConsolidated":
			return `<select id="courseId"><option value="COURSE1">Fall Semester 2026-27 - CSE1001 - Algorithms - Theory Only - A - B</option><option value="COURSE2">Fall Semester 2026-27 - CSE1002 - Databases - Theory Only - A - B</option></select>`, 200
		case "/vtop/academics/CoursePageConsolidated/getCourseDetail":
			_ = r.ParseForm()
			if r.Form.Get("CourseId") != "COURSE1" {
				t.Fatal("wrong course scope")
			}
			return `<table id="materialTable"><tbody><tr><td>1</td><td>-</td><td><div class="mt-1"><span style="color:#2E86C1">Lecture One</span></div></td><td><div class="mt-1"><span>ERP1 - Ada - SCOPE</span><span>2026-01-01</span></div></td><td><button name="downloadmat" data-fileid="FILE1"></button></td></tr><tr><td>2</td><td>-</td><td>Lecture Two</td><td><div class="mt-1"><span>ERP2 - Grace - SCOPE</span><span>2026-01-01</span></div></td><td><button name="downloadmat" data-fileid="FILE2"></button></td></tr></tbody></table>`, 200
		case "/vtop/downloadCourseMaterialFacultyPdf":
			downloads++
			_ = r.ParseForm()
			if r.Form.Get("fileId") != "FILE1" {
				t.Fatal("faculty scope lost")
			}
			return "%PDF-1.4 fixture", 200
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return "", 500
		}
	})
	deps := Dependencies{CheckPolicy: func() int { return 0 }, Authenticate: func() (features.Session, error) { return session, nil }}
	status, raw, _ := invoke(t, deps, "courses", "--output", "json")
	if status != 0 {
		t.Fatal(raw)
	}
	options := decodeReply(t, raw).Data.(map[string]any)["options"].([]any)
	if len(options) != 2 || options[0].(map[string]any)["id"] != "COURSE1" {
		t.Fatal(raw)
	}
	status, raw, _ = invoke(t, deps, "course-page", "--course-id", "COURSE1", "--output", "json")
	if status != 4 || len(decodeReply(t, raw).Error.Choices) != 2 {
		t.Fatal(raw)
	}
	status, raw, _ = invoke(t, deps, "course-page", "--course-id", "COURSE1", "--faculty-id", "ERP1", "--output", "json")
	reply := decodeReply(t, raw)
	if status != 0 || downloads != 0 {
		t.Fatalf("listing downloaded files: %s", raw)
	}
	materials := reply.Data.(map[string]any)["materials"].([]any)
	if len(materials) != 1 || materials[0].(map[string]any)["id"] != "FILE1" {
		t.Fatal(raw)
	}
	status, raw, _ = invoke(t, deps, "course-page", "--course-id", "COURSE1", "--faculty-id", "ERP1", "--materials", "0", "--output", "json")
	if status != 0 || downloads != 1 {
		t.Fatalf("download failed %d %s", status, raw)
	}
	files := decodeReply(t, raw).Data.(map[string]any)["files"].([]any)
	if len(files) != 1 {
		t.Fatal(raw)
	}
	path := files[0].(map[string]any)["path"].(string)
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "%PDF-1.4 fixture" {
		t.Fatalf("saved artifact: %q %v", content, err)
	}
}
func TestPlanningPartialFailureAndNoInstructionalDay(t *testing.T) {
	session := fixtureSession(t)
	session.Now = func() time.Time { return time.Date(2026, 10, 26, 8, 0, 0, 0, time.FixedZone("IST", 19800)) }
	attendanceFails := false
	statusText := "Instructional Day"
	fixtureTransport(t, func(r *http.Request) (string, int) {
		switch r.URL.Path {
		case "/vtop/academics/common/StudentAttendance":
			return `<select id="semesterSubId"><option value="S1">Fall</option></select>`, 200
		case "/vtop/processViewTimeTable":
			return `<table class="table"><tbody><tr><td>1</td><td>-</td><td>CSE1001 - Algorithms (Theory Only)</td><td>-</td><td>-</td><td>-</td><td>-</td><td>A1 - SJT101</td></tr></tbody></table>`, 200
		case "/vtop/getDateForSemesterPreview":
			return `<select id="classGroupId"><option value="G1">Group A</option></select>`, 200
		case "/vtop/getListForSemester":
			return `<a class="btn btn-md btn-primary" onclick="view('01-OCT-2026')">Oct</a>`, 200
		case "/vtop/processViewCalendar":
			return `<table class="calendar-table"><tr><td><span>26</span><span>` + statusText + `</span></td></tr></table>`, 200
		case "/vtop/examinations/doSearchExamScheduleForStudent":
			return `<p>No exams scheduled</p>`, 200
		case "/vtop/processViewStudentAttendance":
			if attendanceFails {
				return "unavailable", 503
			}
			return `<table id="AttendanceDetailDataTable"><tbody><tr><td>1</td><td>-</td><td>CSE1001 - Algorithms - Theory Only</td><td>-</td><td>Ada</td><td>10</td><td>10</td><td>100%</td></tr></tbody></table>`, 200
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return "", 500
		}
	})
	deps := Dependencies{CheckPolicy: func() int { return 0 }, Authenticate: func() (features.Session, error) { return session, nil }}
	status, raw, _ := invoke(t, deps, "today", "--output", "json")
	reply := decodeReply(t, raw)
	if status != 0 {
		t.Fatal(raw)
	}
	data := reply.Data.(map[string]any)
	classes := data["classes"].([]any)
	if len(classes) != 1 || classes[0].(map[string]any)["can_skip"] != true || data["complete"] != true {
		t.Fatalf("incorrect planner: %s", raw)
	}
	attendanceFails = true
	status, raw, _ = invoke(t, deps, "today", "--output", "json")
	reply = decodeReply(t, raw)
	if status != 7 || reply.Error.Kind != "partial_failure" || len(reply.Warnings) != 1 {
		t.Fatal(raw)
	}
	data = reply.Data.(map[string]any)
	if data["complete"] != false || data["classes"].([]any)[0].(map[string]any)["can_skip"] != nil {
		t.Fatal(raw)
	}
	statusText = "No Instructional Day"
	status, raw, _ = invoke(t, deps, "today", "--output", "json")
	reply = decodeReply(t, raw)
	if status != 0 || len(reply.Data.(map[string]any)["classes"].([]any)) != 0 {
		t.Fatalf("scheduled classes on a noninstructional day: %s", raw)
	}
}
func TestProxySharesTypedResultsAndPartialSync(t *testing.T) {
	session := fixtureSession(t)
	fixtureTransport(t, func(r *http.Request) (string, int) {
		switch r.URL.Path {
		case "/vtop/studentsRecord/StudentProfileAllView":
			return `<label for="no">TEST1</label><label for="branchno">CS</label><label for="vmail">test@example.com</label><label for="schoolno">SCOPE</label>`, 200
		default:
			return "unavailable", 503
		}
	})
	deps := Dependencies{CheckPolicy: func() int { return 0 }, ProxyAuthenticate: func(u, p string) (features.Session, error) { return session, nil }}
	status, raw, stderr := invoke(t, deps, "proxy", "user", "password", "profile")
	var response proxyResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatal(err)
	}
	if status != 0 || !response.Success || response.StructuredRaw["data"].(map[string]any)["register_number"] != "TEST1" || stderr != "" {
		t.Fatalf("proxy failed: %s", raw)
	}
	status, raw, _ = invoke(t, deps, "proxy", "user", "password", "sync", "--commands", "profile,receipts")
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatal(err)
	}
	if status != 7 || response.Success || response.Error.Kind != "partial_failure" {
		t.Fatalf("partial sync masked: %s", raw)
	}
	entries := response.StructuredRaw["results"].([]any)
	if entries[0].(map[string]any)["success"] != true || entries[1].(map[string]any)["success"] != false {
		t.Fatal(raw)
	}
}

func TestEventRegistrationRequiresConfirmationAndVerifiesResult(t *testing.T) {
	session := fixtureSession(t)
	submissions := 0
	page := func(registered bool) string {
		table, status := "dataTable1", `<button onclick="doEventRegistraiton('E1')">Register</button>`
		if registered {
			table, status = "dataTable2", "Registered"
		}
		return fmt.Sprintf(`<table id="%s"><tbody><tr><td>1</td><td>ACM</td><td>Hack Night</td><td>Description</td><td><span>31-Dec-2099</span></td><td><span>06:00 PM</span><span>09:00 PM</span></td><td>-</td><td><span>01-Jan-2026</span><span>30-Dec-2099</span></td><td>SJT101</td><td>%s</td></tr></tbody></table>`, table, status)
	}
	fixtureTransport(t, func(r *http.Request) (string, int) {
		if r.URL.Path == "/vtop/event/swf/registered/doEventRegistraiton" {
			submissions++
			return page(true), 200
		}
		return page(false), 200
	})
	deps := Dependencies{CheckPolicy: func() int { return 0 }, Authenticate: func() (features.Session, error) { return session, nil }}
	status, raw, _ := invoke(t, deps, "events", "--event-id", "E1", "--output", "json")
	if status != 4 || decodeReply(t, raw).Error.Kind != "confirmation_required" || submissions != 0 {
		t.Fatalf("unexpected submission %s", raw)
	}
	status, raw, _ = invoke(t, deps, "events", "--event-id", "E1", "--confirm", "--output", "json")
	reply := decodeReply(t, raw)
	if status != 0 || submissions != 1 {
		t.Fatalf("registration not verified: %d %s", status, raw)
	}
	registered := reply.Data.(map[string]any)["registered"].([]any)
	if len(registered) != 1 || registered[0].(map[string]any)["id"] != nil {
		t.Fatalf("missing registered ID must remain null: %s", raw)
	}
}
