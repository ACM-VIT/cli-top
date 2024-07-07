package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func PrintCgpa(regNo string, cookies types.Cookies, url string) {

	// Fetch the CGPA data
	body, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println("Error fetching CGPA data:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil && debug.Debug {
		fmt.Println("Error parsing HTML:", err)
		return
	}

	fmt.Println("CGPA Details")

	// Extract and print data from the specified HTML structure
	table := doc.Find("div.table-responsive table.table tbody tr")
	row := table.First()

	// Extract data from each column
	creditsRegistered := row.Find("td").Eq(0).Text()
	creditsEarned := row.Find("td").Eq(1).Text()
	cgpa := row.Find("td").Eq(2).Text()
	sGrades := row.Find("td").Eq(3).Text()
	aGrades := row.Find("td").Eq(4).Text()
	bGrades := row.Find("td").Eq(5).Text()
	cGrades := row.Find("td").Eq(6).Text()
	dGrades := row.Find("td").Eq(7).Text()
	eGrades := row.Find("td").Eq(8).Text()
	fGrades := row.Find("td").Eq(9).Text()
	nGrades := row.Find("td").Eq(10).Text()

	// Create a table
	tableData := [][]string{
		{"Credits Registered", creditsRegistered},
		{"Credits Earned", creditsEarned},
		{"CGPA", fmt.Sprintf("\033[32m%s\033[0m", cgpa)}, // Highlight CGPA in green
		{"S Grades", sGrades},
		{"A Grades", aGrades},
		{"B Grades", bGrades},
		{"C Grades", cGrades},
		{"D Grades", dGrades},
		{"E Grades", eGrades},
		{"F Grades", fGrades},
		{"N Grades", nGrades},
	}

	tableObj := tablewriter.NewWriter(log.Writer())

	// Append data to the table
	for _, data := range tableData {
		tableObj.Append(data)
	}

	// Render the table
	tableObj.Render()
}
