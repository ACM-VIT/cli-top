package nightsliputil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cli-top/helpers"
	"cli-top/types"

	"github.com/PuerkitoBio/goquery"
)

const (
	tableSelector           = "table#LateHourStatusTable"
	rowsSelector            = "tbody tr"
	cellSelector            = "td"
	selectionPageSize       = 12
	searchRequiredThreshold = 50
)

type ApplyInput struct {
	Apply           bool
	CostCentreID    string
	AppliedTo       string
	RoomTypeID      string
	BuildingID      string
	Venue           string
	LateHourEventID string
	Details         string
	FromDate        string
	FromTime        string
	ToDate          string
	ToTime          string
}

type FormOption struct {
	Value string
	Label string
}

var ErrUserCanceled = errors.New("nightslip application canceled")

func ParseRequests(body []byte) ([]types.NightSlipRequest, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var requests []types.NightSlipRequest
	doc.Find(tableSelector).Find(rowsSelector).Each(func(_ int, rowSelection *goquery.Selection) {
		venue := strings.TrimSpace(rowSelection.Find(cellSelector).Eq(2).Text())
		eventType := strings.TrimSpace(rowSelection.Find(cellSelector).Eq(3).Text())
		details := strings.TrimSpace(rowSelection.Find(cellSelector).Eq(4).Text())
		appliedTo := strings.TrimSpace(rowSelection.Find(cellSelector).Eq(5).Text())
		fromDate := helpers.FormatDate(strings.TrimSpace(rowSelection.Find(cellSelector).Eq(6).Text()))
		toDate := helpers.FormatDate(strings.TrimSpace(rowSelection.Find(cellSelector).Eq(7).Text()))
		fromToTime := strings.TrimSpace(rowSelection.Find(cellSelector).Eq(8).Text())
		rawStatus := normalizeStatus(strings.TrimSpace(rowSelection.Find(cellSelector).Eq(9).Text()))

		if venue == "" {
			return
		}

		requests = append(requests, types.NightSlipRequest{
			Venue:      venue,
			EventType:  eventType,
			Details:    details,
			AppliedTo:  appliedTo,
			FromDate:   fromDate,
			ToDate:     toDate,
			FromToTime: fromToTime,
			RawStatus:  rawStatus,
			Status:     helpers.ColorStatus(rawStatus),
		})
	})

	return requests, nil
}

func ResolveFieldInteractively(reader *bufio.Reader, scope *goquery.Selection, fieldName string, label string) (string, error) {
	if value := strings.TrimSpace(scope.Find(fmt.Sprintf("input[name='%s']", fieldName)).First().AttrOr("value", "")); value != "" {
		return value, nil
	}

	fieldSelection := scope.Find(fmt.Sprintf("select[name='%s']", fieldName)).First()
	if fieldSelection.Length() == 0 {
		return "", fmt.Errorf("VTOP did not expose %s in the apply form", fieldName)
	}

	options, selectedValue := CollectOptions(fieldSelection)
	if selectedValue != "" {
		return selectedValue, nil
	}
	if len(options) == 1 {
		return options[0].Value, nil
	}
	if len(options) == 0 {
		return "", fmt.Errorf("VTOP returned no usable options for %s", fieldName)
	}

	return PromptOptionSelection(reader, options, label)
}

