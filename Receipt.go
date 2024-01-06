// 060d1713-1b9a-4034-b253-7d5b48697c82
package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func main() {
	client := &http.Client{}
	var data = strings.NewReader(`verifyMenu=true&authorizedID=22BDS0188&_csrf=060d1713-1b9a-4034-b253-7d5b48697c82&nocache=@(new Date().getTime())`)
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/finance/getStudentReceipts", data)
	if err != nil {
		log.Fatal(err)
	}
	setHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Parse and print the HTML table with the specified class
	parseAndPrintTable(resp.Body)
}

func setHeaders(req *http.Request) {
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("cookie", "JSESSIONID=26E94E7A312F0F311BA6D4565B8EC7F0; _ga=GA1.1.1763604292.1699285434; _ga_VY3TWN1FJ7=GS1.1.1702848300.2.0.1702848300.60.0.0; _hp2_id.3863792970=%7B%22userId%22%3A%224954569497889160%22%2C%22pageviewId%22%3A%226755249576479608%22%2C%22sessionId%22%3A%224822149987821438%22%2C%22identity%22%3Anull%2C%22trackerVersion%22%3A%224.0%22%7D; SERVERID=s1")
	req.Header.Set("origin", "https://vtop.vit.ac.in")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content?")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
}

func parseAndPrintTable(body io.Reader) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"RECEIPT NUMBER", "DATE", "AMOUNT", "CAMPUS CODE"})

	// Skip first row flag
	skipFirstRow := true

	// Find the table with the target class
	doc.Find("table.table-bordered tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
		// Skip the first row
		if skipFirstRow {
			skipFirstRow = false
			return
		}

		// Extract data from each cell in the row
		row := []string{}
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
