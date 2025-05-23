package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

const (
	HostelTableSelector = "div.table-responsive table.table"
	HostelRowsSelector  = "tbody tr"
	HostelCellSelector  = "td"
)

func PrintHostelInfo(regNo string, cookies types.Cookies, url string) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
	body, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println("Error fetching HTML:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil && debug.Debug {
		fmt.Println("Error parsing HTML:", err)
		return
	}

	fmt.Println("Student Accommodation Info")

	table := doc.Find(HostelTableSelector + " " + HostelRowsSelector)
	lastFiveRows := table.Slice(-5, table.Length())

	// Prepare nested list for PrintTable
	nestedList := [][]string{{"Field", "Information"}}
	lastFiveRows.Each(func(j int, rowSelection *goquery.Selection) {
		header := rowSelection.Find(HostelCellSelector).Eq(0).Text()
		value := rowSelection.Find(HostelCellSelector).Eq(1).Text()
		nestedList = append(nestedList, []string{
			strings.TrimSpace(header),
			strings.TrimSpace(value),
		})
	})

	// Use PrintTable to display the information
	helpers.PrintTable(nestedList, 0)
}
