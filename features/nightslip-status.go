package features

import (
	"bufio"
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/internal/nightsliputil"
	"cli-top/types"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	nightSlipMenuURL   = "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/1"
	nightSlipApplyURL  = "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/2"
	nightSlipSubmitURL = "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/8"
	nightSlipStatusURL = "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/9"
)

type NightSlipApplyInput = nightsliputil.ApplyInput

type resolvedNightSlipApplyInput struct {
	NightSlipApplyInput
	CSRF string
}

var errNightSlipUserCanceled = nightsliputil.ErrUserCanceled

var nightSlipControlURLs = map[string]string{
	"costCentreId":    "hostels/late/hour/student/request/3",
	"roomTypeId":      "hostels/late/hour/student/request/4",
	"buildingId":      "hostels/late/hour/student/request/5",
	"lateHourEventId": "hostels/late/hour/student/request/6",
	"fromTime":        "hostels/late/hour/student/request/7",
}

func ExecuteNightSlip(regNo string, cookies types.Cookies, input NightSlipApplyInput) error {
	if !helpers.ValidateLogin(cookies) {
		return nil
	}

	if !input.Apply {
		GetNightSlipStatus(regNo, cookies)
		return nil
	}

	return ApplyNightSlip(regNo, cookies, input)
}

func RunInteractiveNightSlip(regNo string, cookies types.Cookies) error {
	if !helpers.ValidateLogin(cookies) {
		return nil
	}

	existingRequests, err := fetchNightSlipRequests(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error fetching night slip status:", err)
		}
		helpers.Println("Unable to fetch night slip status.")
		return err
	}

	printNightSlipRequests(existingRequests)

	if nightsliputil.HasPendingRequest(existingRequests) {
		helpers.Println("A nightslip request is already pending approval. New applications are blocked until it clears.")
		return nil
	}

	if !helpers.IsInteractiveOutput(os.Stdin) {
		helpers.Println("No pending nightslip requests. Run this command in an interactive terminal to submit a new one.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	proceed, err := promptNightSlipConfirmation(reader, "No pending nightslip requests. Apply for a new one? (yes/no): ")
	if err != nil {
		if errors.Is(err, errNightSlipUserCanceled) {
			helpers.Println("Nightslip application canceled.")
			return nil
		}
		return err
	}
	if !proceed {
		helpers.Println("Nightslip application canceled.")
		return nil
	}

	applyBody, err := fetchNightSlipApplyPage(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error opening night slip apply form:", err)
		}
		helpers.Println("Unable to open the night slip application form.")
		return err
	}

	if message := nightsliputil.ExtractMessage(applyBody); message != "" && strings.Contains(strings.ToUpper(message), "PENDING") {
		helpers.Println(message)
		return fmt.Errorf(message)
	}

	input, err := promptNightSlipApplyInput(reader, regNo, cookies, applyBody)
	if err != nil {
		if errors.Is(err, errNightSlipUserCanceled) {
			helpers.Println("Nightslip application canceled.")
			return nil
		}
		helpers.Println(err.Error())
		return err
	}

	return ApplyNightSlip(regNo, cookies, input)
}

func GetNightSlipStatus(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	nightSlipRequests, err := fetchNightSlipRequests(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error fetching night slip status:", err)
		}
		return
	}

	printNightSlipRequests(nightSlipRequests)
}

