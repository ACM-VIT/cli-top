package features

import (
	"bytes"
	"cli-top/internal/nightsliputil"
	"strconv"
	"strings"
	"time"

	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/types"
	"github.com/PuerkitoBio/goquery"
)

func (s Session) simple(name string, o Options) (any, contract.Context, []contract.Warning, error) {
	ctx := contract.Context{Timezone: "Asia/Kolkata"}
	switch name {
	case "events":
		body, err := helpers.FetchReq(s.RegNo, s.Cookies, eventsLoadURL, "", "", "POST", "")
		if err != nil {
			return nil, ctx, nil, err
		}
		doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if doc.Find(eventsUpcomingTableSelector+", "+eventsRegisteredSelector).Length() == 0 && !emptyPage(doc) {
			return nil, ctx, nil, parseFailure("events")
		}
		up, registered, err := ParseEventsHTML(string(body), time.FixedZone("IST", 19800))
		if o.EventID != "" {
			choices := []contract.Choice{}
			for i, event := range up {
				if event.CanRegister {
					choices = append(choices, contract.Choice{ID: event.EventID, Label: event.Title, Scope: "events", Index: i + 1})
				}
			}
			choice, e := helpers.Choose("event", choices, o.EventID, "", 0)
			if e != nil {
				return nil, ctx, nil, e
			}
			if !o.Confirm {
				return nil, ctx, nil, confirmationError()
			}
			for _, event := range up {
				if event.EventID == choice.ID {
					var errorMessage string
					up, registered, _, errorMessage, e = registerForEvent(s.RegNo, s.Cookies, event, time.FixedZone("IST", 19800))
					if e != nil {
						return nil, ctx, nil, contract.Wrap("upstream_error", "Event submission outcome is unknown; check registrations before retrying.", e)
					}
					if errorMessage != "" {
						return nil, ctx, nil, contract.New("upstream_error", "VTOP rejected the event registration.")
					}
					found := false
					for _, v := range registered {
						if v.EventID == choice.ID || (v.EventID == "" && v.Title == event.Title && v.StartDateTime.Equal(event.StartDateTime) && v.Association == event.Association) {
							found = true
						}
					}
					if !found {
						return nil, ctx, nil, contract.New("upstream_error", "Event registration could not be verified; check registrations before retrying.")
					}
					break
				}
			}
		}
		r := types.EventsResult{Upcoming: []types.EventResult{}, Registered: []types.EventResult{}}
		for _, e := range up {
			r.Upcoming = append(r.Upcoming, eventDTO(e))
		}
		for _, e := range registered {
			r.Registered = append(r.Registered, eventDTO(e))
		}
		return r, ctx, nil, err
	case "leave":
		submitted := false
		if o.Apply {
			if err := s.applyLeave(o); err != nil {
				return nil, ctx, nil, err
			}
			submitted = true
		}
		rows, err := fetchLeaveRequests(s.RegNo, s.Cookies)
		r := types.LeaveResult{Requests: []types.LeaveEntry{}, Submitted: submitted}
		for _, v := range rows {
			r.Requests = append(r.Requests, types.LeaveEntry{Place: v.VisitPlace, Reason: v.Reason, Type: v.LeaveType, From: v.From, To: v.To, Status: helpers.StripAnsiCodes(v.Status), RawStatus: v.RawStatus})
		}
		return r, ctx, nil, err
	case "nightslip":
		submitted := false
		if o.Apply {
			if err := s.applyNightslip(o); err != nil {
				return nil, ctx, nil, err
			}
			submitted = true
		}
		rows, err := fetchNightSlipRequests(s.RegNo, s.Cookies)
		r := types.NightslipResult{Requests: []types.NightslipEntry{}, Submitted: submitted}
		for _, v := range rows {
			r.Requests = append(r.Requests, types.NightslipEntry{Venue: v.Venue, EventType: v.EventType, Details: v.Details, AppliedTo: v.AppliedTo, FromDate: v.FromDate, ToDate: v.ToDate, Time: v.FromToTime, Status: helpers.StripAnsiCodes(v.Status), RawStatus: v.RawStatus})
		}
		return r, ctx, nil, err
	case "facility":
		return s.facilities(o)
	}
	path := ""
	selector := ""
	switch name {
	case "cgpa":
		path = "/vtop/examinations/examGradeView/StudentGradeHistory"
		selector = CGPATableSelector
	case "hostel":
		path = "/vtop/studentsRecord/StudentProfileAllView"
		selector = HostelTableSelector
	case "library-dues":
		path = "/vtop/finance/libraryPayments"
		selector = LibraryDuesTableSelector
	case "receipts":
		path = "/vtop/finance/getStudentReceipts"
		selector = ReceiptTableSelector
	case "msg":
		path = "/vtop/academics/common/StudentClassMessage"
		selector = "h5"
	}
	if path == "" {
		return nil, ctx, nil, contract.New("invalid_arguments", "Unknown read command.")
	}
	d, err := s.document(path, "", "", "")
	if err != nil {
		return nil, ctx, nil, err
	}
	if d.Find(selector).Length() == 0 && !emptyPage(d) {
		return nil, ctx, nil, parseFailure(name)
	}
	var parseErr error
	switch name {
	case "cgpa":
		cells := d.Find(selector + " tbody tr").First().Find("td")
		r := types.CGPAResult{Grades: []types.GradeCount{}}
		if cells.Length() < 11 {
			return nil, ctx, nil, parseFailure("CGPA")
		}
		r.CreditsRegistered, err = nullableNumber(cells.Eq(0).Text())
		if err != nil {
			return nil, ctx, nil, err
		}
		r.CreditsEarned, err = nullableNumber(cells.Eq(1).Text())
		if err != nil {
			return nil, ctx, nil, err
		}
		r.CGPA, err = nullableNumber(cells.Eq(2).Text())
		if err != nil {
			return nil, ctx, nil, err
		}
		for i, g := range []string{"S", "A", "B", "C", "D", "E", "F", "N"} {
			n, e := strconv.Atoi(strings.TrimSpace(cells.Eq(i + 3).Text()))
			if e != nil || n < 0 {
				return nil, ctx, nil, parseFailure("grade counts")
			}
			r.Grades = append(r.Grades, types.GradeCount{Grade: g, Count: n})
		}
		return r, ctx, nil, nil
	case "hostel":
		r := types.HostelResult{Fields: []types.FieldValue{}}
		rows := d.Find(selector + " tbody tr")
		start := rows.Length() - 5
		if start < 0 {
			start = 0
		}
		rows.Slice(start, rows.Length()).Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				parseErr = parseFailure("hostel row")
				return
			}
			r.Fields = append(r.Fields, types.FieldValue{Field: strings.TrimSpace(cells.Eq(0).Text()), Value: strings.TrimSpace(cells.Eq(1).Text())})
		})
		return r, ctx, nil, parseErr
	case "library-dues":
		r := types.LibraryResult{Dues: []types.LibraryDue{}}
		d.Find(selector + " tbody tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() == 0 {
				return
			}
			if cells.Length() < 2 {
				if !emptyPage(d) {
					parseErr = parseFailure("library row")
				}
				return
			}
			raw := strings.TrimSpace(cells.Eq(1).Text())
			amount, e := nullableNumber(raw)
			if e != nil {
				parseErr = e
			}
			r.Dues = append(r.Dues, types.LibraryDue{Type: strings.TrimSpace(cells.Eq(0).Text()), Amount: amount, AmountText: raw})
		})
		return r, ctx, nil, parseErr
	case "receipts":
		r := types.ReceiptsResult{Receipts: []types.Receipt{}}
		d.Find(selector + " tbody tr").Each(func(_ int, row *goquery.Selection) {
			if row.Find("th").Length() > 0 {
				return
			}
			cells := row.Find("td")
			if cells.Length() == 0 {
				return
			}
			if cells.Length() < 4 {
				if !emptyPage(d) {
					parseErr = parseFailure("receipt row")
				}
				return
			}
			if strings.EqualFold(strings.TrimSpace(cells.Eq(0).Text()), "Invoice Number") {
				return
			}
			raw := strings.TrimSpace(cells.Eq(3).Text())
			amount, e := nullableNumber(raw)
			if e != nil {
				parseErr = e
			}
			r.Receipts = append(r.Receipts, types.Receipt{InvoiceNumber: strings.TrimSpace(cells.Eq(0).Text()), ReceiptNumber: strings.TrimSpace(cells.Eq(1).Text()), Date: strings.TrimSpace(cells.Eq(2).Text()), Amount: amount, AmountText: raw})
		})
		return r, ctx, nil, parseErr
	case "msg":
		r := types.MessagesResult{Messages: []types.Message{}}
		d.Find("h5").Each(func(_ int, h *goquery.Selection) {
			spans := h.Find("span")
			if spans.Length() != 2 {
				return
			}
			r.Messages = append(r.Messages, types.Message{Course: strings.TrimSpace(spans.Eq(0).Text()), Message: strings.TrimSpace(spans.Eq(1).Text())})
		})
		return r, ctx, nil, nil
	}
	return nil, ctx, nil, contract.New("internal_error", "Unregistered read command.")
}
func eventDTO(v types.Event) types.EventResult {
	date := func(t time.Time) *string {
		if t.IsZero() {
			return nil
		}
		s := t.Format(time.RFC3339)
		return &s
	}
	var id *string
	if v.EventID != "" {
		value := v.EventID
		id = &value
	}
	return types.EventResult{ID: id, Title: v.Title, Association: v.Association, Description: v.Description, Start: date(v.StartDateTime), End: date(v.EndDateTime), RegistrationDeadline: date(v.RegistrationDeadline), Venue: v.Venue, Status: v.RegisterStatus, CanRegister: v.CanRegister}
}
func (s Session) facilities(o Options) (any, contract.Context, []contract.Warning, error) {
	ctx := contract.Context{Timezone: "Asia/Kolkata"}
	r := types.FacilityResult{Facilities: []types.FacilityEntry{}, Registrations: []types.RegistrationEntry{}}
	warnings := []contract.Warning{}
	registrations, e1 := ListRegistrations(s.RegNo, s.Cookies)
	facilities, e2 := fetchAvailableFacilities(s.RegNo, s.Cookies)
	if e1 != nil {
		warnings = append(warnings, contract.Warning{Kind: contract.From(e1).Kind, Message: "Registrations unavailable.", Component: "registrations"})
	}
	if e2 != nil {
		warnings = append(warnings, contract.Warning{Kind: contract.From(e2).Kind, Message: "Facilities unavailable.", Component: "facilities"})
	}
	for _, v := range facilities {
		for _, reg := range registrations {
			if strings.EqualFold(v.Name, reg.FacilityName) {
				v.Registered = true
			}
		}
		fee, _ := nullableNumber(v.Fees)
		r.Facilities = append(r.Facilities, types.FacilityEntry{ID: v.ID, Name: v.Name, Fees: fee, FeesText: v.Fees, SeatsAvailable: v.SeatsAvailable, Registered: v.Registered})
	}
	for _, v := range registrations {
		r.Registrations = append(r.Registrations, types.RegistrationEntry{Name: v.FacilityName, Status: v.StatusMessage, Paid: v.IsPaid})
	}
	if len(warnings) > 0 {
		return r, ctx, warnings, contract.New("partial_failure", "Facility information is incomplete.")
	}
	if o.FacilityID == "" && o.FacilityQuery == "" {
		return r, ctx, warnings, nil
	}
	choices := []contract.Choice{}
	for i, v := range facilities {
		choices = append(choices, contract.Choice{ID: v.ID, Label: v.Name, Scope: "facility", Index: i + 1})
	}
	choice, err := helpers.Choose("facility", choices, o.FacilityID, o.FacilityQuery, 0)
	if err != nil {
		return r, ctx, warnings, err
	}
	check := s.CheckPolicy
	if check == nil {
		check = helpers.CheckKillSwitch
	}
	if check() == 4 {
		return r, ctx, warnings, contract.New("policy_blocked", "Facility registration is currently view-only.")
	}
	if !o.Confirm {
		return r, ctx, warnings, confirmationError()
	}
	for _, v := range facilities {
		if v.ID == choice.ID {
			if v.Registered || v.SeatsAvailable <= 0 {
				return r, ctx, warnings, contract.New("invalid_selection", "Facility is already registered or unavailable.")
			}
			if err := performRegistration(s.RegNo, s.Cookies, v); err != nil {
				return r, ctx, warnings, contract.Wrap("upstream_error", "Facility registration could not be verified.", err)
			}
			r.Submitted = true
			updated, e := ListRegistrations(s.RegNo, s.Cookies)
			if e != nil {
				warnings = append(warnings, contract.Warning{Kind: contract.From(e).Kind, Message: "Registration succeeded but refreshed status is unavailable.", Component: "registrations"})
				return r, ctx, warnings, contract.New("partial_failure", "Registration succeeded; inspect status before submitting again.")
			}
			r.Registrations = []types.RegistrationEntry{}
			for _, reg := range updated {
				r.Registrations = append(r.Registrations, types.RegistrationEntry{Name: reg.FacilityName, Status: reg.StatusMessage, Paid: reg.IsPaid})
			}
			for i := range r.Facilities {
				if r.Facilities[i].ID == choice.ID {
					r.Facilities[i].Registered = true
				}
			}
			return r, ctx, warnings, nil
		}
	}
	return r, ctx, warnings, contract.New("invalid_selection", "Facility unavailable.")
}
func leaveInput(o Options) LeaveApplyInput {
	v := o.Values
	return LeaveApplyInput{Apply: o.Apply, LeaveCode: v["leave-code"], VisitingPlace: v["visiting-place"], Reason: v["reason"], FromDate: v["from-date"], FromTime: v["from-time"], ToDate: v["to-date"], ToTime: v["to-time"]}
}
func (s Session) applyLeave(o Options) error {
	input, err := normalizeLeaveApplyInput(leaveInput(o))
	if err != nil {
		return contract.Wrap("invalid_arguments", "Invalid leave application fields.", err)
	}
	before, err := fetchLeaveRequests(s.RegNo, s.Cookies)
	if err != nil {
		return err
	}
	if hasPendingLeaveRequest(before) {
		return contract.New("policy_blocked", "A leave request is already pending.")
	}
	body, err := fetchLeaveApplyPage(s.RegNo, s.Cookies)
	if err != nil {
		return err
	}
	resolved, err := resolveLeaveApplyInput(body, s.Cookies.CSRF, input)
	if err != nil {
		return contract.Wrap("invalid_selection", "Leave application options are unavailable.", err)
	}
	payload, err := buildLeaveSubmitPayload(s.RegNo, body, resolved)
	if err != nil {
		return err
	}
	_, err = helpers.FetchReq(s.RegNo, s.Cookies, leaveSubmitURL, "", payload.Encode(), "POST", "")
	if err != nil {
		return contract.Wrap("upstream_error", "Submission outcome is unknown; check leave status before retrying.", err)
	}
	after, err := fetchLeaveRequests(s.RegNo, s.Cookies)
	if err != nil {
		return contract.Wrap("partial_failure", "Leave request was submitted; status could not be confirmed. Inspect leave status before submitting again.", err)
	}
	if !wasLeaveSubmitted(before, after, input) {
		return contract.New("upstream_error", "Leave submission could not be verified; check its status before retrying.")
	}
	return nil
}
func nightSlipInput(o Options) NightSlipApplyInput {
	v := o.Values
	return NightSlipApplyInput{Apply: true, CostCentreID: v["cost-centre-id"], AppliedTo: v["applied-to"], RoomTypeID: v["room-type-id"], BuildingID: v["building-id"], Venue: v["venue"], LateHourEventID: v["event-id"], Details: v["details"], FromDate: v["from-date"], FromTime: v["from-time"], ToDate: v["to-date"], ToTime: v["to-time"]}
}

