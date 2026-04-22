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
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

const (
	LeaveStatusTableSelector = "table#LeaveAppliedTable"
	LeaveStatusRowsSelector  = "tbody tr"
	LeaveStatusCellSelector  = "td"
)

const (
	leaveMenuURL   = "https://vtop.vit.ac.in/vtop/hostels/student/leave/1"
	leaveApplyURL  = "https://vtop.vit.ac.in/vtop/hostels/student/leave/2"
	leaveSubmitURL = "https://vtop.vit.ac.in/vtop/hostels/student/leave/3"
	leaveStatusURL = "https://vtop.vit.ac.in/vtop/hostels/student/leave/4"
)

const (
	leaveVisitingPlaceMaxLength = 150
	leaveReasonMaxLength        = 500
)

type LeaveApplyInput struct {
	Apply         bool
	LeaveCode     string
	VisitingPlace string
	Reason        string
	FromDate      string
	FromTime      string
	ToDate        string
	ToTime        string
}

type resolvedLeaveApplyInput struct {
	LeaveApplyInput
	CSRF string
}

var errLeaveUserCanceled = errors.New("leave application canceled")

func ExecuteLeave(regNo string, cookies types.Cookies, input LeaveApplyInput) error {
	if !helpers.ValidateLogin(cookies) {
		return nil
	}

	if !input.Apply {
		GetLeaveStatus(regNo, cookies)
		return nil
	}

	return ApplyLeave(regNo, cookies, input)
}

func RunInteractiveLeave(regNo string, cookies types.Cookies) error {
	if !helpers.ValidateLogin(cookies) {
		return nil
	}

	existingRequests, err := fetchLeaveRequests(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error fetching leave status:", err)
		}
		helpers.Println("Unable to fetch leave status.")
		return err
	}

	printLeaveRequests(existingRequests)

	if hasPendingLeaveRequest(existingRequests) {
		helpers.Println("A leave request is already pending approval. New applications are blocked until it clears.")
		return nil
	}

	if !helpers.IsInteractiveOutput(os.Stdin) {
		helpers.Println("No pending leave requests. Run this command in an interactive terminal to submit a new one.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	proceed, err := promptLeaveConfirmation(reader, "No pending leave requests. Apply for a new one? (yes/no): ")
	if err != nil {
		if errors.Is(err, errLeaveUserCanceled) {
			helpers.Println("Leave application canceled.")
			return nil
		}
		return err
	}
	if !proceed {
		helpers.Println("Leave application canceled.")
		return nil
	}

	applyBody, err := fetchLeaveApplyPage(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error opening leave apply form:", err)
		}
		helpers.Println("Unable to open the leave application form.")
		return err
	}

	if message := extractLeaveMessage(applyBody); message != "" && strings.Contains(strings.ToUpper(message), "PENDING") {
		helpers.Println(message)
		return errors.New(message)
	}

	input, err := promptLeaveApplyInput(reader, applyBody)
	if err != nil {
		if errors.Is(err, errLeaveUserCanceled) {
			helpers.Println("Leave application canceled.")
			return nil
		}
		helpers.Println(err.Error())
		return err
	}

	return ApplyLeave(regNo, cookies, input)
}

func GetLeaveStatus(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	leaveRequests, err := fetchLeaveRequests(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error fetching leave status:", err)
		}
		return
	}

	printLeaveRequests(leaveRequests)
}