func ApplyNightSlip(regNo string, cookies types.Cookies, input NightSlipApplyInput) error {
	existingRequests, err := fetchNightSlipRequests(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error preparing night slip request:", err)
		}
		return err
	}

	if nightsliputil.HasPendingRequest(existingRequests) {
		helpers.Println()
		helpers.Println("A nightslip request is already pending approval. Clear it before applying for another one.")
		printNightSlipRequests(existingRequests)
		return fmt.Errorf("nightslip request already pending")
	}

	applyBody, err := fetchNightSlipApplyPage(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error opening night slip apply form:", err)
		}
		return err
	}

	if message := nightsliputil.ExtractMessage(applyBody); message != "" && strings.Contains(strings.ToUpper(message), "PENDING") {
		helpers.Println()
		helpers.Println(message)
		return fmt.Errorf(message)
	}

	resolvedInput, err := resolveNightSlipApplyInput(applyBody, cookies.CSRF, input)
	if err != nil {
		helpers.Println()
		helpers.Println(err.Error())
		return err
	}

	submitPayload, err := buildNightSlipSubmitPayload(regNo, applyBody, resolvedInput)
	if err != nil {
		helpers.Println()
		helpers.Println(err.Error())
		return err
	}

	submitBody, err := helpers.FetchReq(regNo, cookies, nightSlipSubmitURL, "", submitPayload.Encode(), http.MethodPost, "")
	if err != nil {
		if debug.Debug {
			helpers.Println("Error submitting night slip request:", err)
		}
		return err
	}

	submitMessage := nightsliputil.ExtractMessage(submitBody)
	updatedRequests, updatedErr := fetchNightSlipRequests(regNo, cookies)
	if updatedErr == nil && wasNightSlipSubmitted(existingRequests, updatedRequests, resolvedInput.NightSlipApplyInput) {
		helpers.Println()
		if submitMessage != "" {
			helpers.Println(submitMessage)
		} else {
			helpers.Println("Nightslip request submitted successfully.")
		}
		printNightSlipRequests(updatedRequests)
		return nil
	}

	if submitMessage != "" {
		helpers.Println()
		helpers.Println(submitMessage)
	}

	if updatedErr != nil {
		return updatedErr
	}

	return fmt.Errorf("unable to verify nightslip submission")
}

func promptNightSlipApplyInput(reader *bufio.Reader, regNo string, cookies types.Cookies, body []byte) (NightSlipApplyInput, error) {
	scope, err := parseNightSlipScope(body)
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	input := NightSlipApplyInput{Apply: true}

	input.CostCentreID, err = nightsliputil.ResolveFieldInteractively(reader, scope, "costCentreId", "cost centre")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	facultyScope, err := fetchNightSlipControlScope(scope, regNo, cookies, input, "costCentreId")
	if err != nil {
		return NightSlipApplyInput{}, fmt.Errorf("unable to load faculty options: %w", err)
	}
	input.AppliedTo, err = nightsliputil.ResolveFieldInteractively(reader, facultyScope, "appliedTo", "warden/approver")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	input.RoomTypeID, err = nightsliputil.ResolveFieldInteractively(reader, scope, "roomTypeId", "venue type")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	academicScope, err := fetchNightSlipControlScope(scope, regNo, cookies, input, "roomTypeId")
	if err != nil {
		return NightSlipApplyInput{}, fmt.Errorf("unable to load venue options: %w", err)
	}
	input.BuildingID, err = resolveNightSlipOptionalFieldInteractively(reader, academicScope, "buildingId", "academic block")
	if err != nil {
		return NightSlipApplyInput{}, err
	}
	if input.BuildingID != "" {
		academicScope, err = fetchNightSlipControlScope(academicScope, regNo, cookies, input, "buildingId")
		if err != nil {
			return NightSlipApplyInput{}, fmt.Errorf("unable to load venue list: %w", err)
		}
	}
	input.Venue, err = resolveNightSlipOptionalFieldInteractively(reader, academicScope, "venue", "venue")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	input.LateHourEventID, err = nightsliputil.ResolveFieldInteractively(reader, scope, "lateHourEventId", "event")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	dateScope, err := fetchNightSlipControlScope(scope, regNo, cookies, input, "lateHourEventId")
	if err != nil {
		return NightSlipApplyInput{}, fmt.Errorf("unable to load time slots: %w", err)
	}

	input.Details, err = promptNightSlipValue(reader, "Enter reason/details (or type 'exit' to cancel): ", func(value string) (string, error) {
		sanitized := helpers.SanitizeString(strings.TrimSpace(value))
		if sanitized == "" {
			return "", fmt.Errorf("details cannot be empty")
		}
		return sanitized, nil
	})
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	input.FromDate, err = promptNightSlipValue(reader, "Enter from date (YYYY-MM-DD): ", func(value string) (string, error) {
		return normalizeNightSlipDate(value, "--from-date")
	})
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	input.FromTime, err = nightsliputil.ResolveFieldInteractively(reader, dateScope, "fromTime", "from time")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	input.ToDate, err = promptNightSlipValue(reader, "Enter to date (YYYY-MM-DD): ", func(value string) (string, error) {
		return normalizeNightSlipDate(value, "--to-date")
	})
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	mainScope, err := fetchNightSlipControlScope(dateScope, regNo, cookies, input, "fromTime")
	if err != nil {
		return NightSlipApplyInput{}, fmt.Errorf("unable to load end-time options: %w", err)
	}
	input.ToTime, err = nightsliputil.ResolveFieldInteractively(reader, mainScope, "toTime", "to time")
	if err != nil {
		return NightSlipApplyInput{}, err
	}

	if input, err = nightsliputil.NormalizeApplyInput(input); err != nil {
		return NightSlipApplyInput{}, err
	}

	helpers.Println()
	helpers.Println("Review nightslip request:")
	helpers.Printf("Details: %s\n", input.Details)
	helpers.Printf("From: %s %s\n", input.FromDate, input.FromTime)
	helpers.Printf("To: %s %s\n", input.ToDate, input.ToTime)

	confirm, err := promptNightSlipConfirmation(reader, "Submit this nightslip request? (yes/no): ")
	if err != nil {
		return NightSlipApplyInput{}, err
	}
	if !confirm {
		return NightSlipApplyInput{}, errNightSlipUserCanceled
	}

	return input, nil
}

