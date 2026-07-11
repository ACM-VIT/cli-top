package tests

import (
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

const interactiveFeatureRegNo = "24BCE0001"

var interactiveFeatureCookies = types.Cookies{
	SERVERID:   "fake-server",
	CSRF:       "fake-cookie-csrf",
	JSESSIONID: "fake-session",
}

type workflowRequest struct {
	method      string
	path        string
	contentType string
	cookie      string
	form        url.Values
}

type workflowResponder func(*workflowTransport, workflowRequest) (string, error)

// workflowTransport is a stateful, in-memory stand-in for VTOP. Mutating
// workflows only change submitted; subsequent status responses read that state.
type workflowTransport struct {
	requests  []workflowRequest
	submitted bool
	responder workflowResponder
	errors    []error
}

func (transport *workflowTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	record, err := recordWorkflowRequest(request)
	if err != nil {
		transport.errors = append(transport.errors, err)
		return nil, err
	}
	transport.requests = append(transport.requests, record)

	body, err := transport.responder(transport, record)
	if err != nil {
		transport.errors = append(transport.errors, err)
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}, nil
}

func recordWorkflowRequest(request *http.Request) (workflowRequest, error) {
	form := url.Values{}
	if request.Body != nil {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			return workflowRequest{}, fmt.Errorf("read %s request body: %w", request.URL.Path, err)
		}
		form, err = url.ParseQuery(string(body))
		if err != nil {
			return workflowRequest{}, fmt.Errorf("parse %s request body: %w", request.URL.Path, err)
		}
	}

	return workflowRequest{
		method:      request.Method,
		path:        request.URL.Path,
		contentType: request.Header.Get("Content-Type"),
		cookie:      request.Header.Get("Cookie"),
		form:        form,
	}, nil
}

func TestInteractiveFeaturesEndToEnd(t *testing.T) {
	t.Run("events select and register", testEventsWorkflow)
	t.Run("facility select and register", testFacilityWorkflow)
	t.Run("course allocation select and display", testCourseAllocationWorkflow)
	t.Run("leave submit and verify", testLeaveWorkflow)
	t.Run("nightslip submit and verify", testNightSlipWorkflow)
}

