package features

import (
	"bytes"
	"fmt"
	"strings"
	"cli-top/debug"
	"cli-top/types"

	"cli-top/helpers"

	"github.com/PuerkitoBio/goquery"
)

func PrintHostelInfo(regNo string, cookies types.Cookies, url string) {
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

	// fmt.Println("+-----------------------------+------------------------------------------------------+")
	fmt.Println("Student Accommodation Info")
	fmt.Println("+-----------------------------+--------------------------------------------------------------------+")

	table := doc.Find("div.table-responsive table.table tbody tr")
	lastFiveRows := table.Slice(-5, table.Length())

	lastFiveRows.Each(func(j int, rowSelection *goquery.Selection) {
		header := rowSelection.Find("td").Eq(0).Text()
		value := rowSelection.Find("td").Eq(1).Text()

		fmt.Printf("| %-27s | %-66s |\n", strings.TrimSpace(header), strings.TrimSpace(value))
	})

	fmt.Println("+-----------------------------+--------------------------------------------------------------------+")
}
