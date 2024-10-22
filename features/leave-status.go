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

const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Reset  = "\033[0m" 
)

type LeaveRequest struct {
	VisitPlace string
	Reason     string
	LeaveType  string
	From       string
	To         string
	Status     string
}

func GetLeaveStatus(regNo string, cookies types.Cookies) {
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

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML:", err)
		}
		return
	}

	var leaveRequests []LeaveRequest

	doc.Find("table#LeaveAppliedTable tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		visitPlace := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(1).Text())
		reason := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(2).Text())
		leaveType := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(3).Text())
		from := formatDate(strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(4).Text()))
		to := formatDate(strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(5).Text()))
		status := strings.TrimSpace(rowSelection.Find("td.text-primary.text-nowrap").Eq(6).Text())

		if visitPlace != "" {
			leaveRequests = append(leaveRequests, LeaveRequest{
				VisitPlace: visitPlace,
				Reason:     reason,
				LeaveType:  leaveType,
				From:       from,
				To:         to,
				Status:     status,
			})
		}
	})

	GenerateLeaveStatusTable(leaveRequests)
}

func formatDate(dateStr string) string {
	parsedTime, err := time.Parse("02-Jan-2006 15:04", dateStr)
	if err != nil {
		return dateStr
	}
	return parsedTime.Format("02/01/06 15:04") 
}

func GenerateLeaveStatusTable(leaveRequests []LeaveRequest) {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"VISIT PLACE", "REASON", "LEAVE TYPE", "FROM", "TO", "STATUS"})

	table.SetBorder(false)
	table.SetHeaderLine(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnSeparator("│")

	const maxVisitPlaceLength = 30 // Define max length for "Visit Place" field

	for _, leave := range leaveRequests {
		visitPlace := truncateWithEllipses(leave.VisitPlace, maxVisitPlaceLength)
		reason := leave.Reason
		leaveType := leave.LeaveType
		from := leave.From
		to := leave.To
		status := colorStatus(leave.Status)

		table.Append([]string{reason, visitPlace, leaveType, from, to, status})
	}

	table.Render()
	output := buf.String()

	output = strings.ReplaceAll(output, "+", "┼")
	output = strings.ReplaceAll(output, "-", "─")
	output = strings.ReplaceAll(output, "|", "│")

	output = addLeftPadding(output, 2)

	fmt.Println("\n")
	fmt.Print(output)
	fmt.Println("\n")
}

func truncateWithEllipses(text string, maxLength int) string {
	if len(text) > maxLength {
		return text[:maxLength-3] + "..." // Add ellipses if text exceeds max length
	}
	return text
}

func colorStatus(status string) string {
	if strings.Contains(status, "APPROVAL PENDING") {
		return Red + status + Reset
	} else if strings.Contains(status, "APPROVED") {
		return Green + status + Reset
	}
	return status
}

func addLeftPadding(text string, padding int) string {
	paddingString := strings.Repeat(" ", padding)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = paddingString + line
	}
	return strings.Join(lines, "\n")
}