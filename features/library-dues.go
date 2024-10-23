package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"
	"github.com/PuerkitoBio/goquery"
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
	var AllDuesTable [][]string
	AllDuesTable = append(AllDuesTable, []string{ "TYPE", "AMOUNT"})

	doc.Find("table.table-bordered tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		row := []string{}
		rowSelection.Find("td").Each(func(j int, cellSelection *goquery.Selection) {
			cellText := strings.TrimSpace(cellSelection.Text())
			row = append(row, cellText)
		})
		// Append the row to the table
		AllDuesTable = append(AllDuesTable, row)
	})

	// Render the table
	fmt.Println()
	helpers.PrintTable(AllDuesTable, 1)
	fmt.Println()
}