func ApplyLeave(regNo string, cookies types.Cookies, input LeaveApplyInput) error {
	existingRequests, err := fetchLeaveRequests(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error preparing leave request:", err)
		}
		return err
	}

	if hasPendingLeaveRequest(existingRequests) {
		helpers.Println()
		helpers.Println("A leave request is already pending approval. Clear it before applying for another one.")
		printLeaveRequests(existingRequests)
		return fmt.Errorf("leave request already pending")
	}

	applyBody, err := fetchLeaveApplyPage(regNo, cookies)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error opening leave apply form:", err)
		}
		return err
	}

	if message := extractLeaveMessage(applyBody); message != "" && strings.Contains(strings.ToUpper(message), "PENDING") {
		helpers.Println()
		helpers.Println(message)
		return errors.New(message)
	}

	resolvedInput, err := resolveLeaveApplyInput(applyBody, cookies.CSRF, input)
	if err != nil {
		helpers.Println()
		helpers.Println(err.Error())
		return err
	}

	submitPayload, err := buildLeaveSubmitPayload(regNo, applyBody, resolvedInput)
	if err != nil {
		helpers.Println()
		helpers.Println(err.Error())
		return err
	}

	submitBody, err := helpers.FetchReq(regNo, cookies, leaveSubmitURL, "", submitPayload.Encode(), http.MethodPost, "")
	if err != nil {
		if debug.Debug {
			helpers.Println("Error submitting leave request:", err)
		}
		return err
	}

	submitMessage := extractLeaveMessage(submitBody)
	updatedRequests, updatedErr := fetchLeaveRequests(regNo, cookies)
	if updatedErr == nil && wasLeaveSubmitted(existingRequests, updatedRequests, resolvedInput.LeaveApplyInput) {
		helpers.Println()
		if submitMessage != "" {
			helpers.Println(submitMessage)
		} else {
			helpers.Println("Leave request submitted successfully.")
		}
		printLeaveRequests(updatedRequests)
		return nil
	}

	if submitMessage != "" {
		helpers.Println()
		helpers.Println(submitMessage)
	}

	if updatedErr != nil {
		return updatedErr
	}

	return fmt.Errorf("unable to verify leave submission")
}

func fetchLeaveRequests(regNo string, cookies types.Cookies) ([]types.LeaveRequest, error) {
	if err := openLeaveMenu(regNo, cookies); err != nil {
		return nil, err
	}

	bodyText, err := fetchLeaveStatusPage(regNo, cookies)
	if err != nil {
		return nil, err
	}

	return ParseLeaveRequests(bodyText)
}

func openLeaveMenu(regNo string, cookies types.Cookies) error {
	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d",
		regNo,
		cookies.CSRF,
		time.Now().UnixMilli(),
	)
	_, err := helpers.FetchReq(regNo, cookies, leaveMenuURL, "", payload, http.MethodPost, "")
	return err
}

func fetchLeaveStatusPage(regNo string, cookies types.Cookies) ([]byte, error) {
	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&status=&form=undefined&control=status&x=%s",
		cookies.CSRF,
		regNo,
		leaveHTTPTime(),
	)
	return helpers.FetchReq(regNo, cookies, leaveStatusURL, "", payload, http.MethodPost, "")
}

func fetchLeaveApplyPage(regNo string, cookies types.Cookies) ([]byte, error) {
	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&apply=&form=undefined&control=apply&x=%s",
		cookies.CSRF,
		regNo,
		leaveHTTPTime(),
	)
	return helpers.FetchReq(regNo, cookies, leaveApplyURL, "", payload, http.MethodPost, "")
}

func ParseLeaveRequests(bodyText []byte) ([]types.LeaveRequest, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		return nil, err
	}

	var leaveRequests []types.LeaveRequest
	doc.Find(LeaveStatusTableSelector).Find(LeaveStatusRowsSelector).Each(func(i int, rowSelection *goquery.Selection) {
		visitPlace, reason, leaveType, from, to, status := parseLeaveStatusRow(rowSelection)
		if visitPlace != "" {
			leaveRequests = append(leaveRequests, types.LeaveRequest{
				VisitPlace: visitPlace,
				Reason:     reason,
				LeaveType:  leaveType,
				From:       helpers.FormatDate(from),
				To:         helpers.FormatDate(to),
				RawStatus:  status,
				Status:     helpers.ColorStatus(status),
			})
		}
	})

	return leaveRequests, nil
}

func NormalizeLeaveApplyInput(input LeaveApplyInput) (LeaveApplyInput, error) {
	return normalizeLeaveApplyInput(input)
}

func ResolveLeaveApplyInput(body []byte, defaultCSRF string, input LeaveApplyInput) (LeaveApplyInput, error) {
	resolved, err := resolveLeaveApplyInput(body, defaultCSRF, input)
	if err != nil {
		return LeaveApplyInput{}, err
	}
	return resolved.LeaveApplyInput, nil
}

func BuildLeaveSubmitPayload(regNo string, body []byte, defaultCSRF string, input LeaveApplyInput) (url.Values, error) {
	resolved, err := resolveLeaveApplyInput(body, defaultCSRF, input)
	if err != nil {
		return nil, err
	}
	return buildLeaveSubmitPayload(regNo, body, resolved)
}

