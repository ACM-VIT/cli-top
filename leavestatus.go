package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func main() {
	leaveStatus, err := getLeaveStatus("22BCT0355")
	if err != nil {
		log.Fatal(err)
	}

	// Parse and print human-readable leave status
	parseAndPrintHumanReadable(leaveStatus)
}

func getLeaveStatus(authorizedID string) (string, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf("_csrf=37249a38-9cde-4e10-a3e8-b0f899e369ce&authorizedID=%s&history=&form=undefined&control=history&x=Wed, 20 Dec 2023 14:12:17 GMT", authorizedID))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/hostels/student/leave/6", data)
	if err != nil {
		return "", err
	}

	// Set headers (unchanged)
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("cookie", "JSESSIONID=D233CAAD6D85248E4D1C09BA04295F13; SERVERID=s2")
	req.Header.Set("origin", "https://vtop.vit.ac.in")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content?")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.71 Safari/537.36")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyText), nil
}

func parseAndPrintHumanReadable(htmlContent string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	// Create a new table and set its properties
	table := tablewriter.NewWriter(log.Writer())
	table.SetHeader([]string{"Leave ID", "Visit Place", "Reason", "Leave Type", "From", "To", "Status", "Remarks"})

	// Find and print data from the HTML
	doc.Find("tr").Each(func(i int, trSelection *goquery.Selection) {
		var rowData []string
		// Skip the first column (Leave ID)
		trSelection.Find("td:not(:first-child)").Each(func(j int, tdSelection *goquery.Selection) {
			// Add text content of each table cell to the row data
			rowData = append(rowData, tdSelection.Text())
		})
		// Add row data to the table
		table.Append(rowData)
	})

	// Render the table
	table.Render()
}