func testEventsWorkflow(t *testing.T) {
	t.Setenv("CLI_TOP_PROXY_MODE", "")
	transport := installSharedWorkflowTransport(t, eventsWorkflowResponse)

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	previousStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = previousStdin
		_ = writer.Close()
		_ = reader.Close()
	})
	if _, err := io.WriteString(writer, "1\n"); err != nil {
		t.Fatalf("seed event selection: %v", err)
	}

	var tables []helpers.TableSnapshot
	confirmationSent := false
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
		if !confirmationSent && len(snapshot.Headers) > 0 && snapshot.Headers[0] == "EVENT" {
			confirmationSent = true
			if _, err := io.WriteString(writer, "yes\n"); err != nil {
				t.Errorf("send event confirmation: %v", err)
			}
		}
	})
	t.Cleanup(restoreCapture)

	features.GetEvents(interactiveFeatureRegNo, interactiveFeatureCookies)

	assertWorkflowTransportClean(t, transport)
	if !transport.submitted {
		t.Fatal("event registration did not update mock VTOP state")
	}
	load := requireWorkflowRequest(t, transport, "/vtop/event/swf/loadEventRegistration", 1)
	assertCommonWorkflowRequest(t, load, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	if load.form.Get("verifyMenu") != "true" || load.form.Get("nocache") == "" {
		t.Fatalf("event load form = %s, want verifyMenu and nocache", load.form.Encode())
	}
	register := requireWorkflowRequest(t, transport, "/vtop/event/swf/registered/doEventRegistraiton", 1)
	assertCommonWorkflowRequest(t, register, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	if register.form.Get("eventId") != "EVT-E2E-1" || register.form.Get("x") == "" {
		t.Fatalf("event registration form = %s, want selected event and timestamp", register.form.Encode())
	}
	assertTableRow(t, tables, "EVENT", []string{"Mock Hack Night"})
}

func testFacilityWorkflow(t *testing.T) {
	originalLatestJSONURL := helpers.GetLatestJSONURL()
	t.Cleanup(func() { helpers.SetLatestJSONURL(originalLatestJSONURL) })
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	t.Setenv("CLI_TOP_LATEST_JSON_URL", "https://mock.cli-top.test/latest.json")
	helpers.SetLatestJSONURL(originalLatestJSONURL)
	transport := installSharedWorkflowTransport(t, facilityWorkflowResponse)

	previousDefaultTransport := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previousDefaultTransport })

	var tables []helpers.TableSnapshot
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
	})
	t.Cleanup(restoreCapture)

	if err := features.RegisterPhyFacility(interactiveFeatureRegNo, interactiveFeatureCookies, "Basketball", true); err != nil {
		t.Fatalf("RegisterPhyFacility returned error: %v", err)
	}

	assertWorkflowTransportClean(t, transport)
	if !transport.submitted {
		t.Fatal("facility registration did not update mock VTOP state")
	}
	latest := requireWorkflowRequest(t, transport, "/latest.json", 1)
	if latest.method != http.MethodGet {
		t.Fatalf("latest.json method = %q, want GET", latest.method)
	}
	available := requireWorkflowRequest(t, transport, "/vtop/phyedu/facilityAvailable", 3)
	assertCommonWorkflowRequest(t, available, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	registration := requireWorkflowRequest(t, transport, "/vtop/phyedu/PhyFacilityProcessRegistration", 1)
	assertCommonWorkflowRequest(t, registration, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	if registration.form.Get("facilityId") != "1" || registration.form.Get("miscId") != "9876" {
		t.Fatalf("facility registration form = %s, want facilityId=1 and miscId=9876", registration.form.Encode())
	}
	assertTableRow(t, tables, "NO.", []string{"Basketball", "Registered (Paid)"})
}

func testCourseAllocationWorkflow(t *testing.T) {
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	transport := installSharedWorkflowTransport(t, courseAllocationWorkflowResponse)

	var tables []helpers.TableSnapshot
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
	})
	t.Cleanup(restoreCapture)

	features.ExecuteInteractiveCourseAllocationView(
		interactiveFeatureRegNo,
		interactiveFeatureCookies,
		features.DefaultCourseAllocationPageURL,
		"1",
		"1",
	)

	assertWorkflowTransportClean(t, transport)
	initial := requireWorkflowRequest(t, transport, "/vtop/academics/common/StudentRegistrationScheduleAllocation", 1)
	assertCommonWorkflowRequest(t, initial, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	if initial.form.Get("verifyMenu") != "true" || initial.form.Get("nocache") == "" {
		t.Fatalf("course allocation initial form = %s, want verifyMenu and nocache", initial.form.Encode())
	}
	courses := requireWorkflowRequest(t, transport, "/vtop/academics/common/getCoursesListForCurriculmCategory", 1)
	assertCommonWorkflowRequest(t, courses, "PAGE-AUTH", "page-csrf")
	if courses.form.Get("cccategory") != "CORE" {
		t.Fatalf("course list form = %s, want selected category CORE", courses.form.Encode())
	}
	details := requireWorkflowRequest(t, transport, "/vtop/academics/common/getCoursesDetailForRegistration", 1)
	assertCommonWorkflowRequest(t, details, "PAGE-AUTH", "page-csrf")
	if details.form.Get("courseCode") != "CSE2001" {
		t.Fatalf("course detail form = %s, want courseCode=CSE2001", details.form.Encode())
	}

	want := helpers.TableSnapshot{
		Headers: []string{"FACULTY", "VENUE", "SLOT", "TYPE"},
		Rows:    [][]string{{"Dr Ada Lovelace", "SJT-101", "A1", "Theory"}},
	}
	if len(tables) != 1 || !reflect.DeepEqual(tables[0], want) {
		t.Fatalf("course allocation tables = %#v, want %#v", tables, []helpers.TableSnapshot{want})
	}
}