func PromptOptionSelection(reader *bufio.Reader, options []FormOption, label string) (string, error) {
	searchQuery := ""
	searchRequired := len(options) > searchRequiredThreshold
	visibleCount := len(options)
	if searchRequired {
		visibleCount = selectionPageSize
	}

	for {
		filteredOptions := FilterOptions(options, searchQuery)

		switch {
		case searchRequired && searchQuery == "":
			helpers.Println()
			helpers.Printf("Select %s.\n", label)
			helpers.Printf("%d options available. Start typing to search by name, acronym, or id.\n", len(options))
		case len(filteredOptions) == 0:
			helpers.Println()
			helpers.Printf("No matching %s options found for %q.\n", label, searchQuery)
			helpers.Println("Type another search term, `clear` to reset, or `exit` to cancel.")
		default:
			maxVisible := len(filteredOptions)
			if searchRequired || searchQuery != "" {
				if visibleCount > maxVisible {
					visibleCount = maxVisible
				}
				maxVisible = visibleCount
			}

			helpers.Println()
			if searchQuery == "" {
				helpers.Printf("Select %s (%d of %d)\n", label, maxVisible, len(filteredOptions))
			} else {
				helpers.Printf("Matching %s options for %q (%d of %d)\n", label, searchQuery, maxVisible, len(filteredOptions))
			}
			helpers.Println()
			helpers.PrintTable(BuildOptionTable(filteredOptions[:maxVisible]), 1)
			helpers.Println()

			if maxVisible < len(filteredOptions) {
				helpers.Printf("Type `m` to show %d more.\n", min(selectionPageSize, len(filteredOptions)-maxVisible))
			}
		}

		prompt := "Enter a number, search term, `m`, `clear`, or `exit`: "
		if searchRequired && searchQuery == "" {
			prompt = "Enter a search term or `exit`: "
		} else if !searchRequired && searchQuery == "" {
			prompt = "Enter a number, search term, `clear`, or `exit`: "
		}
		helpers.Print(prompt)
		rawInput, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		trimmed := strings.TrimSpace(rawInput)
		if isCancelInput(trimmed) {
			return "", ErrUserCanceled
		}
		lowerTrimmed := strings.ToLower(trimmed)

		switch lowerTrimmed {
		case "":
			continue
		case "clear":
			searchQuery = ""
			if searchRequired {
				visibleCount = selectionPageSize
			} else {
				visibleCount = len(options)
			}
			continue
		case "m", "more":
			filteredOptions := FilterOptions(options, searchQuery)
			if len(filteredOptions) == 0 {
				continue
			}
			if !searchRequired && searchQuery == "" {
				helpers.Println("All matching options are already shown.")
				continue
			}
			if visibleCount >= len(filteredOptions) {
				helpers.Println("All matching options are already shown.")
				continue
			}
			visibleCount += selectionPageSize
			if visibleCount > len(filteredOptions) {
				visibleCount = len(filteredOptions)
			}
			continue
		}

		maxVisible := len(filteredOptions)
		if searchRequired || searchQuery != "" {
			if visibleCount < maxVisible {
				maxVisible = visibleCount
			}
		}

		choice, err := strconv.Atoi(trimmed)
		if err == nil && choice >= 1 && choice <= maxVisible {
			return filteredOptions[choice-1].Value, nil
		}

		if !searchRequired && searchQuery == "" && err == nil {
			helpers.Printf("Invalid selection. Please enter a number between 1 and %d, or type a search term.\n", len(options))
			continue
		}

		searchQuery = trimmed
		visibleCount = selectionPageSize
	}
}

func BuildOptionTable(options []FormOption) [][]string {
	table := [][]string{{"Option"}}
	for _, option := range options {
		displayLabel := option.Label
		if displayLabel == "" {
			displayLabel = option.Value
		}
		table = append(table, []string{displayLabel})
	}
	return table
}

func FilterOptions(options []FormOption, searchQuery string) []FormOption {
	query := strings.TrimSpace(searchQuery)
	if query == "" {
		return append([]FormOption(nil), options...)
	}

	searchTable := buildOptionSearchTable(options)
	exactMatches := make([]FormOption, 0, len(options))
	seen := make(map[int]struct{}, len(options))
	appendExactMatch := func(index int) {
		if index < 1 || index > len(options) {
			return
		}
		if _, exists := seen[index]; exists {
			return
		}
		seen[index] = struct{}{}
		exactMatches = append(exactMatches, options[index-1])
	}

	queryLower := strings.ToLower(query)
	for index, option := range options {
		if strings.Contains(strings.ToLower(option.Label), queryLower) || strings.Contains(strings.ToLower(option.Value), queryLower) {
			appendExactMatch(index + 1)
		}
	}
	if len(exactMatches) > 0 {
		return exactMatches
	}

	acronymMatches := make([]FormOption, 0, len(options))
	for _, index := range helpers.FuzzySearchWithAcronym(searchTable, query) {
		if index < 1 || index > len(options) {
			continue
		}
		acronymMatches = append(acronymMatches, options[index-1])
	}
	if len(acronymMatches) > 0 {
		return acronymMatches
	}

	fuzzyMatches := make([]FormOption, 0, len(options))
	for _, index := range helpers.NewFuzzySearch(searchTable, query) {
		if index < 1 || index > len(options) {
			continue
		}
		fuzzyMatches = append(fuzzyMatches, options[index-1])
	}

	return fuzzyMatches
}

