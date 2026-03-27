package features

import (
	"bufio"
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	eventsLoadURL               = "https://vtop.vit.ac.in/vtop/event/swf/loadEventRegistration"
	eventsRegisterURL           = "https://vtop.vit.ac.in/vtop/event/swf/registered/doEventRegistraiton"
	eventsUpcomingTableSelector = "table#dataTable1"
	eventsRegisteredSelector    = "table#dataTable2"
	eventSelectionPageSize      = 6
)

var eventRegistrationIDRegex = regexp.MustCompile(`doEventRegistraiton\(['"]([^'"]+)['"]\)`)
var associationTypeSuffixRegex = regexp.MustCompile(`\s+\((CLUB|CHAPTER)\)\s*$`)
var associationAcronymRegex = regexp.MustCompile(`^(.*)\(([^()]+)\)\s*$`)
var venueModeRegex = regexp.MustCompile(`^(.*?)(?:\s+\((ONLINE|OFFLINE)\))?$`)

func GetEvents(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	loc := time.FixedZone("IST", 5*3600+1800)
	now := time.Now().In(loc)

	upcomingEvents, registeredEvents, err := fetchEventsPage(regNo, cookies, loc)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error fetching events:", err)
		}
		helpers.Println("Error fetching events:", err)
		return
	}

	if len(upcomingEvents) == 0 && len(registeredEvents) == 0 {
		helpers.Println("No events found.")
		return
	}

	registeredUpcoming := filterCurrentAndUpcomingEvents(registeredEvents, now)
	registerableEvents := filterRegisterableEvents(upcomingEvents)

	if helpers.ShouldMuteUI() {
		if len(upcomingEvents) > 0 {
			printEventSection("Upcoming Events", upcomingEvents, now)
		}
		if len(registeredUpcoming) > 0 {
			helpers.Println()
			printEventSection("Your Upcoming Registrations", registeredUpcoming, now)
		} else if len(registeredEvents) > 0 {
			helpers.Printf("\nNo upcoming registered events. Historical registrations on VTOP: %d\n", len(registeredEvents))
		}
		return
	}

	printInteractiveEventsOverview(upcomingEvents, registeredUpcoming, registerableEvents, now)
	if len(registerableEvents) == 0 {
		return
	}

	selectedEvent, selected := promptEventSelection(registerableEvents, registeredUpcoming, now)
	if !selected {
		return
	}

	if !confirmEventRegistration(selectedEvent) {
		return
	}

	updatedUpcoming, updatedRegistered, successMessage, errorMessage, err := registerForEvent(regNo, cookies, selectedEvent, loc)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error registering for event:", err)
		}
		helpers.Println("Error registering for event:", err)
		return
	}

	if errorMessage != "" {
		helpers.Println(errorMessage)
		return
	}

	if successMessage != "" {
		helpers.Println()
		helpers.Println(successMessage)
	}

	printEventRegistrationResult(selectedEvent, updatedUpcoming, updatedRegistered, time.Now().In(loc))
}

func ParseEventsHTML(html string, loc *time.Location) ([]types.Event, []types.Event, error) {
	if loc == nil {
		loc = time.FixedZone("IST", 5*3600+1800)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, nil, err
	}
	return parseEventsDocument(doc, loc), parseRegisteredEventsDocument(doc, loc), nil
}

func ParseEventRegistrationResultHTML(html string) (string, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "", err
	}

	successMessage := helpers.SanitizeString(doc.Find("#error1").First().Text())
	errorMessage := helpers.SanitizeString(doc.Find("#error2").First().Text())
	return successMessage, errorMessage, nil
}

func fetchEventsPage(regNo string, cookies types.Cookies, loc *time.Location) ([]types.Event, []types.Event, error) {
	payloadMap := map[string]string{
		"verifyMenu":   "true",
		"authorizedID": regNo,
		"_csrf":        cookies.CSRF,
		"nocache":      fmt.Sprintf("%d", time.Now().UnixMilli()),
	}

	body, err := helpers.FetchReq(
		regNo,
		cookies,
		eventsLoadURL,
		"",
		helpers.FormatBodyData(payloadMap),
		"POST",
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, nil, err
	}

	return ParseEventsHTML(string(body), loc)
}

