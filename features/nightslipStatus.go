package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func GetNightSlipStatus(regNo string, cookies types.Cookies) {
	url := "https://vtop.vit.ac.in/vtop/hostels/late/hour/student/request/9"

	// Using 'nightslip' in the payload to indicate the request type
	payload := "nightslipRequest"

	// Fetch the page with the modified payload
	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching data:", err)
		}
		return
	}

	// Parse the HTML content
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing HTML:", err)
		}
		return
	}

	// If no error, proceed with processing table data
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Event Type", "Details", "Applied To", "From Date", "To Date", "From/To Time", "Status", "Remarks"})
	table.SetAutoWrapText(false) 
	table.SetAlignment(tablewriter.ALIGN_LEFT) 


	doc.Find("table tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		row := []string{}
		rowSelection.Find("td.text-primary").Each(func(j int, cellSelection *goquery.Selection) {
			cellText := strings.TrimSpace(cellSelection.Text())
			row = append(row, cellText)
		})

		if len(row) > 0 {
			table.Append(row)
		}
	})

	table.Render()
}
