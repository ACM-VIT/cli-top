package tests

import (
	"cli-top/features"
	"testing"
	"time"
)

func TestParseEventsHTMLSeparatesUpcomingAndRegisteredEvents(t *testing.T) {
	loc := time.FixedZone("IST", 5*3600+1800)
	html := `
		<table id="dataTable1">
			<tbody>
				<tr>
					<td>1</td>
					<td>Mozilla Firefox Club (CLUB)</td>
					<td>Hack Night</td>
					<td><div class="modal-body">Bring your laptop</div></td>
					<td><span>30-Mar-2026</span><span>30-Mar-2026</span></td>
					<td><span>06:00 PM</span><span>08:00 PM</span></td>
					<td>-</td>
					<td><span>25-Mar-2026</span><span>29-Mar-2026</span></td>
					<td>TT 302 (OFFLINE)</td>
					<td><button onclick="doEventRegistraiton('EVT100')">Register</button></td>
				</tr>
			</tbody>
		</table>
		<table id="dataTable2">
			<tbody>
				<tr>
					<td>1</td>
					<td>Debate Society (CHAPTER)</td>
					<td>Open Mic</td>
					<td><div class="modal-body">Already registered</div></td>
					<td><span>01-Apr-2026</span><span>01-Apr-2026</span></td>
					<td><span>07:00 PM</span><span>09:00 PM</span></td>
					<td>-</td>
					<td><span>20-Mar-2026</span><span>31-Mar-2026</span></td>
					<td>Anna Auditorium (OFFLINE)</td>
					<td></td>
				</tr>
			</tbody>
		</table>
	`

	upcoming, registered, err := features.ParseEventsHTML(html, loc)
	if err != nil {
		t.Fatalf("ParseEventsHTML returned error: %v", err)
	}

	if len(upcoming) != 1 {
		t.Fatalf("expected 1 upcoming event, got %d", len(upcoming))
	}
	if len(registered) != 1 {
		t.Fatalf("expected 1 registered event, got %d", len(registered))
	}

	if upcoming[0].EventID != "EVT100" {
		t.Fatalf("expected upcoming event id EVT100, got %q", upcoming[0].EventID)
	}
	if !upcoming[0].CanRegister || upcoming[0].RegisterStatus != "Open" {
		t.Fatalf("expected upcoming event to be open for registration, got canRegister=%v status=%q", upcoming[0].CanRegister, upcoming[0].RegisterStatus)
	}
	if registered[0].RegisterStatus != "Registered" {
		t.Fatalf("expected registered event status to default to Registered, got %q", registered[0].RegisterStatus)
	}
	if registered[0].Description != "Already registered" {
		t.Fatalf("expected registered event description to be parsed, got %q", registered[0].Description)
	}
}

func TestParseEventRegistrationResultHTMLExtractsMessages(t *testing.T) {
	successMessage, errorMessage, err := features.ParseEventRegistrationResultHTML(`
		<div id="error1"> Registered successfully </div>
		<div id="error2"> Limit exceeded </div>
	`)
	if err != nil {
		t.Fatalf("ParseEventRegistrationResultHTML returned error: %v", err)
	}

	if successMessage != "Registered successfully" {
		t.Fatalf("unexpected success message: %q", successMessage)
	}
	if errorMessage != "Limit exceeded" {
		t.Fatalf("unexpected error message: %q", errorMessage)
	}
}
