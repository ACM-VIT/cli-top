package tests

import (
	appfeatures "cli-top/features"
	"net/url"
	"strings"
	"testing"
)

func TestParseLeaveRequestsReadsVTOPStatusTable(t *testing.T) {
	body := []byte(`
		<table id="LeaveAppliedTable">
			<tbody>
				<tr>
					<td style="display:none"></td>
					<td></td>
					<td class="text-primary text-nowrap">Chennai</td>
					<td class="text-primary text-nowrap">Family visit</td>
					<td class="text-primary text-nowrap">HOME TOWN</td>
					<td class="text-primary text-nowrap">23-Apr-2026 08:00</td>
					<td class="text-primary text-nowrap">24-Apr-2026 20:00</td>
					<td class="text-primary text-nowrap">REQUEST RAISED-PENDING</td>
				</tr>
			</tbody>
		</table>
	`)

	got, err := appfeatures.ParseLeaveRequests(body)
	if err != nil {
		t.Fatalf("ParseLeaveRequests returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 leave request, got %d", len(got))
	}
	if got[0].VisitPlace != "Chennai" || got[0].Reason != "Family visit" || got[0].LeaveType != "HOME TOWN" {
		t.Fatalf("unexpected leave request: %+v", got[0])
	}
	if got[0].From != "23/04/26" || got[0].To != "24/04/26" {
		t.Fatalf("expected formatted dates, got from=%q to=%q", got[0].From, got[0].To)
	}
	if got[0].RawStatus != "REQUEST RAISED-PENDING" {
		t.Fatalf("expected raw status, got %+v", got[0])
	}
}

func TestNormalizeLeaveApplyInputNormalizesDatesAndTimes(t *testing.T) {
	got, err := appfeatures.NormalizeLeaveApplyInput(appfeatures.LeaveApplyInput{
		LeaveCode:     " HT1 ",
		VisitingPlace: " Chennai ",
		Reason:        " Family visit ",
		FromDate:      "23/04/2026",
		FromTime:      "8:30 PM",
		ToDate:        "2026-04-24",
		ToTime:        "06:15",
	})
	if err != nil {
		t.Fatalf("NormalizeLeaveApplyInput returned error: %v", err)
	}

	if got.FromDate != "2026-04-23" || got.ToDate != "2026-04-24" {
		t.Fatalf("unexpected dates: %+v", got)
	}
	if got.FromTime != "20:30" || got.ToTime != "06:15" {
		t.Fatalf("unexpected times: %+v", got)
	}
}

func TestNormalizeLeaveApplyInputRejectsUnsafeTextAndBadTimeWindow(t *testing.T) {
	_, err := appfeatures.NormalizeLeaveApplyInput(appfeatures.LeaveApplyInput{
		LeaveCode:     "HT1",
		VisitingPlace: "<script>",
		Reason:        "Family visit",
		FromDate:      "2026-04-23",
		FromTime:      "20:30",
		ToDate:        "2026-04-24",
		ToTime:        "06:15",
	})
	if err == nil || !strings.Contains(err.Error(), "visiting place cannot contain") {
		t.Fatalf("expected unsafe visiting place to be rejected, got %v", err)
	}

	_, err = appfeatures.NormalizeLeaveApplyInput(appfeatures.LeaveApplyInput{
		LeaveCode:     "HT1",
		VisitingPlace: "Chennai",
		Reason:        "Family visit",
		FromDate:      "2026-04-24",
		FromTime:      "20:30",
		ToDate:        "2026-04-23",
		ToTime:        "06:15",
	})
	if err == nil || !strings.Contains(err.Error(), "--to-date must be on or after --from-date") {
		t.Fatalf("expected invalid date range to be rejected, got %v", err)
	}
}

func TestResolveLeaveApplyInputRejectsUnknownLeaveCode(t *testing.T) {
	body := leaveApplyFormFixture()

	_, err := appfeatures.ResolveLeaveApplyInput(body, "fallback", appfeatures.LeaveApplyInput{
		LeaveCode:     "BAD",
		VisitingPlace: "Chennai",
		Reason:        "Family visit",
		FromDate:      "2026-04-23",
		FromTime:      "20:30",
		ToDate:        "2026-04-24",
		ToTime:        "06:15",
	})
	if err == nil || !strings.Contains(err.Error(), `invalid --leave-code "BAD"`) {
		t.Fatalf("expected invalid leave code error, got %v", err)
	}
}

func TestResolveLeaveApplyInputAcceptsLeaveCodeLabel(t *testing.T) {
	body := leaveApplyFormFixture()

	got, err := appfeatures.ResolveLeaveApplyInput(body, "fallback", appfeatures.LeaveApplyInput{
		LeaveCode:     "home town",
		VisitingPlace: "Chennai",
		Reason:        "Family visit",
		FromDate:      "2026-04-23",
		FromTime:      "20:30",
		ToDate:        "2026-04-24",
		ToTime:        "06:15",
	})
	if err != nil {
		t.Fatalf("ResolveLeaveApplyInput returned error: %v", err)
	}
	if got.LeaveCode != "HT1" {
		t.Fatalf("expected label to resolve to HT1, got %q", got.LeaveCode)
	}
}

func TestBuildLeaveSubmitPayloadCopiesDefaultsAndSetsSubmitFields(t *testing.T) {
	body := leaveApplyFormFixture()

	payload, err := appfeatures.BuildLeaveSubmitPayload("23BME0453", body, "csrf-from-form", appfeatures.LeaveApplyInput{
		LeaveCode:     "HT1",
		VisitingPlace: "Chennai",
		Reason:        "Family visit",
		FromDate:      "2026-04-23",
		FromTime:      "20:30",
		ToDate:        "2026-04-24",
		ToTime:        "06:15",
	})
	if err != nil {
		t.Fatalf("BuildLeaveSubmitPayload returned error: %v", err)
	}

	want := url.Values{
		"_csrf":         {"csrf-from-form"},
		"authorizedID":  {"23BME0453"},
		"leaveCode":     {"HT1"},
		"visitingPlace": {"Chennai"},
		"reason":        {"Family visit"},
		"leaveFromDate": {"2026-04-23"},
		"fromTime":      {"20:30"},
		"leaveToDate":   {"2026-04-24"},
		"toTime":        {"06:15"},
		"form":          {"LeaveRequestForm"},
		"control":       {"submitControl2"},
	}

	for key, values := range want {
		if payload.Get(key) != values[0] {
			t.Fatalf("expected %s=%q, got %q in %s", key, values[0], payload.Get(key), payload.Encode())
		}
	}
	if !strings.Contains(payload.Get("x"), "GMT") {
		t.Fatalf("expected x to be an HTTP-date, got %q", payload.Get("x"))
	}
}

func leaveApplyFormFixture() []byte {
	return []byte(`
		<form id="LeaveRequestForm" name="LeaveRequestForm">
			<input type="hidden" name="_csrf" value="csrf-from-form" />
			<select name="leaveCode">
				<option value="">-Select-</option>
				<option value="HT1">HOME TOWN</option>
			</select>
			<input name="visitingPlace" value="" />
			<textarea name="reason"></textarea>
		</form>
	`)
}
