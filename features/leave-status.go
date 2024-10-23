// features/leave-status.go
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

// LeaveRequest represents a single leave request status
type LeaveRequest struct {
	VisitPlace string
	Reason     string
	LeaveType  string
	From       string
	To         string
	Status     string
}

// GetLeaveStatus fetches and displays the leave status for a given registration number
func GetLeaveStatus(regNo string, cookies types.Cookies) {
	// First POST request to initialize the session/menu
	url1 := "https://vtop.vit.ac.in/vtop/hostels/student/leave/1"
	payload1 := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d",
		regNo,
		cookies.CSRF,
		time.Now().UnixNano(),
	)
	_, err := helpers.FetchReq(regNo, cookies, url1, "", payload1, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching leave status menu:", err)
		}
		return
	}

	// Second POST request to fetch the actual data
	url2 := "https://vtop.vit.ac.in/vtop/hostels/student/leave/4"
	payload2 := fmt.Sprintf("_csrf=%s&authorizedID=%s&status=&form=undefined&control=status&x=%s",
		cookies.CSRF,
		regNo,
		time.Now().UTC().Format(time.RFC1123),
	)
	bodyText, err := helpers.FetchReq(regNo, cookies, url2, "", payload2, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching leave status data:", err)
		}
		return
	}

	// Parse the HTML response
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML:", err)
		}
		return
	}

	var leaveRequests []LeaveRequest

	// Traverse the table rows and extract data
	doc.Find("table#LeaveAppliedTable tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		reason := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(1).Text())
		visitPlace := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(0).Text())
		leaveType := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(2).Text())
		from := helpers.FormatDate(strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(3).Text()))
		to := helpers.FormatDate(strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(4).Text()))
		status := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(5).Text())
		coloredStatus := helpers.ColorStatus(status)
		if visitPlace != "" {
			leaveRequests = append(leaveRequests, LeaveRequest{
				VisitPlace: visitPlace,
				Reason:     reason,
				LeaveType:  leaveType,
				From:       from,
				To:         to,
				Status:     coloredStatus,
			})
		}
	})

	fmt.Println()
	if len(leaveRequests) == 0 {
		fmt.Println("No leave requests found.")
		return
	}
	var allRequests [][]string
	// Table headers
	allRequests = append(allRequests, []string{"VISIT PLACE", "REASON", "LEAVE TYPE", "FROM", "TO", "STATUS"})

	// Populate table rows
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
	fmt.Println()
}

// GenerateLeaveStatusTable creates a formatted table for Leave statuses using tablewriter
func GenerateLeaveStatusTable(leaveRequests []LeaveRequest) {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"VISIT PLACE", "REASON", "LEAVE TYPE", "FROM", "TO", "STATUS"})

	// Table formatting options
	table.SetBorder(false)
	table.SetHeaderLine(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnSeparator("│")

	const maxVisitPlaceLength = 30 // Define max length for "Visit Place" field

	for _, leave := range leaveRequests {
		visitPlace := helpers.TruncateWithEllipses(leave.VisitPlace, maxVisitPlaceLength)
		reason := leave.Reason
		leaveType := leave.LeaveType
		from := leave.From
		to := leave.To
		status := leave.Status

		table.Append([]string{visitPlace, reason, leaveType, from, to, status})
	}

	table.Render()
	output := buf.String()

	// Replace default table characters with box-drawing characters for better aesthetics
	output = strings.ReplaceAll(output, "+", "┼")
	output = strings.ReplaceAll(output, "-", "─")
	output = strings.ReplaceAll(output, "|", "│")

	// Add left padding for better readability
	output = helpers.AddLeftPadding(output, 2)

	fmt.Println("\n")
	fmt.Print(output)
	fmt.Println("\n")
}
