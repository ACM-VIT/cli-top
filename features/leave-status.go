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
)

const (
	LeaveStatusTableSelector = "table#LeaveAppliedTable"
	LeaveStatusRowsSelector  = "tbody tr"
	LeaveStatusCellSelector  = "td"
)

func GetLeaveStatus(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
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

	var leaveRequests []types.LeaveRequest

	doc.Find(LeaveStatusTableSelector).Find(LeaveStatusRowsSelector).Each(func(i int, rowSelection *goquery.Selection) {
		reason := strings.TrimSpace(rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap").Eq(1).Text())
		visitPlace := strings.TrimSpace(rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap").Eq(0).Text())
		leaveType := strings.TrimSpace(rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap").Eq(2).Text())
		from := helpers.FormatDate(strings.TrimSpace(rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap").Eq(3).Text()))
		to := helpers.FormatDate(strings.TrimSpace(rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap").Eq(4).Text()))
		status := strings.TrimSpace(rowSelection.Find(LeaveStatusCellSelector + ".text-primary.text-nowrap").Eq(5).Text())
		coloredStatus := helpers.ColorStatus(status)
		if visitPlace != "" {
			leaveRequests = append(leaveRequests, types.LeaveRequest{
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
	fmt.Println()
}
