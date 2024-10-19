package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"strconv"
	"strings"
	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func GetLibraryDues(regNo string, cookies types.Cookies) {
	url := "https://vtop.vit.ac.in/vtop/finance/libraryPayments"

	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println("Error fetching data:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil && debug.Debug {
		fmt.Println("Error parsing HTML:", err)
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"SERIAL", "TYPE", "AMOUNT"})

	doc.Find("table.table-bordered tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		// Extract data from each cell in the row
		row := []string{strconv.Itoa(i + 1)}
		rowSelection.Find("td").Each(func(j int, cellSelection *goquery.Selection) {
			cellText := strings.TrimSpace(cellSelection.Text())
			row = append(row, cellText)
		})
		// Append the row to the table
		table.Append(row)
	})

	// Render the table
	table.Render()
}