func testLeaveWorkflow(t *testing.T) {
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	transport := installSharedWorkflowTransport(t, leaveWorkflowResponse)

	var tables []helpers.TableSnapshot
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
	})
	t.Cleanup(restoreCapture)

	input := features.LeaveApplyInput{
		Apply:         true,
		LeaveCode:     "HOME TOWN",
		VisitingPlace: "Chennai",
		Reason:        "Family visit",
		FromDate:      "2027-01-02",
		FromTime:      "08:30 PM",
		ToDate:        "2027-01-03",
		ToTime:        "06:15",
	}
	if err := features.ExecuteLeave(interactiveFeatureRegNo, interactiveFeatureCookies, input); err != nil {
		t.Fatalf("ExecuteLeave returned error: %v", err)
	}

	assertWorkflowTransportClean(t, transport)
	if !transport.submitted {
		t.Fatal("leave submission did not update mock VTOP state")
	}
	requireWorkflowRequest(t, transport, "/vtop/hostels/student/leave/1", 2)
	requireWorkflowRequest(t, transport, "/vtop/hostels/student/leave/4", 2)
	requireWorkflowRequest(t, transport, "/vtop/hostels/student/leave/2", 1)
	submit := requireWorkflowRequest(t, transport, "/vtop/hostels/student/leave/3", 1)
	assertCommonWorkflowRequest(t, submit, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	assertFormValues(t, submit.form, map[string]string{
		"leaveCode":     "HT1",
		"visitingPlace": "Chennai",
		"reason":        "Family visit",
		"leaveFromDate": "2027-01-02",
		"fromTime":      "20:30",
		"leaveToDate":   "2027-01-03",
		"toTime":        "06:15",
		"campusCode":    "VLR",
		"form":          "LeaveRequestForm",
		"control":       "submitControl2",
	})
	if submit.form.Get("x") == "" {
		t.Fatal("leave submit form is missing its HTTP timestamp")
	}
	assertTableRow(t, tables, "VISIT PLACE", []string{"Chennai", "Family visit", "PENDING"})
}

func testNightSlipWorkflow(t *testing.T) {
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	transport := installSharedWorkflowTransport(t, nightSlipWorkflowResponse)

	var tables []helpers.TableSnapshot
	restoreCapture := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tables = append(tables, snapshot)
	})
	t.Cleanup(restoreCapture)

	input := features.NightSlipApplyInput{
		Apply:           true,
		CostCentreID:    "CC-1",
		AppliedTo:       "WARDEN-7",
		RoomTypeID:      "ACADEMIC",
		BuildingID:      "SJT",
		Venue:           "SJT Portico",
		LateHourEventID: "CLUB",
		Details:         "ACM board meeting",
		FromDate:        "2027-01-02",
		FromTime:        "20:30",
		ToDate:          "2027-01-02",
		ToTime:          "22:00",
	}
	if err := features.ExecuteNightSlip(interactiveFeatureRegNo, interactiveFeatureCookies, input); err != nil {
		t.Fatalf("ExecuteNightSlip returned error: %v", err)
	}

	assertWorkflowTransportClean(t, transport)
	if !transport.submitted {
		t.Fatal("nightslip submission did not update mock VTOP state")
	}
	requireWorkflowRequest(t, transport, "/vtop/hostels/late/hour/student/request/1", 2)
	requireWorkflowRequest(t, transport, "/vtop/hostels/late/hour/student/request/9", 2)
	requireWorkflowRequest(t, transport, "/vtop/hostels/late/hour/student/request/2", 1)
	submit := requireWorkflowRequest(t, transport, "/vtop/hostels/late/hour/student/request/8", 1)
	assertCommonWorkflowRequest(t, submit, interactiveFeatureRegNo, interactiveFeatureCookies.CSRF)
	assertFormValues(t, submit.form, map[string]string{
		"costCentreId":     "CC-1",
		"appliedTo":        "WARDEN-7",
		"roomTypeId":       "ACADEMIC",
		"buildingId":       "SJT",
		"venue":            "SJT Portico",
		"lateHourEventId":  "CLUB",
		"details":          "ACM board meeting",
		"lateHourFromDate": "2027-01-02",
		"fromTime":         "08:30 PM",
		"lateHourToDate":   "2027-01-02",
		"toTime":           "10:00 PM",
		"hostelId":         "HOSTEL-1",
		"form":             "LateHourRequestForm",
		"control":          "submitButton1",
	})
	if submit.form.Get("x") == "" {
		t.Fatal("nightslip submit form is missing its HTTP timestamp")
	}
	assertTableRow(t, tables, "VENUE", []string{"SJT Portico", "ACM board meeting", "APPROVAL PENDING"})
}

