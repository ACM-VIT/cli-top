package features

import (
    "bytes"
    "fmt"
    "strings"
    "cli-top/debug"
    "cli-top/helpers"
    "cli-top/types"
    "github.com/PuerkitoBio/goquery"
)

func PrintHostelInfo(regNo string, cookies types.Cookies, url string) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
        fmt.Println("Please login first using the cli-top login command")
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

    table := doc.Find("div.table-responsive table.table tbody tr")
    lastFiveRows := table.Slice(-5, table.Length())

    // Prepare nested list for PrintTable
    nestedList := [][]string{{"Field", "Information"}}
    lastFiveRows.Each(func(j int, rowSelection *goquery.Selection) {
        header := rowSelection.Find("td").Eq(0).Text()
        value := rowSelection.Find("td").Eq(1).Text()
        nestedList = append(nestedList, []string{
            strings.TrimSpace(header),
            strings.TrimSpace(value),
        })
    })

    // Use PrintTable to display the information
    helpers.PrintTable(nestedList, 0)
}