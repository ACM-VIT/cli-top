package tests

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"cli-top/features"
	"cli-top/helpers"
	"cli-top/types"
)

const downloadE2EHost = "vtop.vit.ac.in"

var downloadE2ECookies = types.Cookies{
	CSRF:       "fake-download-csrf",
	JSESSIONID: "fake-download-session",
	SERVERID:   "fake-download-server",
}

type downloadE2ERoute struct {
	body          []byte
	headers       http.Header
	wantForm      map[string]string
	nonEmptyForm  []string
	expectedCalls int
}

type downloadE2ERoundTripper struct {
	cookies types.Cookies
	routes  map[string]downloadE2ERoute

	mu         sync.Mutex
	callCounts map[string]int
	violations []string
}

func newDownloadE2ERoundTripper(routes map[string]downloadE2ERoute) *downloadE2ERoundTripper {
	return &downloadE2ERoundTripper{
		cookies:    downloadE2ECookies,
		routes:     routes,
		callCounts: make(map[string]int),
	}
}

func (rt *downloadE2ERoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	route, exists := rt.routes[path]

	body, readErr := io.ReadAll(req.Body)
	form, formErr := url.ParseQuery(string(body))
	issues := rt.validateRequest(req, route, exists, form, readErr, formErr)

	rt.mu.Lock()
	rt.callCounts[path]++
	rt.violations = append(rt.violations, issues...)
	rt.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("download e2e transport rejected unexpected path %q", path)
	}

	headers := make(http.Header)
	for key, values := range route.headers {
		headers[key] = append([]string(nil), values...)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(route.body)),
		Request:    req,
	}, nil
}

func (rt *downloadE2ERoundTripper) validateRequest(
	req *http.Request,
	route downloadE2ERoute,
	routeExists bool,
	form url.Values,
	readErr error,
	formErr error,
) []string {
	var issues []string
	prefix := req.URL.Path + ": "

	if req.Method != http.MethodPost {
		issues = append(issues, prefix+fmt.Sprintf("method = %q, want POST", req.Method))
	}
	if req.URL.Scheme != "https" || req.URL.Host != downloadE2EHost {
		issues = append(issues, prefix+fmt.Sprintf("URL = %q, want https://%s/...", req.URL.String(), downloadE2EHost))
	}
	if req.URL.RawQuery != "" {
		issues = append(issues, prefix+fmt.Sprintf("unexpected query string %q", req.URL.RawQuery))
	}
	if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		issues = append(issues, prefix+fmt.Sprintf("Content-Type = %q", got))
	}
	if readErr != nil {
		issues = append(issues, prefix+fmt.Sprintf("read body: %v", readErr))
	}
	if formErr != nil {
		issues = append(issues, prefix+fmt.Sprintf("parse form: %v", formErr))
	}

	cookies := make(map[string]string)
	for _, cookie := range req.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	wantCookies := map[string]string{
		"JSESSIONID": rt.cookies.JSESSIONID,
		"SERVERID":   rt.cookies.SERVERID,
	}
	if len(cookies) != len(wantCookies) {
		issues = append(issues, prefix+fmt.Sprintf("cookies = %#v, want exactly the fake session cookies", cookies))
	}
	for name, want := range wantCookies {
		if got := cookies[name]; got != want {
			issues = append(issues, prefix+fmt.Sprintf("cookie %s = %q, want %q", name, got, want))
		}
	}

	if !routeExists {
		issues = append(issues, prefix+"path is not registered")
		return issues
	}

	wantKeyCount := len(route.wantForm) + len(route.nonEmptyForm)
	if len(form) != wantKeyCount {
		issues = append(issues, prefix+fmt.Sprintf("form keys = %#v, want %d exact keys", sortedDownloadE2EKeys(form), wantKeyCount))
	}
	for key, want := range route.wantForm {
		values, ok := form[key]
		if !ok || len(values) != 1 || values[0] != want {
			issues = append(issues, prefix+fmt.Sprintf("form[%s] = %#v, want [%q]", key, values, want))
		}
	}
	for _, key := range route.nonEmptyForm {
		values, ok := form[key]
		if !ok || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
			issues = append(issues, prefix+fmt.Sprintf("form[%s] = %#v, want one non-empty value", key, values))
		}
	}

	return issues
}