func parseLeaveStatusRow(rowSelection *goquery.Selection) (string, string, string, string, string, string) {
	cells := rowSelection.Find(LeaveStatusCellSelector)
	if cells.Length() >= 8 {
		return strings.TrimSpace(cells.Eq(2).Text()),
			strings.TrimSpace(cells.Eq(3).Text()),
			strings.TrimSpace(cells.Eq(4).Text()),
			strings.TrimSpace(cells.Eq(5).Text()),
			strings.TrimSpace(cells.Eq(6).Text()),
			strings.TrimSpace(cells.Eq(7).Text())
	}

	primaryCells := rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap")
	return strings.TrimSpace(primaryCells.Eq(0).Text()),
		strings.TrimSpace(primaryCells.Eq(1).Text()),
		strings.TrimSpace(primaryCells.Eq(2).Text()),
		strings.TrimSpace(primaryCells.Eq(3).Text()),
		strings.TrimSpace(primaryCells.Eq(4).Text()),
		strings.TrimSpace(primaryCells.Eq(5).Text())
}

func printLeaveRequests(leaveRequests []types.LeaveRequest) {
	helpers.Println()
	if len(leaveRequests) == 0 {
		helpers.Println("No leave requests found.")
		return
	}
	var allRequests [][]string
	allRequests = append(allRequests, []string{"VISIT PLACE", "REASON", "LEAVE TYPE", "FROM", "TO", "STATUS"})

	for _, leave := range leaveRequests {
		allRequests = append(allRequests, []string{
			leave.VisitPlace,
			leave.Reason,
			leave.LeaveType,
			leave.From,
			leave.To,
			leave.Status,
		})
	}

	helpers.PrintTable(allRequests, 0)
	helpers.Println()
}

func promptLeaveApplyInput(reader *bufio.Reader, body []byte) (LeaveApplyInput, error) {
	scope, err := parseLeaveScope(body)
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input := LeaveApplyInput{Apply: true}

	input.LeaveCode, err = resolveLeaveFieldInteractively(reader, scope, "leaveCode", "leave type")
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.VisitingPlace, err = promptLeaveValue(reader, "Enter visiting place (or type 'exit' to cancel): ", func(value string) (string, error) {
		return normalizeLeaveTextField(value, "visiting place", "--visiting-place", leaveVisitingPlaceMaxLength)
	})
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.Reason, err = promptLeaveValue(reader, "Enter reason (or type 'exit' to cancel): ", func(value string) (string, error) {
		return normalizeLeaveTextField(value, "reason", "--reason", leaveReasonMaxLength)
	})
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.FromDate, err = promptLeaveValue(reader, "Enter from date (YYYY-MM-DD): ", func(value string) (string, error) {
		return normalizeLeaveDate(value, "--from-date")
	})
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.FromTime, err = promptLeaveValue(reader, "Enter from time (HH:MM or h:mm AM/PM): ", func(value string) (string, error) {
		return normalizeLeaveTime(value, "--from-time")
	})
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.ToDate, err = promptLeaveValue(reader, "Enter to date (YYYY-MM-DD): ", func(value string) (string, error) {
		return normalizeLeaveDate(value, "--to-date")
	})
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.ToTime, err = promptLeaveValue(reader, "Enter to time (HH:MM or h:mm AM/PM): ", func(value string) (string, error) {
		return normalizeLeaveTime(value, "--to-time")
	})
	if err != nil {
		return LeaveApplyInput{}, err
	}

	if input, err = normalizeLeaveApplyInput(input); err != nil {
		return LeaveApplyInput{}, err
	}

	helpers.Println()
	helpers.Println("Review leave request:")
	helpers.Printf("Leave type: %s\n", input.LeaveCode)
	helpers.Printf("Visiting place: %s\n", input.VisitingPlace)
	helpers.Printf("Reason: %s\n", input.Reason)
	helpers.Printf("From: %s %s\n", input.FromDate, input.FromTime)
	helpers.Printf("To: %s %s\n", input.ToDate, input.ToTime)

	confirm, err := promptLeaveConfirmation(reader, "Submit this leave request? (yes/no): ")
	if err != nil {
		return LeaveApplyInput{}, err
	}
	if !confirm {
		return LeaveApplyInput{}, errLeaveUserCanceled
	}

	return input, nil
}