func registerForEvent(regNo string, cookies types.Cookies, event types.Event, loc *time.Location) ([]types.Event, []types.Event, string, string, error) {
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"eventId":      event.EventID,
		"authorizedID": regNo,
		"x":            formatEventRequestTime(time.Now().UTC()),
	}

	body, err := helpers.FetchReq(
		regNo,
		cookies,
		eventsRegisterURL,
		"",
		helpers.FormatBodyData(payloadMap),
		"POST",
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, nil, "", "", err
	}

	successMessage, errorMessage, err := ParseEventRegistrationResultHTML(string(body))
	if err != nil {
		return nil, nil, "", "", err
	}

	upcomingEvents, registeredEvents, err := ParseEventsHTML(string(body), loc)
	if err != nil {
		return nil, nil, successMessage, errorMessage, err
	}

	return upcomingEvents, registeredEvents, successMessage, errorMessage, nil
}

func parseEventsDocument(doc *goquery.Document, loc *time.Location) []types.Event {
	return parseEventTable(doc.Find(eventsUpcomingTableSelector).First(), false, loc)
}

func parseRegisteredEventsDocument(doc *goquery.Document, loc *time.Location) []types.Event {
	return parseEventTable(doc.Find(eventsRegisteredSelector).First(), true, loc)
}

func parseEventTable(table *goquery.Selection, registered bool, loc *time.Location) []types.Event {
	if table.Length() == 0 {
		return nil
	}

	var events []types.Event
	table.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		event, ok := parseEventRow(row, registered, loc)
		if ok {
			events = append(events, event)
		}
	})

	sort.Slice(events, func(i, j int) bool {
		if events[i].StartDateTime.Equal(events[j].StartDateTime) {
			return events[i].Title < events[j].Title
		}
		return events[i].StartDateTime.Before(events[j].StartDateTime)
	})

	return events
}

