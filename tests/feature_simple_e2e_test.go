package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"cli-top/features"
	"cli-top/helpers"
	"cli-top/types"
)

const simpleFeatureRegNo = "24BCE0001"

var simpleFeatureCookies = types.Cookies{
	CSRF:       "fake-csrf-token",
	JSESSIONID: "fake-session-id",
	SERVERID:   "fake-server-id",
}

type simpleFeatureRoundTripFunc func(*http.Request) (*http.Response, error)

func (f simpleFeatureRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type simpleFeatureRequest struct {
	method      string
	path        string
	contentType string
	cookie      string
	form        url.Values
}

func TestSimpleReadOnlyFeaturesEndToEnd(t *testing.T) {
	t.Setenv("CLI_TOP_PROXY_MODE", "1")

	t.Run("Profile", func(t *testing.T) {
		html := `<html><body>
			<label for="no">24BCE0001</label>
			<label for="branchno">B.Tech. Computer Science and Engineering</label>
			<label for="vmail">student@vitstudent.ac.in</label>
			<label for="schoolno">School of Computer Science and Engineering</label>
		</body></html>`

		request, snapshots, _ := runSimpleFeature(t,
			"/vtop/studentsRecord/StudentProfileAllView",
			html,
			func() { features.Profile(simpleFeatureCookies, simpleFeatureRegNo) },
		)

		assertSimpleFeatureRequest(t, request, true)
		assertSimpleFeatureSnapshots(t, snapshots, helpers.TableSnapshot{
			Headers: []string{"FIELD", "INFORMATION"},
			Rows: [][]string{
				{"Register Number", "24BCE0001"},
				{"Program & Branch", "B.Tech. Computer Science and Engineering"},
				{"VIT Email", "student@vitstudent.ac.in"},
				{"School Name", "School of Computer Science and Engineering"},
			},
		})
	})

	t.Run("PrintHostelInfo", func(t *testing.T) {
		html := `<div class="table-responsive"><table class="table"><tbody>
			<tr><td>Application Number</td><td>HOSTEL-2026-001</td></tr>
			<tr><td>Block</td><td>Q Block</td></tr>
			<tr><td>Room</td><td>412</td></tr>
			<tr><td>Bed Type</td><td>4 Bed AC</td></tr>
			<tr><td>Mess</td><td>Special</td></tr>
			<tr><td>Status</td><td>Allotted</td></tr>
		</tbody></table></div>`
		endpoint := "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView"

		request, snapshots, output := runSimpleFeature(t,
			"/vtop/studentsRecord/StudentProfileAllView",
			html,
			func() { features.PrintHostelInfo(simpleFeatureRegNo, simpleFeatureCookies, endpoint) },
		)

		assertSimpleFeatureRequest(t, request, true)
		assertSimpleFeatureSnapshots(t, snapshots, helpers.TableSnapshot{
			Headers: []string{"FIELD", "INFORMATION"},
			Rows: [][]string{
				{"Block", "Q Block"},
				{"Room", "412"},
				{"Bed Type", "4 Bed AC"},
				{"Mess", "Special"},
				{"Status", "Allotted"},
			},
		})
		if !strings.Contains(output, "Student Accommodation Info") {
			t.Fatalf("output = %q, want accommodation heading", output)
		}
	})

	t.Run("PrintCgpa", func(t *testing.T) {
		html := `<div class="table-responsive"><table class="table"><tbody><tr>
			<td>160</td><td>154</td><td>9.42</td><td>12</td><td>18</td><td>8</td>
			<td>2</td><td>1</td><td>0</td><td>0</td><td>0</td>
		</tr></tbody></table></div>`
		endpoint := "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory"

		request, snapshots, output := runSimpleFeature(t,
			"/vtop/examinations/examGradeView/StudentGradeHistory",
			html,
			func() { features.PrintCgpa(simpleFeatureRegNo, simpleFeatureCookies, endpoint) },
		)

		assertSimpleFeatureRequest(t, request, true)
		assertSimpleFeatureSnapshots(t, snapshots, helpers.TableSnapshot{
			Headers: []string{"GRADE", "COUNT"},
			Rows: [][]string{
				{"S Grades", "12"},
				{"A Grades", "18"},
				{"B Grades", "8"},
				{"C Grades", "2"},
				{"D Grades", "1"},
				{"E Grades", "0"},
				{"F Grades", "0"},
				{"N Grades", "0"},
			},
		})
		for _, want := range []string{"Credits Registered: 160", "Credits Earned: 154", "CGPA: \x1b[32m9.42\x1b[0m"} {
			if !strings.Contains(output, want) {
				t.Errorf("output = %q, want %q", output, want)
			}
		}
	})

	t.Run("GetReceipt", func(t *testing.T) {
		html := `<table class="table-bordered"><tbody>
			<tr><td>Invoice Number</td><td>Receipt Number</td><td>Date</td><td>Amount</td><td>View</td></tr>
			<tr><td>INV-2026-001</td><td>RCPT-001</td><td>01-Jul-2026</td><td>95000.00</td><td>VIEW</td></tr>
		</tbody></table>`

		request, snapshots, _ := runSimpleFeature(t,
			"/vtop/finance/getStudentReceipts",
			html,
			func() { features.GetReceipt(simpleFeatureRegNo, simpleFeatureCookies) },
		)

		assertSimpleFeatureRequest(t, request, true)
		assertSimpleFeatureSnapshots(t, snapshots, helpers.TableSnapshot{
			Headers:  []string{"INDEX", "INVOICE NUMBER", "RECEIPT NUMBER", "DATE", "AMOUNT"},
			Rows:     [][]string{{"1", "INV-2026-001", "RCPT-001", "01-Jul-2026", "95000.00"}},
			HasIndex: true,
		})
	})

	t.Run("GetLibraryDues", func(t *testing.T) {
		html := `<table class="table-bordered"><tbody>
			<tr><td>Overdue book fine</td><td>250.00</td></tr>
			<tr><td>Lost library card</td><td>100.00</td></tr>
		</tbody></table>`

		request, snapshots, _ := runSimpleFeature(t,
			"/vtop/finance/libraryPayments",
			html,
			func() { features.GetLibraryDues(simpleFeatureRegNo, simpleFeatureCookies) },
		)

		assertSimpleFeatureRequest(t, request, true)
		assertSimpleFeatureSnapshots(t, snapshots, helpers.TableSnapshot{
			Headers: []string{"INDEX", "TYPE", "AMOUNT"},
			Rows: [][]string{
				{"1", "Overdue book fine", "250.00"},
				{"2", "Lost library card", "100.00"},
			},
			HasIndex: true,
		})
	})

	t.Run("GetClassMessage", func(t *testing.T) {
		html := `<html><body>
			<h5><span>CSE2001 - Data Structures and Algorithms</span><span>Bring your lab record for evaluation on Monday.</span></h5>
			<h5><span>MAT2001 - Linear Algebra - Online Course</span><span>Quiz 2 opens at 6 PM and closes at 7 PM.</span></h5>
		</body></html>`

		request, snapshots, _ := runSimpleFeature(t,
			"/vtop/academics/common/StudentClassMessage",
			html,
			func() { features.GetClassMessage(simpleFeatureRegNo, simpleFeatureCookies) },
		)

		assertSimpleFeatureRequest(t, request, false)
		if request.form.Get("x") == "" {
			t.Fatal("class message request is missing its UTC cache-buster")
		}
		assertSimpleFeatureSnapshots(t, snapshots, helpers.TableSnapshot{
			Headers: []string{"INDEX", "COURSE", "MESSAGE"},
			Rows: [][]string{
				{"1", "Data Structures and Algorithms", "Bring your lab record for evaluation on Monday."},
				{"2", "Linear Algebra", "Quiz 2 opens at 6 PM and closes at 7 PM."},
			},
			HasIndex: true,
		})
	})
}

func runSimpleFeature(t *testing.T, expectedPath, html string, invoke func()) (simpleFeatureRequest, []helpers.TableSnapshot, string) {
	t.Helper()

	client := helpers.GetHTTPClient()
	previousTransport := client.Transport
	var requests []simpleFeatureRequest
	client.Transport = simpleFeatureRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != expectedPath {
			return nil, fmt.Errorf("unexpected request path %q; want %q", req.URL.Path, expectedPath)
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("parse request body: %w", err)
		}
		requests = append(requests, simpleFeatureRequest{
			method:      req.Method,
			path:        req.URL.Path,
			contentType: req.Header.Get("Content-Type"),
			cookie:      req.Header.Get("Cookie"),
			form:        form,
		})

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(html)),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() { client.Transport = previousTransport })

	var snapshots []helpers.TableSnapshot
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		snapshots = append(snapshots, snapshot)
	})
	t.Cleanup(restoreCapture)

	output := captureSimpleFeatureOutput(t, invoke)
	if len(requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(requests))
	}
	return requests[0], snapshots, output
}

