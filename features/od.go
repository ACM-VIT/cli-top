package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func Events(regNo string, cookies types.Cookies, url string, daysFlag int, allFlag bool) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		fmt.Println("Please login using the cli-top login command.")
		return
	}

	nocache := fmt.Sprintf("%d", time.Now().UnixMilli())

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%s",
		regNo,
		cookies.CSRF,
		nocache,
	)

	body, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		fmt.Println("Error fetching Events:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil && debug.Debug {
		fmt.Println("Error parsing HTML:", err)
		return
	}

	var events []types.Event
	currentDate := time.Now()

	doc.Find("#dataTable1 tbody tr").Each(func(i int, s *goquery.Selection) {
		// Parse dates and times
		eventDates := strings.Split(s.Find("td:nth-child(5)").Text(), " - ")
		eventTimes := strings.Split(s.Find("td:nth-child(6)").Text(), " - ")
		regDeadlines := strings.Split(s.Find("td:nth-child(8)").Text(), " - ")

		startDateTime, _ := time.Parse("02-Jan-2006 03:04 PM", fmt.Sprintf("%s %s", strings.TrimSpace(eventDates[0]), strings.TrimSpace(eventTimes[0])))
		endDateTime, _ := time.Parse("02-Jan-2006 03:04 PM", fmt.Sprintf("%s %s", strings.TrimSpace(eventDates[1]), strings.TrimSpace(eventTimes[1])))
		regDeadline, _ := time.Parse("2006-01-02", strings.TrimSpace(regDeadlines[1])) // Use end date of registration

		// Skip events based on daysFlag
		if daysFlag >= 0 {
			targetDate := currentDate.AddDate(0, 0, daysFlag)
			// Compare dates ignoring time
			if startDateTime.Format("2006-01-02") != targetDate.Format("2006-01-02") {
				return
			}
		}

		// Handle registration status
		regStatus := ""
		canRegister := false
		regTd := s.Find("td:nth-child(10)")

		if btn := regTd.Find("button"); btn.Length() > 0 {
			regStatus = strings.TrimSpace(btn.Text())
			canRegister = regStatus == "Register"
		} else if span := regTd.Find("span"); span.Length() > 0 {
			regStatus = strings.TrimSpace(span.Text())
			// Skip upcoming events
			if regStatus == "Upcoming" {
				return
			}
			// Clean up status text
			if regStatus == "ClosedLimit Exceeded" {
				regStatus = "Limit Exceeded"
			}
		}

		// Skip non-registerable events if allFlag is false
		if !allFlag && !canRegister {
			return
		}

		association := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
		association = strings.TrimSuffix(association, " (CLUB)")
		association = strings.TrimSuffix(association, " (CHAPTER)")

		event := types.Event{
			Number:               i + 1,
			Association:          association,
			Title:                strings.TrimSpace(s.Find("td:nth-child(3)").Text()),
			Description:          strings.TrimSpace(s.Find("td:nth-child(4) .modal-body").Text()),
			StartDateTime:        startDateTime,
			EndDateTime:          endDateTime,
			DaysLeft:             helpers.ParseFloat(s.Find("td:nth-child(7)").Text()),
			RegistrationDeadline: regDeadline,
			Venue:                strings.TrimSpace(s.Find("td:nth-child(9)").Text()),
			RegisterStatus:       regStatus,
			CanRegister:          canRegister,
		}

		events = append(events, event)
	})

	// Sort events by start date and duration
	sort.Slice(events, func(i, j int) bool {
		// Compare dates first
		if !events[i].StartDateTime.Equal(events[j].StartDateTime) {
			return events[i].StartDateTime.Before(events[j].StartDateTime)
		}
		// If same start date, compare duration
		iDuration := events[i].EndDateTime.Sub(events[i].StartDateTime)
		jDuration := events[j].EndDateTime.Sub(events[j].StartDateTime)
		return iDuration > jDuration
	})

	// Prepare table headers and rows
    headers := []string{"Event", "Association", "DateTime", "Venue", "Deadline"}

	if allFlag {
		headers = append(headers, "Status")
	}
	var rows [][]string
	rows = append(rows, headers)

	for _, event := range events {
		// Skip online events if allFlag is false
		if !allFlag && strings.Contains(strings.ToUpper(event.Venue), "ONLINE") {
			continue
		}

		// Clean up venue text when not showing all events
		venue := event.Venue
		if !allFlag {
			venue = strings.TrimSuffix(strings.TrimSuffix(venue, " (OFFLINE)"), " (ONLINE)")
		}

        truncate := func(str string, num int) string {
            if len(str) > num {
                return str[:num-3] + "..."
            }
            return str
        }

        row := []string{
            truncate(event.Title, 40),
            // truncate(event.Association, 40),
            event.Association,
            fmt.Sprintf("%s - %s",
            event.StartDateTime.Format("3:04 PM 02-Jan-2006"),
            event.EndDateTime.Format("3:04 PM 02-Jan-2006")),
            truncate(venue, 20),
            event.RegistrationDeadline.Format("02-Jan-2006"),
        }
		if allFlag {
			row = append(row, event.RegisterStatus)
		}
		rows = append(rows, row)
	}

	if len(rows) > 0 {
		fmt.Println()
		helpers.PrintTable(rows, 1)
		fmt.Println()
	} else {
		fmt.Println("\nNo events found.")
	}
}
