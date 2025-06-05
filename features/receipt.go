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

const (
	ReceiptTableSelector  = "table.table-bordered"
	ReceiptRowsSelector   = "tbody tr"
	ReceiptCellSelector   = "td"
	ReceiptHeaderSelector = "th"
)

func GetReceipt(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
	url := "https://vtop.vit.ac.in/vtop/finance/getStudentReceipts"

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

	// Initialize the nested list for receipts
	var receipts [][]string
	// Add the header row
	receipts = append(receipts, []string{"INVOICE NUMBER", "RECEIPT NUMBER", "DATE", "AMOUNT"})

	// Iterate through the table rows and extract data
	doc.Find(ReceiptTableSelector + " " + ReceiptRowsSelector).Each(func(i int, rowSelection *goquery.Selection) {
		if i == 0 && strings.TrimSpace(rowSelection.Find(ReceiptHeaderSelector).First().Text()) != "SERIAL" {
			return
		}

		// Extract data from each cell in the row
		var row []string
		rowSelection.Find(ReceiptCellSelector).Each(func(j int, cellSelection *goquery.Selection) {
			if j < 4 { // Exclude the "VIEW" column
				cellText := strings.TrimSpace(cellSelection.Text())
				row = append(row, cellText)
			}
		})
		// Append the row to the receipts list
		receipts = append(receipts, row)
	})

	// Print the table using the helpers.PrintTable function
	helpers.PrintTable(receipts, 1)
}