func resolveLeaveApplyInput(body []byte, defaultCSRF string, input LeaveApplyInput) (resolvedLeaveApplyInput, error) {
	normalizedInput, err := normalizeLeaveApplyInput(input)
	if err != nil {
		return resolvedLeaveApplyInput{}, err
	}

	scope, err := parseLeaveScope(body)
	if err != nil {
		return resolvedLeaveApplyInput{}, err
	}

	resolved := resolvedLeaveApplyInput{
		LeaveApplyInput: normalizedInput,
		CSRF:            strings.TrimSpace(scope.Find("input[name='_csrf']").First().AttrOr("value", "")),
	}
	if resolved.CSRF == "" {
		resolved.CSRF = defaultCSRF
	}

	resolved.LeaveCode, err = resolveLeaveField(scope, "leaveCode", normalizedInput.LeaveCode, "--leave-code")
	if err != nil {
		return resolvedLeaveApplyInput{}, err
	}

	return resolved, nil
}

func buildLeaveSubmitPayload(regNo string, body []byte, input resolvedLeaveApplyInput) (url.Values, error) {
	scope, err := parseLeaveScope(body)
	if err != nil {
		return nil, err
	}

	normalizedInput, err := normalizeLeaveApplyInput(input.LeaveApplyInput)
	if err != nil {
		return nil, err
	}
	input.LeaveApplyInput = normalizedInput

	values := url.Values{}
	copyLeaveFormDefaults(values, scope)

	values.Set("_csrf", input.CSRF)
	values.Set("authorizedID", regNo)
	values.Set("leaveCode", input.LeaveCode)
	values.Set("visitingPlace", input.VisitingPlace)
	values.Set("reason", input.Reason)
	values.Set("leaveFromDate", input.FromDate)
	values.Set("fromTime", input.FromTime)
	values.Set("leaveToDate", input.ToDate)
	values.Set("toTime", input.ToTime)
	values.Set("form", "LeaveRequestForm")
	values.Set("control", "submitControl2")
	values.Set("x", leaveHTTPTime())

	return values, nil
}

func parseLeaveScope(body []byte) (*goquery.Selection, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	scope := doc.Find("form#LeaveRequestForm, form[name='LeaveRequestForm']").First()
	if scope.Length() == 0 {
		scope = doc.Selection
	}

	return scope, nil
}

func resolveLeaveFieldInteractively(reader *bufio.Reader, scope *goquery.Selection, fieldName string, label string) (string, error) {
	if value := strings.TrimSpace(scope.Find(fmt.Sprintf("input[name='%s']", fieldName)).First().AttrOr("value", "")); value != "" {
		return value, nil
	}

	fieldSelection := scope.Find(fmt.Sprintf("select[name='%s']", fieldName)).First()
	if fieldSelection.Length() == 0 {
		return "", fmt.Errorf("VTOP did not expose %s in the apply form", fieldName)
	}

	options, selectedValue := nightsliputil.CollectOptions(fieldSelection)
	if selectedValue != "" {
		return selectedValue, nil
	}
	if len(options) == 1 {
		return options[0].Value, nil
	}
	if len(options) == 0 {
		return "", fmt.Errorf("VTOP returned no usable options for %s", fieldName)
	}

	value, err := nightsliputil.PromptOptionSelection(reader, options, label)
	if errors.Is(err, nightsliputil.ErrUserCanceled) {
		return "", errLeaveUserCanceled
	}
	return value, err
}

func resolveLeaveField(scope *goquery.Selection, fieldName string, providedValue string, flagName string) (string, error) {
	providedValue = strings.TrimSpace(providedValue)

	if value := strings.TrimSpace(scope.Find(fmt.Sprintf("input[name='%s']", fieldName)).First().AttrOr("value", "")); value != "" {
		if providedValue != "" && providedValue != value {
			return "", fmt.Errorf("invalid %s %q; VTOP fixed %s to %q", flagName, providedValue, fieldName, value)
		}
		return value, nil
	}

	fieldSelection := scope.Find(fmt.Sprintf("select[name='%s']", fieldName)).First()
	if fieldSelection.Length() == 0 {
		if providedValue != "" {
			return providedValue, nil
		}
		return "", fmt.Errorf("missing %s and VTOP did not expose %s in the apply form", flagName, fieldName)
	}

	options, selectedValue := nightsliputil.CollectOptions(fieldSelection)
	if providedValue != "" {
		if matchedValue, ok := matchLeaveOptionValue(options, providedValue); ok {
			return matchedValue, nil
		}
		return "", fmt.Errorf("invalid %s %q; available %s values: %s", flagName, providedValue, fieldName, formatLeaveOptions(options))
	}
	if selectedValue != "" {
		return selectedValue, nil
	}
	if len(options) == 1 {
		return options[0].Value, nil
	}
	if len(options) == 0 {
		return "", fmt.Errorf("missing %s and VTOP returned no usable options for %s", flagName, fieldName)
	}

	return "", fmt.Errorf("missing %s; choose one of the available %s values", flagName, fieldName)
}