func installSharedWorkflowTransport(t *testing.T, responder workflowResponder) *workflowTransport {
	t.Helper()
	transport := &workflowTransport{responder: responder}
	client := helpers.GetHTTPClient()
	previousTransport := client.Transport
	client.Transport = transport
	t.Cleanup(func() { client.Transport = previousTransport })
	return transport
}

func assertWorkflowTransportClean(t *testing.T, transport *workflowTransport) {
	t.Helper()
	if len(transport.errors) != 0 {
		t.Fatalf("mock transport errors: %v", transport.errors)
	}
}

func requireWorkflowRequest(t *testing.T, transport *workflowTransport, path string, wantCount int) workflowRequest {
	t.Helper()
	var matches []workflowRequest
	for _, request := range transport.requests {
		if request.path == path {
			matches = append(matches, request)
		}
	}
	if len(matches) != wantCount {
		t.Fatalf("request count for %s = %d, want %d; all requests: %#v", path, len(matches), wantCount, transport.requests)
	}
	return matches[len(matches)-1]
}

func assertCommonWorkflowRequest(t *testing.T, request workflowRequest, regNo string, csrf string) {
	t.Helper()
	if request.method != http.MethodPost {
		t.Errorf("%s method = %q, want POST", request.path, request.method)
	}
	if request.contentType != "application/x-www-form-urlencoded" {
		t.Errorf("%s Content-Type = %q, want application/x-www-form-urlencoded", request.path, request.contentType)
	}
	if request.form.Get("authorizedID") != regNo {
		t.Errorf("%s authorizedID = %q, want %q", request.path, request.form.Get("authorizedID"), regNo)
	}
	if request.form.Get("_csrf") != csrf {
		t.Errorf("%s _csrf = %q, want %q", request.path, request.form.Get("_csrf"), csrf)
	}
	for _, cookie := range []string{"JSESSIONID=fake-session", "SERVERID=fake-server"} {
		if !strings.Contains(request.cookie, cookie) {
			t.Errorf("%s Cookie = %q, want %q", request.path, request.cookie, cookie)
		}
	}
}

func assertFormValues(t *testing.T, form url.Values, want map[string]string) {
	t.Helper()
	for key, value := range want {
		if form.Get(key) != value {
			t.Errorf("form[%q] = %q, want %q; form: %s", key, form.Get(key), value, form.Encode())
		}
	}
}

func assertTableRow(t *testing.T, tables []helpers.TableSnapshot, firstHeader string, cells []string) {
	t.Helper()
	for _, table := range tables {
		if len(table.Headers) == 0 || table.Headers[0] != firstHeader {
			continue
		}
		for _, row := range table.Rows {
			joined := helpers.StripAnsiCodes(strings.Join(row, " | "))
			matched := true
			for _, cell := range cells {
				if !strings.Contains(joined, cell) {
					matched = false
					break
				}
			}
			if matched {
				return
			}
		}
	}
	t.Fatalf("no %q table row contained %q; tables: %#v", firstHeader, cells, tables)
}

