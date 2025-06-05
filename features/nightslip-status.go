package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	NightSlipTableSelector = "table#LateHourStatusTable"
	NightSlipRowsSelector  = "tbody tr"
	NightSlipCellSelector  = "td"
)

func GetNightSlipStatus(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML:", err)
		}
		return
	}

	var nightSlipRequests []types.NightSlipRequest

	doc.Find(NightSlipTableSelector).Find(NightSlipRowsSelector).Each(func(i int, rowSelection *goquery.Selection) {
		venue := strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(2).Text())
		eventType := strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(3).Text())
		details := strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(4).Text())
		appliedTo := strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(5).Text())
		fromDate := helpers.FormatDate(strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(6).Text()))
		toDate := helpers.FormatDate(strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(7).Text()))
		fromToTime := strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(8).Text())
		status := strings.TrimSpace(rowSelection.Find(NightSlipCellSelector).Eq(9).Text())

		status = strings.Replace(status, "REQUEST RAISED-", "", -1)

		coloredStatus := helpers.ColorStatus(status)

		if venue != "" {
			nightSlipRequests = append(nightSlipRequests, types.NightSlipRequest{
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