// ValidateOptions checks application inputs before authentication or network work.
func ValidateOptions(name string, o Options) error {
	if !o.Apply {
		return nil
	}
	keys := []string{"from-date", "from-time", "to-date", "to-time"}
	if name == "leave" {
		keys = append(keys, "leave-code", "visiting-place", "reason")
	} else {
		keys = append(keys, "details")
	}
	missing := []string{}
	for _, key := range keys {
		if strings.TrimSpace(o.Values[key]) == "" {
			missing = append(missing, "--"+key)
		}
	}
	if len(missing) > 0 {
		e := contract.New("invalid_arguments", "Application fields are required with --apply.")
		e.RequiredFlags = missing
		return e
	}
	var err error
	if name == "leave" {
		_, err = normalizeLeaveApplyInput(leaveInput(o))
	} else {
		_, err = nightsliputil.NormalizeApplyInput(nightSlipInput(o))
	}
	if err != nil {
		return contract.Wrap("invalid_arguments", err.Error(), err)
	}
	return nil
}
func (s Session) applyNightslip(o Options) error {
	before, err := fetchNightSlipRequests(s.RegNo, s.Cookies)
	if err != nil {
		return err
	}
	if nightsliputil.HasPendingRequest(before) {
		return contract.New("policy_blocked", "A nightslip request is already pending.")
	}
	body, err := fetchNightSlipApplyPage(s.RegNo, s.Cookies)
	if err != nil {
		return err
	}
	input, err := resolveNightSlipApplyInput(body, s.Cookies.CSRF, nightSlipInput(o))
	if err != nil {
		if _, ok := err.(*contract.Error); ok {
			return err
		}
		return contract.Wrap("invalid_selection", "Nightslip application options are unavailable.", err)
	}
	payload, err := buildNightSlipSubmitPayload(s.RegNo, body, input)
	if err != nil {
		return contract.Wrap("invalid_arguments", "Invalid nightslip application fields.", err)
	}
	_, err = helpers.FetchReq(s.RegNo, s.Cookies, nightSlipSubmitURL, "", payload.Encode(), "POST", "")
	if err != nil {
		return contract.Wrap("upstream_error", "Submission outcome is unknown; check nightslip status before retrying.", err)
	}
	after, err := fetchNightSlipRequests(s.RegNo, s.Cookies)
	if err != nil {
		return contract.Wrap("partial_failure", "Nightslip request was submitted; status could not be confirmed. Inspect nightslip status before submitting again.", err)
	}
	if !wasNightSlipSubmitted(before, after, input.NightSlipApplyInput) {
		return contract.New("upstream_error", "Nightslip submission could not be verified; check status before retrying.")
	}
	return nil
}

func confirmationError() *contract.Error {
	e := contract.New("confirmation_required", "Rerun with --confirm to register for the selected item.")
	e.RequiredFlags = []string{"--confirm"}
	return e
}