func eventsWorkflowResponse(transport *workflowTransport, request workflowRequest) (string, error) {
	if request.method != http.MethodPost {
		return "", fmt.Errorf("events: unexpected method %s for %s", request.method, request.path)
	}
	switch request.path {
	case "/vtop/event/swf/loadEventRegistration":
		return eventPageHTML(false), nil
	case "/vtop/event/swf/registered/doEventRegistraiton":
		transport.submitted = true
		return eventPageHTML(true), nil
	default:
		return "", fmt.Errorf("events: unexpected request path %s", request.path)
	}
}

func eventPageHTML(registered bool) string {
	status := `<button onclick="doEventRegistraiton('EVT-E2E-1')">Register</button>`
	tableID := "dataTable1"
	message := ""
	if registered {
		status = "Registered"
		tableID = "dataTable2"
		message = `<div id="error1">Registered successfully</div>`
	}
	return fmt.Sprintf(`%s
		<table id="%s"><tbody><tr>
			<td>1</td><td>ACM-VIT (CHAPTER)</td><td>Mock Hack Night</td>
			<td><div class="modal-body">Build and ship a tiny CLI.</div></td>
			<td><span>31-Dec-2099</span><span>31-Dec-2099</span></td>
			<td><span>06:00 PM</span><span>09:00 PM</span></td>
			<td>-</td><td><span>01-Jan-2027</span><span>30-Dec-2099</span></td>
			<td>SJT 101 (OFFLINE)</td><td>%s</td>
		</tr></tbody></table>`, message, tableID, status)
}

func facilityWorkflowResponse(transport *workflowTransport, request workflowRequest) (string, error) {
	switch request.path {
	case "/latest.json":
		if request.method != http.MethodGet {
			return "", fmt.Errorf("facility: latest.json method %s", request.method)
		}
		return `{"version":"test","killSwitch":0}`, nil
	case "/vtop/phyedu/facilityAvailable":
		if request.method != http.MethodPost {
			return "", fmt.Errorf("facility: availability method %s", request.method)
		}
		return facilityPageHTML(transport.submitted), nil
	case "/vtop/phyedu/PhyFacilityProcessRegistration":
		if request.method != http.MethodPost {
			return "", fmt.Errorf("facility: registration method %s", request.method)
		}
		transport.submitted = true
		return facilityPageHTML(true), nil
	default:
		return "", fmt.Errorf("facility: unexpected request path %s", request.path)
	}
}

func facilityPageHTML(registered bool) string {
	registrationRow := ""
	if registered {
		registrationRow = `<tr><td>Basketball</td><td>Paid</td></tr>`
	}
	return fmt.Sprintf(`
		<table class="table-bordered table-hover table-stripped dataTable"><tbody>
			<tr><td>Facility Name</td><td>Fees</td><td>Seats</td><td>Action</td></tr>
			<tr><td>Basketball</td><td>1180</td><td>4</td><td><button onclick="registerNow('1','9876')">Register</button></td></tr>
		</tbody></table>
		<div class="panel-heading panel-head-custom">My Registration(s)</div>
		<div class="box-body"><table class="dataTable"><tbody>%s</tbody></table></div>`, registrationRow)
}