func copyLeaveFormDefaults(values url.Values, scope *goquery.Selection) {
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
		if value, ok := selectedLeaveOptionValue(selection); ok {
			values.Set(name, value)
		}
	})

	scope.Find("textarea[name]").Each(func(_ int, selection *goquery.Selection) {
		name := strings.TrimSpace(selection.AttrOr("name", ""))
		if name == "" {
			return
		}
		value := strings.TrimSpace(selection.Text())
		if value != "" {
			values.Set(name, value)
		}
	})
}

func selectedLeaveOptionValue(selection *goquery.Selection) (string, bool) {
	if selected := selection.Find("option[selected]").First(); selected.Length() > 0 {
		value := strings.TrimSpace(selected.AttrOr("value", ""))
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func promptLeaveValue(reader *bufio.Reader, prompt string, validate func(string) (string, error)) (string, error) {
	for {
		helpers.Print(prompt)
		rawInput, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		trimmed := strings.TrimSpace(rawInput)
		if isLeaveCancelInput(trimmed) {
			return "", errLeaveUserCanceled
		}

		value, err := validate(trimmed)
		if err != nil {
			helpers.Println(err.Error())
			continue
		}

		return value, nil
	}
}

func promptLeaveConfirmation(reader *bufio.Reader, prompt string) (bool, error) {
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
			return false, errLeaveUserCanceled
		default:
			helpers.Println("Invalid input. Please respond with 'yes' or 'no'.")
		}
	}
}

func normalizeLeaveApplyInput(input LeaveApplyInput) (LeaveApplyInput, error) {
	var err error

	input.LeaveCode = strings.TrimSpace(input.LeaveCode)
	input.VisitingPlace, err = normalizeLeaveTextField(input.VisitingPlace, "visiting place", "--visiting-place", leaveVisitingPlaceMaxLength)
	if err != nil {
		return LeaveApplyInput{}, err
	}
	input.Reason, err = normalizeLeaveTextField(input.Reason, "reason", "--reason", leaveReasonMaxLength)
	if err != nil {
		return LeaveApplyInput{}, err
	}

	input.FromDate, err = normalizeLeaveDate(input.FromDate, "--from-date")
	if err != nil {
		return LeaveApplyInput{}, err
	}
	input.FromTime, err = normalizeLeaveTime(input.FromTime, "--from-time")
	if err != nil {
		return LeaveApplyInput{}, err
	}
	input.ToDate, err = normalizeLeaveDate(input.ToDate, "--to-date")
	if err != nil {
		return LeaveApplyInput{}, err
	}
	input.ToTime, err = normalizeLeaveTime(input.ToTime, "--to-time")
	if err != nil {
		return LeaveApplyInput{}, err
	}

	fromDate, _ := time.Parse("2006-01-02", input.FromDate)
	toDate, _ := time.Parse("2006-01-02", input.ToDate)
	if toDate.Before(fromDate) {
		return LeaveApplyInput{}, fmt.Errorf("--to-date must be on or after --from-date")
	}
	if input.FromDate == input.ToDate {
		fromTime, fromErr := time.Parse("15:04", input.FromTime)
		toTime, toErr := time.Parse("15:04", input.ToTime)
		if fromErr == nil && toErr == nil && !toTime.After(fromTime) {
			return LeaveApplyInput{}, fmt.Errorf("--to-time must be after --from-time when both dates are the same")
		}
	}

	return input, nil
}

