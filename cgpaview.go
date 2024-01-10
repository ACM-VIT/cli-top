package main

import (
	"crypto/tls"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	cgpaData, err := getCGPA("22BCT0355")
	if err != nil {
		log.Fatal(err)
	}

	// Parse and print the relevant HTML table data
	writeSpecificTableUsingTableWriter(cgpaData)
}

func getCGPA(authorizedID string) (string, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf("_csrf=df6d37ef-6e12-4d85-bc3a-4132953f0e1e&authorizedID=%s&history=&form=undefined&control=history&x=Wed, 20 Dec 2023 14:12:17 GMT", authorizedID))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory", data)
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("cookie", "JSESSIONID=2F755A5CE11E8EAA8DF054862294A0F1	; SERVERID=s1")
	req.Header.Set("origin", "https://vtop.vit.ac.in")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory")
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

func writeSpecificTableUsingTableWriter(htmlContent string) {
	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	// Find the div with class "box-body table-responsive"
	divNode := findDivWithClass(doc, "box-body table-responsive")
	if divNode != nil {
		// Find the table within the div
		tableNode := findTableInDiv(divNode)
		if tableNode != nil {
			// Extract and render the specific table using tablewriter
			extractAndRenderSpecificTable(table, tableNode)
		} else {
			fmt.Println("Table not found under div class 'box-body table-responsive'")
		}
	} else {
		fmt.Println("Div with class 'box-body table-responsive' not found")
	}
}

func extractAndRenderSpecificTable(table *tablewriter.Table, tableNode *html.Node) {
	// Extract header and rows for the specific table
	header, rows := extractTableDataForSpecificTable(tableNode)

	// Set header and add rows to the table
	table.SetHeader(header)
	table.AppendBulk(rows)

	// Render the table
	table.Render()
}

func extractTableDataForSpecificTable(tableNode *html.Node) ([]string, [][]string) {
	var header []string
	var rows [][]string

	// Extract header and rows
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "th" {
				header = append(header, extractCellData(n))
			} else if n.Data == "tr" {
				row := extractRowDataUsingTableWriter(n)
				rows = append(rows, row)
			}
		}
		// Recursively extract data from children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(tableNode)

	return header, rows
}

func findDivWithClass(node *html.Node, className string) *html.Node {
	var targetNode *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, className) {
					targetNode = n
					return
				}
			}
		}
		// Recursively search for the div in children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}

	find(node)
	return targetNode
}

func findTableInDiv(divNode *html.Node) *html.Node {
	var targetNode *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			targetNode = n
			return
		}
		// Recursively search for the table in children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}

	find(divNode)
	return targetNode
}

func extractRowDataUsingTableWriter(rowNode *html.Node) []string {
	var rowData []string
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			cellContent := extractCellData(n)
			rowData = append(rowData, cellContent)
		}
		// Recursively extract cells from children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(rowNode)
	return rowData
}

func extractCellData(cellNode *html.Node) string {
	var cellContent string
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			cellContent += n.Data
		}
		// Recursively extract text content from children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(cellNode)
	return strings.TrimSpace(cellContent)
}