func courseAllocationWorkflowResponse(_ *workflowTransport, request workflowRequest) (string, error) {
	if request.method != http.MethodPost {
		return "", fmt.Errorf("course allocation: unexpected method %s for %s", request.method, request.path)
	}
	switch request.path {
	case "/vtop/academics/common/StudentRegistrationScheduleAllocation":
		return `<html><body>
			<script>function getCoursesListForCurriculmCategory(){}; var csrfValue = "page-csrf"; var id="PAGE-AUTH";</script>
			<select id="curriculumCategory"><option value="CORE">Programme Core</option></select>
			<div>` + strings.Repeat("mock allocation page padding ", 30) + `</div>
		</body></html>`, nil
	case "/vtop/academics/common/getCoursesListForCurriculmCategory":
		return `<select id="courseId"><option value="CSE2001">CSE2001 - Data Structures</option></select>`, nil
	case "/vtop/academics/common/getCoursesDetailForRegistration":
		return `<div id="courseDetailFragement"><table class="table-bordered"><tbody>
			<tr><td>A1</td><td>SJT-101</td><td>Dr Ada Lovelace</td><td>Theory</td></tr>
		</tbody></table></div>`, nil
	default:
		return "", fmt.Errorf("course allocation: unexpected request path %s", request.path)
	}
}

func leaveWorkflowResponse(transport *workflowTransport, request workflowRequest) (string, error) {
	if request.method != http.MethodPost {
		return "", fmt.Errorf("leave: unexpected method %s for %s", request.method, request.path)
	}
	switch request.path {
	case "/vtop/hostels/student/leave/1":
		return `<div id="menu">Leave menu opened</div>`, nil
	case "/vtop/hostels/student/leave/2":
		return `<form id="LeaveRequestForm" name="LeaveRequestForm">
			<input type="hidden" name="_csrf" value="leave-form-csrf">
			<input type="hidden" name="campusCode" value="VLR">
			<select name="leaveCode"><option value="HT1">HOME TOWN</option></select>
		</form>`, nil
	case "/vtop/hostels/student/leave/3":
		transport.submitted = true
		return `<div id="DataBlockMessage">Leave request submitted successfully.</div>`, nil
	case "/vtop/hostels/student/leave/4":
		if !transport.submitted {
			return `<table id="LeaveAppliedTable"><tbody></tbody></table>`, nil
		}
		return `<table id="LeaveAppliedTable"><tbody><tr>
			<td></td><td></td><td>Chennai</td><td>Family visit</td><td>HOME TOWN</td>
			<td>02-Jan-2027 20:30</td><td>03-Jan-2027 06:15</td><td>REQUEST RAISED-PENDING</td>
		</tr></tbody></table>`, nil
	default:
		return "", fmt.Errorf("leave: unexpected request path %s", request.path)
	}
}

func nightSlipWorkflowResponse(transport *workflowTransport, request workflowRequest) (string, error) {
	if request.method != http.MethodPost {
		return "", fmt.Errorf("nightslip: unexpected method %s for %s", request.method, request.path)
	}
	switch request.path {
	case "/vtop/hostels/late/hour/student/request/1":
		return `<div id="menu">Nightslip menu opened</div>`, nil
	case "/vtop/hostels/late/hour/student/request/2":
		return `<form id="LateHourRequestForm" name="LateHourRequestForm">
			<input type="hidden" name="_csrf" value="nightslip-form-csrf">
			<input type="hidden" name="hostelId" value="HOSTEL-1">
		</form>`, nil
	case "/vtop/hostels/late/hour/student/request/8":
		transport.submitted = true
		return `<div id="DataBlockMessage">Nightslip request submitted successfully.</div>`, nil
	case "/vtop/hostels/late/hour/student/request/9":
		if !transport.submitted {
			return `<table id="LateHourStatusTable"><tbody></tbody></table>`, nil
		}
		return `<table id="LateHourStatusTable"><tbody><tr>
			<td></td><td></td><td>SJT Portico</td><td>Clubs and Chapters</td>
			<td>ACM board meeting</td><td>Dr Warden</td><td>02-Jan-2027</td><td>02-Jan-2027</td>
			<td>20:30 to 22:00</td><td>REQUEST RAISED-APPROVAL PENDING</td>
		</tr></tbody></table>`, nil
	default:
		return "", fmt.Errorf("nightslip: unexpected request path %s", request.path)
	}
}