func parseEventRow(row *goquery.Selection, registered bool, loc *time.Location) (types.Event, bool) {
	cells := row.Find("td")
	if cells.Length() < 10 {
		return types.Event{}, false
	}

	number := helpers.StrToInt(helpers.SanitizeString(cells.Eq(0).Text()))
	association := normalizeNullableField(helpers.SanitizeString(cells.Eq(1).Text()))
	title := helpers.SanitizeString(cells.Eq(2).Text())
	description := extractEventDescription(cells.Eq(3))

	startDateText, endDateText := extractSpanRange(cells.Eq(4))
	startTimeText, endTimeText := extractSpanRange(cells.Eq(5))
	startDateTime, endDateTime := parseEventDateTimeRange(startDateText, endDateText, startTimeText, endTimeText, loc)

	registrationStartText, registrationEndText := extractSpanRange(cells.Eq(7))
	registrationStart := parseDateOnly(registrationStartText, loc, false)
	registrationEnd := parseDateOnly(registrationEndText, loc, true)

	venue := helpers.SanitizeString(cells.Eq(8).Text())
	statusCell := cells.Eq(9)
	eventID := extractEventID(statusCell)
	if eventID == "" {
		eventID = extractEventID(cells.Eq(3))
	}

	registerStatus, canRegister := extractEventStatus(statusCell, registered)
	if registered && registerStatus == "" {
		registerStatus = "Registered"
	}

	now := time.Now().In(loc)
	daysLeft := 0.0
	if !startDateTime.IsZero() {
		eventStartDay := time.Date(startDateTime.Year(), startDateTime.Month(), startDateTime.Day(), 0, 0, 0, 0, loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		daysLeft = eventStartDay.Sub(today).Hours() / 24
	}

	return types.Event{
		EventID:              eventID,
		Number:               number,
		Association:          association,
		Title:                title,
		Description:          description,
		StartDateTime:        startDateTime,
		EndDateTime:          endDateTime,
		RegistrationStart:    registrationStart,
		DaysLeft:             daysLeft,
		RegistrationDeadline: registrationEnd,
		Venue:                venue,
		RegisterStatus:       registerStatus,
		CanRegister:          canRegister,
	}, title != ""
}

func extractEventDescription(cell *goquery.Selection) string {
	description := helpers.SanitizeString(cell.Find("div.modal-body").First().Text())
	if description != "" {
		return description
	}
	return helpers.SanitizeString(cell.Text())
}

func extractSpanRange(cell *goquery.Selection) (string, string) {
	values := make([]string, 0, 2)
	seen := make(map[string]struct{})

	cell.Find("span").Each(func(_ int, span *goquery.Selection) {
		text := helpers.SanitizeString(span.Text())
		if text == "" {
			return
		}
		if _, exists := seen[text]; exists {
			return
		}
		seen[text] = struct{}{}
		values = append(values, text)
	})

	if len(values) == 0 {
		text := helpers.SanitizeString(cell.Text())
		if text == "" {
			return "", ""
		}
		return text, text
	}
	if len(values) == 1 {
		return values[0], values[0]
	}
	return values[0], values[1]
}

func parseEventDateTimeRange(startDateText string, endDateText string, startTimeText string, endTimeText string, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.Local
	}

	startDate := parseDateOnly(startDateText, loc, false)
	endDate := parseDateOnly(endDateText, loc, false)
	if startDate.IsZero() || endDate.IsZero() {
		return time.Time{}, time.Time{}
	}

	startTime := parseClockTime(startTimeText)
	endTime := parseClockTime(endTimeText)

	startDateTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
	endDateTime := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)

	if endDateTime.Before(startDateTime) && sameDay(startDateTime, endDateTime) {
		endDateTime = endDateTime.Add(24 * time.Hour)
	}

	return startDateTime, endDateTime
}

func parseDateOnly(value string, loc *time.Location, inclusiveEnd bool) time.Time {
	if loc == nil {
		loc = time.Local
	}

	layouts := []string{"02-Jan-2006", "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, loc)
		if err == nil {
			if inclusiveEnd {
				return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, loc)
			}
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
		}
	}

	return time.Time{}
}