func fetchNightSlipRequests(regNo string, cookies types.Cookies) ([]types.NightSlipRequest, error) {
	if err := openNightSlipMenu(regNo, cookies); err != nil {
		return nil, err
	}

	bodyText, err := fetchNightSlipStatusPage(regNo, cookies)
	if err != nil {
		return nil, err
	}

	if debug.Debug {
		helpers.Println("---- Response Body Start ----")
		helpers.Println(string(bodyText))
		helpers.Println("---- Response Body End ----")
	}

	return nightsliputil.ParseRequests(bodyText)
}

func openNightSlipMenu(regNo string, cookies types.Cookies) error {
	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d",
		regNo,
		cookies.CSRF,
		time.Now().UnixMilli(),
	)
	_, err := helpers.FetchReq(regNo, cookies, nightSlipMenuURL, "", payload, http.MethodPost, "")
	return err
}

func fetchNightSlipStatusPage(regNo string, cookies types.Cookies) ([]byte, error) {
	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&status=&form=undefined&control=status&x=%s",
		cookies.CSRF,
		regNo,
		nightSlipHTTPTime(),
	)
	return helpers.FetchReq(regNo, cookies, nightSlipStatusURL, "", payload, http.MethodPost, "")
}

func fetchNightSlipApplyPage(regNo string, cookies types.Cookies) ([]byte, error) {
	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&apply=&form=undefined&control=apply&x=%s",
		cookies.CSRF,
		regNo,
		nightSlipHTTPTime(),
	)
	return helpers.FetchReq(regNo, cookies, nightSlipApplyURL, "", payload, http.MethodPost, "")
}

func printNightSlipRequests(nightSlipRequests []types.NightSlipRequest) {
	helpers.Println()
	if len(nightSlipRequests) == 0 {
		helpers.Println("No nightslip requests found.")
		helpers.Println()
		return
	}

	allRequests := [][]string{
		{"VENUE", "EVENT TYPE", "DETAILS", "APPLIED TO", "FROM DATE", "TO DATE", "FROM/TO TIME", "STATUS"},
	}

	for _, slip := range nightSlipRequests {
		allRequests = append(allRequests, []string{
			slip.Venue,
			slip.EventType,
			slip.Details,
			slip.AppliedTo,
			slip.FromDate,
			slip.ToDate,
			slip.FromToTime,
			slip.Status,
		})
	}

	helpers.PrintTable(allRequests, 0)
	helpers.Println()
}

