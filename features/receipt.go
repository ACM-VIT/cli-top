package features

import (
	"bytes"
	"cli-top/helpers"
	"cli-top/types"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func GetReceipt(regNo string, cookies types.Cookies) {
	url := "https://vtop.vit.ac.in/vtop/finance/getStudentReceipts"

	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil {
		log.Fatal("Error fetching data:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"SERIAL", "RECEIPT NUMBER", "DATE", "AMOUNT", "CAMPUS CODE"})

	// Skip the first row if it's not a header
	doc.Find("table.table-bordered tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		if i == 0 && strings.TrimSpace(rowSelection.Find("th").First().Text()) != "SERIAL" {
			return
		}

		// Extract data from each cell in the row
		row := []string{strconv.Itoa(i)}
		rowSelection.Find("td").Each(func(j int, cellSelection *goquery.Selection) {
			// Exclude the VIEW column
			if j < 4 {
				cellText := strings.TrimSpace(cellSelection.Text())
				row = append(row, cellText)
			}
		})
		// Append the row to the table
		table.Append(row)
	})

	// Render the table
	table.Render()
}