func normalizeLeaveTextField(value string, label string, flagName string, maxLength int) (string, error) {
	sanitized := helpers.SanitizeString(value)
	if sanitized == "" {
		return "", fmt.Errorf("missing %s", flagName)
	}
	if utf8.RuneCountInString(sanitized) > maxLength {
		return "", fmt.Errorf("%s must be %d characters or fewer", label, maxLength)
	}
	if strings.ContainsAny(sanitized, "<>") {
		return "", fmt.Errorf("%s cannot contain '<' or '>'", label)
	}

	hasLetterOrDigit := false
	for _, r := range sanitized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasLetterOrDigit = true
			break
		}
	}
	if !hasLetterOrDigit {
		return "", fmt.Errorf("%s must contain at least one letter or number", label)
	}

	return sanitized, nil
}

func normalizeLeaveDate(value string, flagName string) (string, error) {
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

func normalizeLeaveTime(value string, flagName string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("missing %s", flagName)
	}

	layouts := []string{
		"15:04",
		"15:04:05",
		"03:04 PM",
		"3:04 PM",
		"03:04PM",
		"3:04PM",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Format("15:04"), nil
		}
	}

	return "", fmt.Errorf("invalid %s; use a time like 20:30 or 08:30 PM", flagName)
}

func hasPendingLeaveRequest(requests []types.LeaveRequest) bool {
	for _, request := range requests {
		status := strings.ToUpper(strings.TrimSpace(request.RawStatus))
		if status == "" {
			status = strings.ToUpper(strings.TrimSpace(helpers.StripAnsiCodes(request.Status)))
		}
		if strings.Contains(status, "PENDING") {
			return true
		}
	}
	return false
}

func wasLeaveSubmitted(before []types.LeaveRequest, after []types.LeaveRequest, input LeaveApplyInput) bool {
	beforeCounts := make(map[string]int, len(before))
	for _, request := range before {
		beforeCounts[leaveRequestKey(request)]++
	}

	expectedFrom := formatLeaveStatusDate(input.FromDate)
	expectedTo := formatLeaveStatusDate(input.ToDate)

	for _, request := range after {
		key := leaveRequestKey(request)
		if beforeCounts[key] > 0 {
			beforeCounts[key]--
			continue
		}

		if !strings.EqualFold(strings.TrimSpace(request.VisitPlace), strings.TrimSpace(input.VisitingPlace)) {
			continue
		}
		if strings.TrimSpace(request.Reason) != strings.TrimSpace(input.Reason) {
			continue
		}
		if request.From != expectedFrom || request.To != expectedTo {
			continue
		}
		if !strings.Contains(strings.ToUpper(request.RawStatus), "PENDING") {
			continue
		}

		return true
	}

	return false
}

func leaveRequestKey(request types.LeaveRequest) string {
	return strings.Join([]string{
		strings.TrimSpace(request.VisitPlace),
		strings.TrimSpace(request.Reason),
		strings.TrimSpace(request.LeaveType),
		strings.TrimSpace(request.From),
		strings.TrimSpace(request.To),
		strings.TrimSpace(request.RawStatus),
	}, "|")
}

func formatLeaveStatusDate(value string) string {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02/01/06")
}

func matchLeaveOptionValue(options []nightsliputil.FormOption, providedValue string) (string, bool) {
	providedValue = helpers.SanitizeString(providedValue)
	for _, option := range options {
		if strings.EqualFold(option.Value, providedValue) || strings.EqualFold(option.Label, providedValue) {
			return option.Value, true
		}
	}
	return "", false
}

func formatLeaveOptions(options []nightsliputil.FormOption) string {
	formatted := make([]string, 0, len(options))
	for _, option := range options {
		if option.Label == "" || option.Label == option.Value {
			formatted = append(formatted, option.Value)
			continue
		}
		formatted = append(formatted, fmt.Sprintf("%s (%s)", option.Value, option.Label))
	}
	return strings.Join(formatted, ", ")
}

func extractLeaveMessage(body []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return ""
	}

	selectors := []string{
		"#DataBlockMessage",
		"#MenuBlockMessage",
		"#messageText",
		"#pageMessage",
		"div.alert",
		"div.alert-success",
		"div.alert-danger",
		"div.message",
		"span.text-danger",
		"span.text-success",
	}

	for _, selector := range selectors {
		text := helpers.SanitizeString(strings.TrimSpace(doc.Find(selector).First().Text()))
		if text != "" {
			return text
		}
	}

	return ""
}

func isLeaveCancelInput(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "exit", "cancel":
		return true
	default:
		return false
	}
}

func leaveHTTPTime() string {
	return time.Now().UTC().Format(http.TimeFormat)
}