func parseNightSlipScope(body []byte) (*goquery.Selection, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	scope := doc.Find("form#LateHourRequestForm, form[name='LateHourRequestForm']").First()
	if scope.Length() == 0 {
		scope = doc.Selection
	}

	return scope, nil
}

func fetchNightSlipControlScope(currentScope *goquery.Selection, regNo string, cookies types.Cookies, input NightSlipApplyInput, controlName string) (*goquery.Selection, error) {
	controlURL := strings.TrimSpace(currentScope.Find(fmt.Sprintf("[name='%s']", controlName)).First().AttrOr("data-urlText", ""))
	if controlURL == "" {
		controlURL = nightSlipControlURLs[controlName]
	}
	if controlURL == "" {
		return nil, fmt.Errorf("missing control URL for %s", controlName)
	}

	body, err := fetchNightSlipControlBody(regNo, cookies, input, controlName, controlURL)
	if err != nil {
		return nil, err
	}

	return parseNightSlipScope(body)
}

func fetchNightSlipControlBody(regNo string, cookies types.Cookies, input NightSlipApplyInput, controlName string, controlURL string) ([]byte, error) {
	urlText := strings.TrimSpace(controlURL)
	if !strings.HasPrefix(urlText, "http") {
		urlText = helpers.VtopBaseURL + "/vtop/" + strings.TrimPrefix(urlText, "/")
	}

	payload := buildNightSlipFormValues(regNo, cookies, input)
	payload.Set("form", "LateHourRequestForm")
	payload.Set("control", controlName)
	payload.Set("x", nightSlipHTTPTime())

	return helpers.FetchReq(regNo, cookies, urlText, "", payload.Encode(), http.MethodPost, "")
}

func buildNightSlipFormValues(regNo string, cookies types.Cookies, input NightSlipApplyInput) url.Values {
	values := url.Values{}
	values.Set("_csrf", cookies.CSRF)
	values.Set("authorizedID", regNo)
	values.Set("costCentreId", input.CostCentreID)
	values.Set("appliedTo", input.AppliedTo)
	values.Set("roomTypeId", input.RoomTypeID)
	values.Set("buildingId", input.BuildingID)
	values.Set("venue", input.Venue)
	values.Set("lateHourEventId", input.LateHourEventID)
	values.Set("details", input.Details)
	values.Set("lateHourFromDate", input.FromDate)
	values.Set("fromTime", input.FromTime)
	values.Set("lateHourToDate", input.ToDate)
	values.Set("toTime", input.ToTime)
	return values
}

func resolveNightSlipOptionalFieldInteractively(reader *bufio.Reader, scope *goquery.Selection, fieldName string, label string) (string, error) {
	if value := strings.TrimSpace(scope.Find(fmt.Sprintf("input[name='%s']", fieldName)).First().AttrOr("value", "")); value != "" {
		return value, nil
	}

	fieldSelection := scope.Find(fmt.Sprintf("select[name='%s']", fieldName)).First()
	if fieldSelection.Length() == 0 {
		return "", nil
	}
	if _, disabled := fieldSelection.Attr("disabled"); disabled {
		return "", nil
	}

	options, selectedValue := nightsliputil.CollectOptions(fieldSelection)
	if selectedValue != "" {
		return selectedValue, nil
	}
	if len(options) == 0 {
		return "", nil
	}
	if len(options) == 1 {
		return options[0].Value, nil
	}

	return nightsliputil.PromptOptionSelection(reader, options, label)
}

