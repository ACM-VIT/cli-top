package tests

import (
	"bufio"
	"bytes"
	"cli-top/internal/nightsliputil"
	"cli-top/types"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseNightSlipRequestsStripsRaisedPrefixAndDetectsPending(t *testing.T) {
	body := []byte(`
		<table id="LateHourStatusTable">
			<tbody>
				<tr>
					<td>1</td>
					<td>2</td>
					<td>SMV Portico</td>
					<td>Clubs and Chapters</td>
					<td>Board meeting</td>
					<td>ASWANI KUMAR CHERUKURI</td>
					<td>28-Mar-2026</td>
					<td>29-Mar-2026</td>
					<td>20:30 to 01:00</td>
					<td>REQUEST RAISED-APPROVAL PENDING</td>
				</tr>
			</tbody>
		</table>
	`)

	requests, err := nightsliputil.ParseRequests(body)
	if err != nil {
		t.Fatalf("ParseRequests returned error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	request := requests[0]
	if request.RawStatus != "APPROVAL PENDING" {
		t.Fatalf("expected normalized raw status, got %q", request.RawStatus)
	}
	if request.FromDate != "28/03/26" || request.ToDate != "29/03/26" {
		t.Fatalf("expected dates to be normalized, got from=%q to=%q", request.FromDate, request.ToDate)
	}
	if !nightsliputil.HasPendingRequest(requests) {
		t.Fatal("expected pending request to be detected")
	}
}

func TestHasPendingNightSlipRequestIgnoresApprovedRows(t *testing.T) {
	requests := []types.NightSlipRequest{
		{RawStatus: "APPROVED"},
		{RawStatus: "REJECTED"},
	}

	if nightsliputil.HasPendingRequest(requests) {
		t.Fatal("expected approved and rejected rows to be treated as non-pending")
	}
}

func TestNormalizeNightSlipApplyInputNormalizesDatesAndTimes(t *testing.T) {
	normalized, err := nightsliputil.NormalizeApplyInput(nightsliputil.ApplyInput{
		Details:  "  Board meeting  ",
		FromDate: "28/03/2026",
		FromTime: "20:30",
		ToDate:   "2026-03-29",
		ToTime:   "1:00 am",
	})
	if err != nil {
		t.Fatalf("NormalizeApplyInput returned error: %v", err)
	}

	if normalized.Details != "Board meeting" {
		t.Fatalf("expected sanitized details, got %q", normalized.Details)
	}
	if normalized.FromDate != "2026-03-28" || normalized.ToDate != "2026-03-29" {
		t.Fatalf("expected normalized dates, got from=%q to=%q", normalized.FromDate, normalized.ToDate)
	}
	if normalized.FromTime != "08:30 PM" || normalized.ToTime != "01:00 AM" {
		t.Fatalf("expected normalized times, got from=%q to=%q", normalized.FromTime, normalized.ToTime)
	}
}

func TestResolveNightSlipFieldUsesSelectedOptionAndReturnsHelpfulError(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<form id="LateHourRequestForm">
			<select name="appliedTo">
				<option value="">Select</option>
				<option value="10288" selected>ASWANI KUMAR CHERUKURI</option>
				<option value="20202">ANOTHER WARDEN</option>
			</select>
			<select name="roomTypeId">
				<option value="">Select</option>
				<option value="10">AC</option>
				<option value="11">Non AC</option>
			</select>
		</form>
	`))
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	appliedTo, err := nightsliputil.ResolveField(doc.Selection, "appliedTo", "", "--applied-to")
	if err != nil {
		t.Fatalf("expected selected appliedTo value, got error: %v", err)
	}
	if appliedTo != "10288" {
		t.Fatalf("expected selected option value, got %q", appliedTo)
	}

	_, err = nightsliputil.ResolveField(doc.Selection, "roomTypeId", "", "--room-type-id")
	if err == nil {
		t.Fatal("expected missing room type to return an error with available options")
	}
	if !strings.Contains(err.Error(), "10 (AC)") || !strings.Contains(err.Error(), "11 (Non AC)") {
		t.Fatalf("expected error to include available options, got %q", err.Error())
	}
}

func TestResolveNightSlipFieldInteractivelyPromptsForSelection(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<form id="LateHourRequestForm">
			<select name="roomTypeId">
				<option value="">Select</option>
				<option value="10">AC</option>
				<option value="11">Non AC</option>
			</select>
		</form>
	`))
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	reader := bufio.NewReader(strings.NewReader("2\n"))
	value, err := nightsliputil.ResolveFieldInteractively(reader, doc.Selection, "roomTypeId", "room type")
	if err != nil {
		t.Fatalf("expected interactive selection to succeed, got error: %v", err)
	}
	if value != "11" {
		t.Fatalf("expected second option to be selected, got %q", value)
	}
}

func TestPromptNightSlipOptionSelectionSupportsSearch(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("ash\n1\n"))
	value, err := nightsliputil.PromptOptionSelection(reader, []nightsliputil.FormOption{
		{Value: "10134", Label: "BIMAL KUMAR RAY"},
		{Value: "10288", Label: "ASWANI KUMAR CHERUKURI"},
		{Value: "10295", Label: "MYTHILI N"},
	}, "warden/approver")
	if err != nil {
		t.Fatalf("expected searchable option selection to succeed, got error: %v", err)
	}
	if value != "10288" {
		t.Fatalf("expected fuzzy search selection to return 10288, got %q", value)
	}
}

func TestFilterNightSlipOptionsMatchesValueAndFuzzyLabel(t *testing.T) {
	options := []nightsliputil.FormOption{
		{Value: "135", Label: "SCORE School of Computer Science Engineering and Information Systems"},
		{Value: "77", Label: "SCOPE School of Computer Science and Engineering"},
		{Value: "10", Label: "SMV Portico"},
	}

	displayTable := nightsliputil.BuildOptionTable(options[:1])
	if len(displayTable[0]) != 1 || displayTable[0][0] != "Option" {
		t.Fatalf("expected display table to hide raw values, got %#v", displayTable[0])
	}

	valueMatches := nightsliputil.FilterOptions(options, "135")
	if len(valueMatches) == 0 || valueMatches[0].Value != "135" {
		t.Fatalf("expected value search to prioritize exact id match, got %#v", valueMatches)
	}

	fuzzyMatches := nightsliputil.FilterOptions(options, "smv")
	if len(fuzzyMatches) == 0 || fuzzyMatches[0].Value != "10" {
		t.Fatalf("expected fuzzy search to match SMV Portico, got %#v", fuzzyMatches)
	}
}

func TestExtractNightSlipMessageFindsDataBlockMessage(t *testing.T) {
	message := nightsliputil.ExtractMessage(bytes.NewBufferString(`
		<p style="display: none;" id="DataBlockMessage"> Already a request is pending for Approval</p>
	`).Bytes())

	if message != "Already a request is pending for Approval" {
		t.Fatalf("unexpected message extracted: %q", message)
	}
}