func HasPendingRequest(requests []types.NightSlipRequest) bool {
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

func NormalizeApplyInput(input ApplyInput) (ApplyInput, error) {
	var err error

	input.CostCentreID = strings.TrimSpace(input.CostCentreID)
	input.AppliedTo = strings.TrimSpace(input.AppliedTo)
	input.RoomTypeID = strings.TrimSpace(input.RoomTypeID)
	input.LateHourEventID = strings.TrimSpace(input.LateHourEventID)
	input.Details = helpers.SanitizeString(strings.TrimSpace(input.Details))
	if input.Details == "" {
		return ApplyInput{}, fmt.Errorf("missing --details")
	}

	input.FromDate, err = normalizeDate(input.FromDate, "--from-date")
	if err != nil {
		return ApplyInput{}, err
	}
	input.FromTime, err = normalizeTime(input.FromTime, "--from-time")
	if err != nil {
		return ApplyInput{}, err
	}
	input.ToDate, err = normalizeDate(input.ToDate, "--to-date")
	if err != nil {
		return ApplyInput{}, err
	}
	input.ToTime, err = normalizeTime(input.ToTime, "--to-time")
	if err != nil {
		return ApplyInput{}, err
	}

	if input.FromDate == input.ToDate {
		fromParsed, fromErr := time.Parse("03:04 PM", input.FromTime)
		toParsed, toErr := time.Parse("03:04 PM", input.ToTime)
		if fromErr == nil && toErr == nil && !toParsed.After(fromParsed) {
			return ApplyInput{}, fmt.Errorf("--to-time must be after --from-time when both dates are the same")
		}
	}

	return input, nil
}

func ResolveField(scope *goquery.Selection, fieldName string, providedValue string, flagName string) (string, error) {
	providedValue = strings.TrimSpace(providedValue)
	if providedValue != "" {
		return providedValue, nil
	}

	if value := strings.TrimSpace(scope.Find(fmt.Sprintf("input[name='%s']", fieldName)).First().AttrOr("value", "")); value != "" {
		return value, nil
	}

	fieldSelection := scope.Find(fmt.Sprintf("select[name='%s']", fieldName)).First()
	if fieldSelection.Length() == 0 {
		return "", fmt.Errorf("missing %s and VTOP did not expose %s in the apply form", flagName, fieldName)
	}

	options, selectedValue := CollectOptions(fieldSelection)
	if selectedValue != "" {
		return selectedValue, nil
	}
	if len(options) == 1 {
		return options[0].Value, nil
	}
	if len(options) == 0 {
		return "", fmt.Errorf("missing %s and VTOP returned no usable options for %s", flagName, fieldName)
	}

	return "", fmt.Errorf("missing %s; available %s values: %s", flagName, fieldName, formatOptions(options))
}

func CollectOptions(selection *goquery.Selection) ([]FormOption, string) {
	options := make([]FormOption, 0)
	selectedValue := ""

	selection.Find("option").Each(func(_ int, option *goquery.Selection) {
		value := strings.TrimSpace(option.AttrOr("value", ""))
		label := helpers.SanitizeString(strings.TrimSpace(option.Text()))
		if value == "" {
			return
		}

		options = append(options, FormOption{
			Value: value,
			Label: label,
		})

		if selectedValue == "" {
			if _, selected := option.Attr("selected"); selected {
				selectedValue = value
			}
		}
	})

	return options, selectedValue
}

func ExtractMessage(body []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return ""
	}

	selectors := []string{
		"#DataBlockMessage",
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

func buildOptionSearchTable(options []FormOption) [][]string {
	table := [][]string{{"Option", "Value"}}
	for _, option := range options {
		displayLabel := option.Label
		if displayLabel == "" {
			displayLabel = option.Value
		}
		table = append(table, []string{displayLabel, option.Value})
	}
	return table
}

func isCancelInput(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "exit", "cancel":
		return true
	default:
		return false
	}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	const prefix = "REQUEST RAISED-"
	if strings.HasPrefix(strings.ToUpper(status), prefix) {
		status = strings.TrimSpace(status[len(prefix):])
	}
	return status
}

func normalizeDate(value string, flagName string) (string, error) {
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

func normalizeTime(value string, flagName string) (string, error) {
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

func formatOptions(options []FormOption) string {
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