func promptNightSlipValue(reader *bufio.Reader, prompt string, validate func(string) (string, error)) (string, error) {
	for {
		helpers.Print(prompt)
		rawInput, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		trimmed := strings.TrimSpace(rawInput)
		if isNightSlipCancelInput(trimmed) {
			return "", errNightSlipUserCanceled
		}

		value, err := validate(trimmed)
		if err != nil {
			helpers.Println(err.Error())
			continue
		}

		return value, nil
	}
}

func promptNightSlipConfirmation(reader *bufio.Reader, prompt string) (bool, error) {
	for {
		helpers.Print(prompt)
		rawInput, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(rawInput)) {
		case "yes", "y":
			return true, nil
		case "no", "n":
			return false, nil
		case "exit", "cancel":
			return false, errNightSlipUserCanceled
		default:
			helpers.Println("Invalid input. Please respond with 'yes' or 'no'.")
		}
	}
}

func isNightSlipCancelInput(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "exit", "cancel":
		return true
	default:
		return false
	}
}

func resolveNightSlipApplyInput(body []byte, defaultCSRF string, input NightSlipApplyInput) (resolvedNightSlipApplyInput, error) {
	normalizedInput, err := nightsliputil.NormalizeApplyInput(input)
	if err != nil {
		return resolvedNightSlipApplyInput{}, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return resolvedNightSlipApplyInput{}, err
	}

	scope := doc.Find("form#LateHourRequestForm, form[name='LateHourRequestForm']").First()
	if scope.Length() == 0 {
		scope = doc.Selection
	}

	resolved := resolvedNightSlipApplyInput{
		NightSlipApplyInput: normalizedInput,
		CSRF:                strings.TrimSpace(scope.Find("input[name='_csrf']").First().AttrOr("value", "")),
	}
	if resolved.CSRF == "" {
		resolved.CSRF = defaultCSRF
	}

	resolved.CostCentreID, err = nightsliputil.ResolveField(scope, "costCentreId", normalizedInput.CostCentreID, "--cost-centre-id")
	if err != nil {
		return resolvedNightSlipApplyInput{}, err
	}

	resolved.AppliedTo, err = nightsliputil.ResolveField(scope, "appliedTo", normalizedInput.AppliedTo, "--applied-to")
	if err != nil {
		return resolvedNightSlipApplyInput{}, err
	}

	resolved.RoomTypeID, err = nightsliputil.ResolveField(scope, "roomTypeId", normalizedInput.RoomTypeID, "--room-type-id")
	if err != nil {
		return resolvedNightSlipApplyInput{}, err
	}

	resolved.LateHourEventID, err = nightsliputil.ResolveField(scope, "lateHourEventId", normalizedInput.LateHourEventID, "--event-id")
	if err != nil {
		return resolvedNightSlipApplyInput{}, err
	}

	return resolved, nil
}

func buildNightSlipSubmitPayload(regNo string, body []byte, input resolvedNightSlipApplyInput) (url.Values, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	scope := doc.Find("form#LateHourRequestForm, form[name='LateHourRequestForm']").First()
	if scope.Length() == 0 {
		scope = doc.Selection
	}

	copyNightSlipFormDefaults(values, scope)

	values.Set("_csrf", input.CSRF)
	values.Set("authorizedID", regNo)
	values.Set("costCentreId", input.CostCentreID)
	values.Set("appliedTo", input.AppliedTo)
	values.Set("roomTypeId", input.RoomTypeID)
	values.Set("buildingId", input.BuildingID)
	values.Set("venue", input.Venue)
	values.Set("lateHourEventId", input.LateHourEventID)
	values.Set("details", input.Details)
	values.Set("lateHourFromDate", input.FromDate)
	values.Set("fromTime", input.FromTime)
	values.Set("lateHourToDate", input.ToDate)
	values.Set("toTime", input.ToTime)
	values.Set("form", "LateHourRequestForm")
	values.Set("control", "submitButton1")
	values.Set("x", nightSlipHTTPTime())

	return values, nil
}

