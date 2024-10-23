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

func GetNightSlipStatus(regNo string, cookies types.Cookies) {
	url := "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/9"

	// custom payload for 'nightslip'
	currentTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	currentTime = strings.ReplaceAll(currentTime, "UTC", "GMT")
	payload := fmt.Sprintf("authorizedID=%s&_csrf=%s&classId=%s&x=%s", regNo, cookies.CSRF, "nightslipRequest", currentTime)

	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching data:", err)
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

	var tableData [][]string

	tableData = append(tableData, []string{"Event Type", "Details", "Applied To", "From Date", "To Date", "From/To Time", "Status", "Remarks"})

	doc.Find("table tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		row := []string{}
		rowSelection.Find("td.text-primary").Each(func(j int, cellSelection *goquery.Selection) {
			cellText := strings.TrimSpace(cellSelection.Text())
			row = append(row, cellText)
		})

		if len(row) > 0 {
			tableData = append(tableData, row)
		}
	})
	fmt.Println()

	if len(tableData) > 1 {
		helpers.PrintTable(tableData,0)
	} else {
		fmt.Println("No night slip status found.")
	}
}
