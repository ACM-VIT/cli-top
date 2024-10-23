// features/nightslip-status.go
package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

type NightSlipRequest struct {
	Venue      string
	EventType  string
	Details    string
	AppliedTo  string
	FromDate   string
	ToDate     string
	FromToTime string
	Status     string
}

func GetNightSlipStatus(regNo string, cookies types.Cookies) {
	url1 := "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/1"
	payload1 := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d",
		regNo,
		cookies.CSRF,
		time.Now().UnixNano(),
	)
	_, err := helpers.FetchReq(regNo, cookies, url1, "", payload1, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching night slip status menu:", err)
		}
		return
	}

	url2 := "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/9"
	payload2 := fmt.Sprintf("_csrf=%s&authorizedID=%s&status=&form=undefined&control=status&x=%s",
		cookies.CSRF,
		regNo,
		time.Now().UTC().Format(time.RFC1123),
	)
	bodyText, err := helpers.FetchReq(regNo, cookies, url2, "", payload2, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching night slip status data:", err)
		}
		return
	}

	if debug.Debug {
        fmt.Println("---- Response Body Start ----")
        fmt.Println(string(bodyText))
        fmt.Println("---- Response Body End ----")
    }

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML:", err)
		}
		return
	}

	var nightSlipRequests []NightSlipRequest

	doc.Find("table#LateHourStatusTable tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		venue := strings.TrimSpace(rowSelection.Find("td").Eq(2).Text())
		eventType := strings.TrimSpace(rowSelection.Find("td").Eq(3).Text())
		details := strings.TrimSpace(rowSelection.Find("td").Eq(4).Text())
		appliedTo := strings.TrimSpace(rowSelection.Find("td").Eq(5).Text())
		fromDate := helpers.FormatDate(strings.TrimSpace(rowSelection.Find("td").Eq(6).Text()))
		toDate := helpers.FormatDate(strings.TrimSpace(rowSelection.Find("td").Eq(7).Text()))
		fromToTime := strings.TrimSpace(rowSelection.Find("td").Eq(8).Text())
		status := strings.TrimSpace(rowSelection.Find("td").Eq(9).Text())
		coloredStatus := helpers.ColorStatus(status)

		// Only append if Venue is not empty (assuming Venue is mandatory)
		if venue != "" {
			nightSlipRequests = append(nightSlipRequests, NightSlipRequest{
				Venue:      venue,
				EventType:  eventType,
				Details:    details,
				AppliedTo:  appliedTo,
				FromDate:   fromDate,
				ToDate:     toDate,
				FromToTime: fromToTime,
				Status:     coloredStatus,
			})
		}
	})

	fmt.Println()
	if len(nightSlipRequests) == 0 {
		fmt.Println("No nightslip requests found.")
		fmt.Println("")
		return
	}

	var allRequests [][]string
	allRequests = append(allRequests, []string{"VENUE", "EVENT TYPE", "DETAILS", "APPLIED TO", "FROM DATE", "TO DATE", "FROM/TO TIME", "STATUS"})

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
	fmt.Println()
}

func GenerateNightSlipStatusTable(nightSlipRequests []NightSlipRequest) {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"VENUE", "EVENT TYPE", "DETAILS", "APPLIED TO", "FROM DATE", "TO DATE", "FROM/TO TIME", "STATUS"})

	table.SetBorder(false)
	table.SetHeaderLine(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnSeparator("│")

	const maxVenueLength = 30     
	const maxEventTypeLength = 20 
	const maxDetailsLength = 50   

	for _, slip := range nightSlipRequests {
		venue := helpers.TruncateWithEllipses(slip.Venue, maxVenueLength)
		eventType := helpers.TruncateWithEllipses(slip.EventType, maxEventTypeLength)
		details := helpers.TruncateWithEllipses(slip.Details, maxDetailsLength)
		appliedTo := slip.AppliedTo
		fromDate := slip.FromDate
		toDate := slip.ToDate
		fromToTime := slip.FromToTime
		status := slip.Status

		table.Append([]string{venue, eventType, details, appliedTo, fromDate, toDate, fromToTime, status})
	}

	table.Render()
	output := buf.String()

	output = strings.ReplaceAll(output, "+", "┼")
	output = strings.ReplaceAll(output, "-", "─")
	output = strings.ReplaceAll(output, "|", "│")

	output = helpers.AddLeftPadding(output, 2)

	fmt.Println("\n")
	fmt.Print(output)
	fmt.Println("\n")
}
