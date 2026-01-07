package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	LibraryDuesTableSelector = "table.table-bordered"
	LibraryDuesRowsSelector  = "tbody tr"
	LibraryDuesCellSelector  = "td"
)

func GetLibraryDues(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
	url := "https://vtop.vit.ac.in/vtop/finance/libraryPayments"

	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil && debug.Debug {
		helpers.Println("Error fetching data:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil && debug.Debug {
		helpers.Println("Error parsing HTML:", err)
		return
	}

	// Create table
	var AllDuesTable [][]string
	AllDuesTable = append(AllDuesTable, []string{"TYPE", "AMOUNT"})

	doc.Find(LibraryDuesTableSelector + " " + LibraryDuesRowsSelector).Each(func(i int, rowSelection *goquery.Selection) {
		row := []string{}
		rowSelection.Find(LibraryDuesCellSelector).Each(func(j int, cellSelection *goquery.Selection) {
			cellText := strings.TrimSpace(cellSelection.Text())
			row = append(row, cellText)
		})
		// Append the row to the table
		AllDuesTable = append(AllDuesTable, row)
	})

	// Render the table
	helpers.Println()
	helpers.PrintTable(AllDuesTable, 1)
	helpers.Println()
}