func assertSimpleFeatureRequest(t *testing.T, request simpleFeatureRequest, expectVerifyMenu bool) {
	t.Helper()

	if request.method != http.MethodPost {
		t.Errorf("request method = %q, want POST", request.method)
	}
	if request.path == "" {
		t.Error("request path is empty")
	}
	if request.contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", request.contentType)
	}
	if request.cookie != "SERVERID=fake-server-id; JSESSIONID=fake-session-id" {
		t.Errorf("Cookie = %q, want fake VTOP session cookies", request.cookie)
	}
	if got := request.form.Get("authorizedID"); got != simpleFeatureRegNo {
		t.Errorf("authorizedID = %q, want %q", got, simpleFeatureRegNo)
	}
	if got := request.form.Get("_csrf"); got != simpleFeatureCookies.CSRF {
		t.Errorf("_csrf = %q, want %q", got, simpleFeatureCookies.CSRF)
	}
	wantVerifyMenu := ""
	if expectVerifyMenu {
		wantVerifyMenu = "true"
	}
	if got := request.form.Get("verifyMenu"); got != wantVerifyMenu {
		t.Errorf("verifyMenu = %q, want %q", got, wantVerifyMenu)
	}
}

func assertSimpleFeatureSnapshots(t *testing.T, got []helpers.TableSnapshot, want helpers.TableSnapshot) {
	t.Helper()

	if len(got) != 1 {
		t.Fatalf("table snapshot count = %d, want 1: %#v", len(got), got)
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Fatalf("table snapshot mismatch\ngot:  %#v\nwant: %#v", got[0], want)
	}
}

func captureSimpleFeatureOutput(t *testing.T, invoke func()) (output string) {
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