func parseClockTime(value string) time.Time {
	parsed, err := time.Parse("03:04 PM", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func extractEventStatus(cell *goquery.Selection, registered bool) (string, bool) {
	if !registered {
		button := cell.Find("button").First()
		if button.Length() > 0 {
			buttonText := helpers.SanitizeString(button.Text())
			if strings.EqualFold(buttonText, "Register") {
				return "Open", true
			}
		}
	}

	status := helpers.SanitizeString(cell.Text())
	return status, false
}

func extractEventID(cell *goquery.Selection) string {
	onclick, exists := cell.Find("button").First().Attr("onclick")
	if !exists {
		onclick = ""
	}
	matches := eventRegistrationIDRegex.FindStringSubmatch(onclick)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func buildEventsTable(events []types.Event, now time.Time) [][]string {
	table := [][]string{{"In", "When", "Title", "Club", "Where", "Status"}}
	for _, event := range events {
		in := "-"
		if !event.StartDateTime.IsZero() {
			in = formatDaysUntil(event.StartDateTime, now)
		}
		table = append(table, []string{
			in,
			formatEventWhenCompact(event),
			helpers.TruncateWithEllipsis(event.Title, 42),
			formatAssociationCompact(event.Association),
			formatVenueCompact(event.Venue),
			formatEventStatusCompact(event.RegisterStatus),
		})
	}
	return table
}

func buildEventSelectionTable(events []types.Event) [][]string {
	table := [][]string{{"Title", "When", "Club", "Where", "Closes"}}
	for _, event := range events {
		table = append(table, []string{
			event.Title,
			formatEventWhenCompact(event),
			formatAssociationCompact(event.Association),
			formatVenueCompact(event.Venue),
			formatRegistrationDeadlineCompact(event),
		})
	}
	return table
}

func filterCurrentAndUpcomingEvents(events []types.Event, now time.Time) []types.Event {
	filtered := make([]types.Event, 0, len(events))
	for _, event := range events {
		if event.EndDateTime.IsZero() {
			continue
		}
		if event.EndDateTime.Before(now) {
			continue
		}
		filtered = append(filtered, event)
	}
	return filtered
}

func filterRegisterableEvents(events []types.Event) []types.Event {
	filtered := make([]types.Event, 0, len(events))
	for _, event := range events {
		if event.CanRegister && event.EventID != "" {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func printInteractiveEventsOverview(upcomingEvents []types.Event, registeredUpcoming []types.Event, registerableEvents []types.Event, now time.Time) {
	helpers.Println()
	helpers.Printf(
		"Upcoming events: %d | Open registrations: %d | Your upcoming registrations: %d\n",
		len(upcomingEvents),
		len(registerableEvents),
		len(registeredUpcoming),
	)

	if nextRegistration, ok := findNextUpcomingEvent(registeredUpcoming); ok {
		helpers.Printf(
			"Next registered: %s — %s\n",
			helpers.TruncateWithEllipsis(nextRegistration.Title, 54),
			formatEventWhenInline(nextRegistration),
		)
	}

	if nextDeadline, ok := findClosestRegistrationDeadline(registerableEvents, now); ok {
		helpers.Printf(
			"Closest deadline: %s — %s\n",
			helpers.TruncateWithEllipsis(nextDeadline.Title, 54),
			formatRegistrationDeadline(nextDeadline),
		)
	}

	shortlist := previewEvents(upcomingEvents, eventSelectionPageSize)
	sectionTitle := "Coming up"
	totalCount := len(upcomingEvents)
	if len(registerableEvents) > 0 {
		shortlist = previewEvents(registerableEvents, eventSelectionPageSize)
		sectionTitle = "Open registrations"
		totalCount = len(registerableEvents)
	}

	if len(shortlist) == 0 {
		return
	}

	helpers.Println()
	helpers.Printf("%s (%d of %d)\n", sectionTitle, len(shortlist), totalCount)
	helpers.Println()
	helpers.PrintTable(buildEventSelectionTable(shortlist), 1)
	helpers.Println()

	if len(registerableEvents) > len(shortlist) {
		helpers.Printf("Type `m` to show %d more open events.\n", minInt(eventSelectionPageSize, len(registerableEvents)-len(shortlist)))
	}
	if len(registerableEvents) > 0 {
		helpers.Println("Choose from the shortlist, search open registrations, type `mine` to review your registrations, `clear` to reset, or `exit`.")
	}
}

func promptEventSelection(events []types.Event, registeredUpcoming []types.Event, now time.Time) (types.Event, bool) {
	if len(events) == 0 {
		return types.Event{}, false
	}

	reader := bufio.NewReader(os.Stdin)
	searchQuery := ""
	visibleCount := minInt(eventSelectionPageSize, len(events))

	for {
		helpers.Print("Choose an event, search, `m`, `mine`, `clear`, or `exit`: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if debug.Debug {
				helpers.Println("Error reading event selection:", err)
			}
			return types.Event{}, false
		}

		input = strings.TrimSpace(input)
		lowerInput := strings.ToLower(input)

		switch lowerInput {
		case "":
			continue
		case "exit":
			return types.Event{}, false
		case "mine":
			printRegisteredEventPreview(registeredUpcoming, now)
			continue
		case "clear":
			searchQuery = ""
			visibleCount = minInt(eventSelectionPageSize, len(events))
			printEventPickerView(events, searchQuery, visibleCount)
			continue
		case "m", "more":
			filteredEvents := filterEventsForSelection(events, searchQuery)
			if len(filteredEvents) == 0 {
				continue
			}
			if visibleCount >= len(filteredEvents) {
				helpers.Println("All matching open events are already shown.")
				continue
			}
			visibleCount += eventSelectionPageSize
			if visibleCount > len(filteredEvents) {
				visibleCount = len(filteredEvents)
			}
			printEventPickerView(filteredEvents, searchQuery, visibleCount)
			continue
		}

		filteredEvents := filterEventsForSelection(events, searchQuery)
		if choice, err := strconv.Atoi(input); err == nil {
			if len(filteredEvents) == 0 {
				helpers.Println("There are no visible open events to select.")
				continue
			}
			if choice < 1 || choice > visibleCount {
				helpers.Printf("Invalid choice. Please enter a number between 1 and %d.\n", visibleCount)
				continue
			}
			return filteredEvents[choice-1], true
		}

		searchQuery = input
		filteredEvents = filterEventsForSelection(events, searchQuery)
		if len(filteredEvents) == 0 {
			helpers.Printf("\nNo matching open events found for %q.\n", searchQuery)
			helpers.Println("Try another search term, `clear` to reset, or `exit` to quit.")
			continue
		}

		visibleCount = minInt(eventSelectionPageSize, len(filteredEvents))
		printEventPickerView(filteredEvents, searchQuery, visibleCount)
	}
}

func filterEventsForSelection(events []types.Event, searchQuery string) []types.Event {
	if strings.TrimSpace(searchQuery) == "" {
		return events
	}

	table := buildEventSelectionTable(events)
	matchedIndices := helpers.NewFuzzySearch(table, searchQuery)
	filtered := make([]types.Event, 0, len(matchedIndices))
	seen := make(map[int]struct{}, len(matchedIndices))

	for _, idx := range matchedIndices {
		if idx < 1 || idx > len(events) {
			continue
		}
		if _, exists := seen[idx]; exists {
			continue
		}
		seen[idx] = struct{}{}
		filtered = append(filtered, events[idx-1])
	}

	return filtered
}

func confirmEventRegistration(event types.Event) bool {
	helpers.Println()
	helpers.Println("Register for this event?")
	helpers.Println()
	helpers.PrintTable(buildEventReviewTable(event), 0)
	description := formatEventDescriptionPreview(event.Description)
	if description != "" {
		helpers.Println()
		helpers.Printf("About: %s\n", description)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		helpers.Print("Proceed with registration? (yes/no): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if debug.Debug {
				helpers.Println("Error reading confirmation:", err)
			}
			return false
		}

		switch strings.ToLower(strings.TrimSpace(input)) {
		case "yes", "y":
			return true
		case "no", "n", "exit":
			return false
		default:
			helpers.Println("Invalid input. Please respond with 'yes' or 'no'.")
		}
	}
}

func printEventPickerView(events []types.Event, searchQuery string, visibleCount int) {
	if len(events) == 0 {
		return
	}
	if visibleCount > len(events) {
		visibleCount = len(events)
	}

	helpers.Println()
	if searchQuery == "" {
		helpers.Printf("Open registrations (%d of %d)\n", visibleCount, len(events))
	} else {
		helpers.Printf("Matches for %q (%d of %d)\n", searchQuery, visibleCount, len(events))
	}
	helpers.Println()
	helpers.PrintTable(buildEventSelectionTable(events[:visibleCount]), 1)
	helpers.Println()

	if visibleCount < len(events) {
		helpers.Printf("Type `m` to show %d more.\n", minInt(eventSelectionPageSize, len(events)-visibleCount))
	}
}

func printRegisteredEventPreview(events []types.Event, now time.Time) {
	helpers.Println()
	if len(events) == 0 {
		helpers.Println("You have no upcoming registrations.")
		return
	}

	visibleCount := minInt(eventSelectionPageSize, len(events))
	helpers.Printf("Your upcoming registrations (%d of %d)\n", visibleCount, len(events))
	helpers.Println()
	helpers.PrintTable(buildEventsTable(events[:visibleCount], now), 0)
	helpers.Println()
	helpers.Println("Search still applies to open registrations.")
}

func buildEventReviewTable(event types.Event) [][]string {
	return [][]string{
		{"Event", "When", "Club", "Where", "Closes"},
		{
			helpers.TruncateWithEllipsis(event.Title, 64),
			formatEventWhenInline(event),
			formatAssociationCompact(event.Association),
			formatVenueCompact(event.Venue),
			formatRegistrationDeadline(event),
		},
	}
}

func printEventRegistrationResult(selectedEvent types.Event, updatedUpcoming []types.Event, updatedRegistered []types.Event, now time.Time) {
	helpers.Println()
	if registeredEvent, ok := findMatchingEvent(updatedRegistered, selectedEvent); ok {
		helpers.Println("Registered event")
		helpers.Println()
		helpers.PrintTable(buildEventReviewTable(registeredEvent), 0)
		return
	}

	if updatedEvent, ok := findMatchingEvent(updatedUpcoming, selectedEvent); ok {
		helpers.Println("Updated event status")
		helpers.Println()
		helpers.PrintTable(buildEventsTable([]types.Event{updatedEvent}, now), 0)
		return
	}

	helpers.Println("Registration submitted.")
}

func previewEvents(events []types.Event, limit int) []types.Event {
	if len(events) == 0 {
		return nil
	}
	if limit <= 0 || len(events) <= limit {
		return append([]types.Event(nil), events...)
	}
	return append([]types.Event(nil), events[:limit]...)
}

func findNextUpcomingEvent(events []types.Event) (types.Event, bool) {
	for _, event := range events {
		if !event.StartDateTime.IsZero() {
			return event, true
		}
	}
	return types.Event{}, false
}

func findClosestRegistrationDeadline(events []types.Event, now time.Time) (types.Event, bool) {
	var closest types.Event
	found := false

	for _, event := range events {
		if event.RegistrationDeadline.IsZero() || event.RegistrationDeadline.Before(now) {
			continue
		}
		if !found || event.RegistrationDeadline.Before(closest.RegistrationDeadline) {
			closest = event
			found = true
		}
	}

	return closest, found
}

func findMatchingEvent(events []types.Event, selectedEvent types.Event) (types.Event, bool) {
	for _, event := range events {
		if selectedEvent.EventID != "" && event.EventID == selectedEvent.EventID {
			return event, true
		}
		if event.Title == selectedEvent.Title && event.StartDateTime.Equal(selectedEvent.StartDateTime) {
			return event, true
		}
	}
	return types.Event{}, false
}

func formatEventWhenInline(event types.Event) string {
	dateRange := formatEventDateRange(event)
	timeRange := formatEventTimeRange(event)

	switch {
	case dateRange == "-" && timeRange == "-":
		return "-"
	case dateRange == "-":
		return timeRange
	case timeRange == "-":
		return dateRange
	default:
		return fmt.Sprintf("%s, %s", dateRange, timeRange)
	}
}

func formatEventDescriptionPreview(value string) string {
	value = helpers.SanitizeString(strings.ReplaceAll(value, "\n", " "))
	if value == "" || value == "-" {
		return ""
	}
	return helpers.TruncateWithEllipsis(value, 180)
}

func formatEventDateRange(event types.Event) string {
	if event.StartDateTime.IsZero() || event.EndDateTime.IsZero() {
		return "-"
	}
	if sameDay(event.StartDateTime, event.EndDateTime) {
		return event.StartDateTime.Format("02 Jan 2006")
	}
	return fmt.Sprintf("%s - %s", event.StartDateTime.Format("02 Jan 2006"), event.EndDateTime.Format("02 Jan 2006"))
}

func formatEventTimeRange(event types.Event) string {
	if event.StartDateTime.IsZero() || event.EndDateTime.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%s-%s", event.StartDateTime.Format("03:04 PM"), event.EndDateTime.Format("03:04 PM"))
}

func formatRegistrationDeadline(event types.Event) string {
	if event.RegistrationDeadline.IsZero() {
		return "-"
	}
	return event.RegistrationDeadline.Format("02 Jan 2006")
}

func formatRegistrationDeadlineCompact(event types.Event) string {
	if event.RegistrationDeadline.IsZero() {
		return "-"
	}
	return event.RegistrationDeadline.Format("02 Jan")
}

func sameDay(first time.Time, second time.Time) bool {
	return first.Year() == second.Year() &&
		first.Month() == second.Month() &&
		first.Day() == second.Day()
}

func normalizeNullableField(value string) string {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	switch lowerValue {
	case "", "null", "null (null)":
		return "-"
	default:
		return value
	}
}

func formatEventRequestTime(t time.Time) string {
	return t.Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

func printEventSection(title string, events []types.Event, now time.Time) {
	helpers.Println(title)
	helpers.Println()
	helpers.PrintTable(buildEventsTable(events, now), 0)
}

func formatEventWhenCompact(event types.Event) string {
	dateRange := formatEventDateRangeCompact(event)
	timeRange := formatEventTimeRange(event)

	switch {
	case dateRange == "-" && timeRange == "-":
		return "-"
	case dateRange == "-":
		return timeRange
	case timeRange == "-":
		return dateRange
	default:
		return fmt.Sprintf("%s\n%s", dateRange, timeRange)
	}
}

func formatEventDateRangeCompact(event types.Event) string {
	if event.StartDateTime.IsZero() || event.EndDateTime.IsZero() {
		return "-"
	}
	if sameDay(event.StartDateTime, event.EndDateTime) {
		return event.StartDateTime.Format("02 Jan")
	}
	if event.StartDateTime.Year() == event.EndDateTime.Year() && event.StartDateTime.Month() == event.EndDateTime.Month() {
		return fmt.Sprintf("%s-%s %s", event.StartDateTime.Format("02"), event.EndDateTime.Format("02"), event.EndDateTime.Format("Jan"))
	}
	return fmt.Sprintf("%s-%s", event.StartDateTime.Format("02 Jan"), event.EndDateTime.Format("02 Jan"))
}

func formatAssociationCompact(value string) string {
	value = normalizeNullableField(value)
	if value == "-" {
		return value
	}

	withoutType := strings.TrimSpace(associationTypeSuffixRegex.ReplaceAllString(value, ""))
	matches := associationAcronymRegex.FindStringSubmatch(withoutType)
	if len(matches) == 3 {
		acronym := strings.TrimSpace(matches[2])
		if acronym != "" {
			return acronym
		}
	}

	return helpers.TruncateWithEllipsis(withoutType, 24)
}

func formatVenueCompact(value string) string {
	venue, _ := splitVenueMode(value)
	return helpers.TruncateWithEllipsis(venue, 28)
}

func splitVenueMode(value string) (string, string) {
	value = helpers.SanitizeString(value)
	if value == "" || value == "-" {
		return "-", ""
	}

	matches := venueModeRegex.FindStringSubmatch(value)
	if len(matches) != 3 {
		return value, ""
	}

	venue := strings.TrimSpace(matches[1])
	mode := strings.TrimSpace(matches[2])
	switch strings.ToUpper(mode) {
	case "ONLINE":
		mode = "Online"
	case "OFFLINE":
		mode = "Offline"
	}

	if venue == "" {
		venue = "-"
	}
	return venue, mode
}

func formatEventStatusCompact(status string) string {
	status = helpers.SanitizeString(status)
	switch strings.ToLower(status) {
	case "":
		return "-"
	case "limit exceeded":
		return "Full"
	default:
		return status
	}
}

func minInt(first int, second int) int {
	if first < second {
		return first
	}
	return second
}