func copyNightSlipFormDefaults(values url.Values, scope *goquery.Selection) {
	scope.Find("input[name]").Each(func(_ int, selection *goquery.Selection) {
		name := strings.TrimSpace(selection.AttrOr("name", ""))
		if name == "" {
			return
		}

		inputType := strings.ToLower(strings.TrimSpace(selection.AttrOr("type", "text")))
		if inputType == "checkbox" || inputType == "radio" {
			if _, checked := selection.Attr("checked"); !checked {
				return
			}
		}

		value := strings.TrimSpace(selection.AttrOr("value", ""))
		if value != "" {
			values.Set(name, value)
		}
	})

	scope.Find("select[name]").Each(func(_ int, selection *goquery.Selection) {
		name := strings.TrimSpace(selection.AttrOr("name", ""))
		if name == "" {
			return
		}
		if value, ok := selectedNightSlipOptionValue(selection); ok {
			values.Set(name, value)
		}
	})
}

func selectedNightSlipOptionValue(selection *goquery.Selection) (string, bool) {
	if selected := selection.Find("option[selected]").First(); selected.Length() > 0 {
		value := strings.TrimSpace(selected.AttrOr("value", ""))
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func normalizeNightSlipDate(value string, flagName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing %s", flagName)
	}

	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		"02/01/06",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("invalid %s; use YYYY-MM-DD", flagName)
}

func normalizeNightSlipTime(value string, flagName string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("missing %s", flagName)
	}

	layouts := []string{
		"03:04 PM",
		"3:04 PM",
		"03:04PM",
		"3:04PM",
		"15:04",
		"15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Format("03:04 PM"), nil
		}
	}

	return "", fmt.Errorf("invalid %s; use a time like 08:30 PM or 20:30", flagName)
}

func wasNightSlipSubmitted(before []types.NightSlipRequest, after []types.NightSlipRequest, input NightSlipApplyInput) bool {
	beforeCounts := make(map[string]int, len(before))
	for _, request := range before {
		beforeCounts[nightSlipRequestKey(request)]++
	}

	expectedFromDate := formatNightSlipStatusDate(input.FromDate)
	expectedToDate := formatNightSlipStatusDate(input.ToDate)
	expectedTimeWindow := formatNightSlipStatusWindow(input.FromTime, input.ToTime)
	expectedDetails := strings.TrimSpace(input.Details)

	for _, request := range after {
		key := nightSlipRequestKey(request)
		if beforeCounts[key] > 0 {
			beforeCounts[key]--
			continue
		}

		if strings.TrimSpace(request.Details) != expectedDetails {
			continue
		}
		if request.FromDate != expectedFromDate || request.ToDate != expectedToDate {
			continue
		}
		if strings.TrimSpace(request.FromToTime) != expectedTimeWindow {
			continue
		}
		if !strings.Contains(strings.ToUpper(request.RawStatus), "PENDING") {
			continue
		}

		return true
	}

	return false
}

func nightSlipRequestKey(request types.NightSlipRequest) string {
	return strings.Join([]string{
		strings.TrimSpace(request.Venue),
		strings.TrimSpace(request.EventType),
		strings.TrimSpace(request.Details),
		strings.TrimSpace(request.AppliedTo),
		strings.TrimSpace(request.FromDate),
		strings.TrimSpace(request.ToDate),
		strings.TrimSpace(request.FromToTime),
		strings.TrimSpace(request.RawStatus),
	}, "|")
}

func formatNightSlipStatusDate(value string) string {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02/01/06")
}

func formatNightSlipStatusWindow(fromTime string, toTime string) string {
	fromParsed, fromErr := time.Parse("03:04 PM", fromTime)
	toParsed, toErr := time.Parse("03:04 PM", toTime)
	if fromErr != nil || toErr != nil {
		return strings.TrimSpace(fromTime) + " to " + strings.TrimSpace(toTime)
	}
	return fmt.Sprintf("%s to %s", fromParsed.Format("15:04"), toParsed.Format("15:04"))
}

func nightSlipHTTPTime() string {
	return time.Now().UTC().Format(http.TimeFormat)
}