func (rt *downloadE2ERoundTripper) assertDone(t *testing.T) {
	t.Helper()

	rt.mu.Lock()
	counts := make(map[string]int, len(rt.callCounts))
	for path, count := range rt.callCounts {
		counts[path] = count
	}
	violations := append([]string(nil), rt.violations...)
	rt.mu.Unlock()

	if len(violations) > 0 {
		t.Errorf("mock transport rejected request details:\n%s", strings.Join(violations, "\n"))
	}
	for path, route := range rt.routes {
		want := route.expectedCalls
		if want == 0 {
			want = 1
		}
		if got := counts[path]; got != want {
			t.Errorf("request count for %s = %d, want %d", path, got, want)
		}
		delete(counts, path)
	}
	if len(counts) > 0 {
		t.Errorf("unexpected request counts: %#v", counts)
	}
}

func sortedDownloadE2EKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestDownloadFeaturesEndToEnd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	t.Setenv("SEMESTER_CACHE", "")
	t.Setenv("CSRF", downloadE2ECookies.CSRF)

	t.Run("ExecuteCoursePageDownload", func(t *testing.T) {
		regNo := "E2E-DOWNLOAD-CONSOLIDATED"
		pdf := []byte("%PDF-1.7\nconsolidated course material\n%%EOF\n")
		transport := newDownloadE2ERoundTripper(map[string]downloadE2ERoute{
			"/vtop/academics/common/CoursePageConsolidated": {
				body: []byte(`<select id="courseId">
					<option value="COURSE-CONSOLIDATED">Fall Semester 2025 - CSE1001 - Data Structures - Theory - L1 - Slot A1</option>
				</select>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"verifyMenu":   "true",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/academics/CoursePageConsolidated/getCourseDetail": {
				body: []byte(`<table id="materialTable"><tbody><tr>
					<td>1</td><td>Module 1</td>
					<td><div class="mt-1">
						<span style="color: #2E86C1">Lecture Notes.pdf</span>
						<span style="color: #28B463">1</span>
					</div></td>
					<td><div class="mt-1"><span>1001 - Ada Lovelace - SCOPE</span><span>10-Jul-2026</span></div></td>
					<td><button name="downloadmat" data-fileid="FILE-CONSOLIDATED">Download</button></td>
				</tr></tbody></table>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"CourseId":     "COURSE-CONSOLIDATED",
					"CoursType":    "Theory",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/downloadCourseMaterialFacultyPdf": {
				body: pdf,
				headers: http.Header{
					"Content-Disposition": {`attachment; filename="lecture-notes.pdf"`},
					"Content-Type":        {"application/pdf"},
				},
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"fileId":       "FILE-CONSOLIDATED",
				},
			},
		})
		installDownloadE2ETransport(t, transport, false)

		features.ExecuteCoursePageDownload(regNo, downloadE2ECookies, 1, 1, "1", "1")

		transport.assertDone(t)
		hash := sha256.Sum256(pdf)
		path := filepath.Join(
			home, "Downloads", "CLI-TOP Downloads", "Course Page",
			"Fall Semester 2025_CSE1001", "Theory", "Ada Lovelace",
			fmt.Sprintf("Module-1_Lecture Notes_%x.pdf", hash[:4]),
		)
		assertDownloadE2EFile(t, path, pdf)
	})

	t.Run("ExecuteCoursePageOldDownload", func(t *testing.T) {
		regNo := "E2E-DOWNLOAD-OLD"
		helpers.InvalidateSemesterCache(regNo)
		t.Cleanup(func() { helpers.InvalidateSemesterCache(regNo) })

		pdf := []byte("%PDF-1.7\nlegacy course material\n%%EOF\n")
		transport := newDownloadE2ERoundTripper(map[string]downloadE2ERoute{
			"/vtop/academics/common/StudentAttendance": {
				body: []byte(`<select id="semesterSubId"><option value="SEM-OLD">Fall Semester 2025</option></select>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"verifyMenu":   "true",
				},
				nonEmptyForm: []string{"nocache"},
			},
			"/vtop/getCourseForCoursePage": {
				body: []byte(`<select id="courseCode"><option value="CLASS-OLD">CSE2001 - Algorithms - Theory</option></select>`),
				wantForm: map[string]string{
					"_csrf":         downloadE2ECookies.CSRF,
					"authorizedID":  regNo,
					"paramReturnId": "getCourseForCoursePage",
					"semSubId":      "SEM-OLD",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/getSlotIdForCoursePage": {
				body: []byte(`<select id="slotId"><option value="SLOT-OLD">A1</option></select>`),
				wantForm: map[string]string{
					"_csrf":         downloadE2ECookies.CSRF,
					"authorizedID":  regNo,
					"paramReturnId": "getSlotIdForCoursePage",
					"semSubId":      "SEM-OLD",
					"classId":       "CLASS-OLD",
					"praType":       "source",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/getFacultyForCoursePage": {
				body: []byte(`<table><tbody><tr>
					<td>1</td><td>-</td><td>-</td><td>-</td><td>-</td><td>-</td>
					<td>A1</td><td>1002 - Grace Hopper - SCOPE</td>
					<td><button onclick="processViewStudentCourseDetail('SEM-OLD', 'ERP-OLD', 'CLASS-OLD')">View</button></td>
				</tr></tbody></table>`),
				wantForm: map[string]string{
					"_csrf":         downloadE2ECookies.CSRF,
					"authorizedID":  regNo,
					"paramReturnId": "getFacultyForCoursePage",
					"semSubId":      "SEM-OLD",
					"classId":       "CLASS-OLD",
					"slotId":        "SLOT-OLD",
					"praType":       "source",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/processViewStudentCourseDetail": {
				body: []byte(`<table>
					<thead><tr><th>Date</th><th>Day/Slot</th><th>M.No</th><th>T.No</th><th>Topic</th><th>Reference Material</th></tr></thead>
					<tbody><tr><td>10-Jul-2026</td><td>Mon/A1</td><td>1</td><td>1</td><td>Graphs</td>
						<td><button name="getDownloadSemPdf" data-matid="MAT-OLD" data-mdate="10-Jul-2026"><span>Graphs.pdf</span></button></td>
					</tr></tbody>
				</table>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"semSubId":     "SEM-OLD",
					"erpId":        "ERP-OLD",
					"classId":      "CLASS-OLD",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/downloadPdf": {
				body: pdf,
				headers: http.Header{
					"Content-Disposition": {`attachment; filename="graphs.pdf"`},
					"Content-Type":        {"application/pdf"},
				},
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"semSubId":     "SEM-OLD",
					"classId":      "CLASS-OLD",
					"materialId":   "MAT-OLD",
					"materialDate": "10-Jul-2026",
				},
				nonEmptyForm: []string{"x"},
			},
		})
		installDownloadE2ETransport(t, transport, false)

		features.ExecuteCoursePageOldDownload(regNo, downloadE2ECookies, 1, 1, "1", "1")

		transport.assertDone(t)
		path := filepath.Join(
			home, "Downloads", "CLI-TOP Downloads", "Course Page",
			"Algorithms_Theory_CSE2001", "A1_Grace-Hopper_SCOPE", "M1_T1_Graphs_1.pdf",
		)
		assertDownloadE2EFile(t, path, pdf)
	})

	t.Run("ExecuteSyllabusDownload", func(t *testing.T) {
		regNo := "E2E-DOWNLOAD-SYLLABUS"
		pdf := []byte("%PDF-1.7\nsyllabus from zip\n%%EOF\n")
		zippedPDF := makeDownloadE2EZip(t, "nested/syllabus.pdf", pdf)
		transport := newDownloadE2ERoundTripper(map[string]downloadE2ERoute{
			"/vtop/academics/common/Curriculum": {
				body: []byte(`<div class="card categoty-card">
					<div class="row" style="cursor: pointer" onclick="categoryOnClick('CAT-CORE')">
						<div class="col-6">Programme Core</div>
					</div>
				</div>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"verifyMenu":   "true",
				},
				nonEmptyForm: []string{"nocache"},
			},
			"/vtop/academics/common/curriculumCategoryView": {
				body: []byte(`<table class="example"><tbody><tr>
					<td>1</td><td><button data-coursecode="CSE1001">CSE1001</button></td><td>Data Structures</td>
				</tr></tbody></table>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"categoryId":   "CAT-CORE",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/courseSyllabusDownload1": {
				body: zippedPDF,
				headers: http.Header{
					"Content-Disposition": {`attachment; filename="syllabus.zip"`},
					"Content-Type":        {"application/zip"},
				},
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"courseCode":   "CSE1001",
				},
			},
		})
		installDownloadE2ETransport(t, transport, false)

		features.ExecuteSyllabusDownload(regNo, downloadE2ECookies, "1")

		transport.assertDone(t)
		path := filepath.Join(home, "Downloads", "CLI-TOP Downloads", "Syllabus", "Data Structures_CSE1001.pdf")
		assertDownloadE2EFile(t, path, pdf)
	})

	t.Run("PrintAllDAs", func(t *testing.T) {
		regNo := "E2E-DOWNLOAD-DA"
		helpers.InvalidateSemesterCache(regNo)
		t.Cleanup(func() { helpers.InvalidateSemesterCache(regNo) })

		pdf := []byte("%PDF-1.7\ndigital assignment question paper\n%%EOF\n")
		transport := newDownloadE2ERoundTripper(map[string]downloadE2ERoute{
			"/vtop/academics/common/StudentAttendance": {
				body: []byte(`<select id="semesterSubId"><option value="SEM-DA">Fall Semester 2025</option></select>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"verifyMenu":   "true",
				},
				nonEmptyForm: []string{"nocache"},
			},
			"/vtop/examinations/doDigitalAssignment": {
				body: []byte(`<table><tbody><tr class="tableContent">
					<td>1</td><td>CLASS-DA</td><td>CSE3001</td><td>Operating Systems</td><td>Theory</td>
				</tr></tbody></table>`),
				wantForm: map[string]string{
					"_csrf":         downloadE2ECookies.CSRF,
					"authorizedID":  regNo,
					"semesterSubId": "SEM-DA",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/examinations/processDigitalAssignment": {
				body: []byte(`<table class="customTable">
					<tr class="tableHeader">
						<td>Sl.No.</td><td>Title</td><td>Description</td><td>Start Date</td><td>Due Date</td><td>QP</td><td>Last Updated</td><td>Upload</td><td>Status</td>
					</tr>
					<tr class="tableContent">
						<td>1</td><td>Assignment One</td><td>Processes</td><td>01-Dec-2019</td><td>01-Jan-2020</td>
						<td><button data-code="QP-DA" data-classid="CLASS-DA">Download</button></td>
						<td>N/A</td><td><input name="code" value="EDIT-DA"></td><td>Pending</td>
					</tr>
				</table>`),
				wantForm: map[string]string{
					"_csrf":        downloadE2ECookies.CSRF,
					"authorizedID": regNo,
					"classId":      "CLASS-DA",
				},
				nonEmptyForm: []string{"x"},
			},
			"/vtop/examinations/doDownloadQuestion/": {
				body: pdf,
				headers: http.Header{
					"Content-Disposition": {`attachment; filename="assignment-one.pdf"`},
					"Content-Type":        {"application/pdf"},
				},
				wantForm: map[string]string{
					"_csrf":         downloadE2ECookies.CSRF,
					"authorizedID":  regNo,
					"code":          "QP-DA",
					"classIdNumber": "CLASS-DA",
				},
				nonEmptyForm: []string{"x"},
			},
		})
		installDownloadE2ETransport(t, transport, true)

		features.PrintAllDAs(regNo, downloadE2ECookies, "1", "1")

		transport.assertDone(t)
		path := filepath.Join(
			home, "Downloads", "CLI-TOP Downloads", "DA",
			"CSE3001_Operating Systems", "Assignment One.pdf",
		)
		assertDownloadE2EFile(t, path, pdf)

		icsPath := filepath.Join(home, "Downloads", "CLI-TOP Downloads", "Other Downloads", "ICS File", "All_DA_Deadlines.ics")
		if _, err := os.Stat(icsPath); !os.IsNotExist(err) {
			t.Fatalf("past-due DA fixture unexpectedly created %q (stat error: %v)", icsPath, err)
		}
	})
}

func installDownloadE2ETransport(t *testing.T, transport http.RoundTripper, replaceDefault bool) {
	t.Helper()

	client := helpers.GetHTTPClient()
	previousTransport := client.Transport
	client.Transport = transport
	t.Cleanup(func() { client.Transport = previousTransport })

	if replaceDefault {
		previousDefault := http.DefaultTransport
		http.DefaultTransport = transport
		t.Cleanup(func() { http.DefaultTransport = previousDefault })
	}
}

func assertDownloadE2EFile(t *testing.T, path string, want []byte) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file %q: %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("downloaded file %q = %q, want %q", path, got, want)
	}
}

func makeDownloadE2EZip(t *testing.T, name string, contents []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := file.Write(contents); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip fixture: %v", err)
	}
	return buffer.Bytes()
}
