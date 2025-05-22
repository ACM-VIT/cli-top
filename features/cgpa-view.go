package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	CGPATableSelector      = "div.table-responsive table.table"
	CGPARowsSelector       = "tbody tr"
	CGPACellSelector       = "td"
	CreditsRegisteredIndex = 0
	CreditsEarnedIndex     = 1
	CGPAIndex              = 2
	SGradesIndex           = 3
	AGradesIndex           = 4
	BGradesIndex           = 5
	CGradesIndex           = 6
	DGradesIndex           = 7
	EGradesIndex           = 8
	FGradesIndex           = 9
	NGradesIndex           = 10
)

func PrintCgpa(regNo string, cookies types.Cookies, url string) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

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

	// Extract and print data from the specified HTML structure
	table := doc.Find(CGPATableSelector + " " + CGPARowsSelector)
	row := table.First()

	// Extract data from each column
	creditsRegistered := row.Find(CGPACellSelector).Eq(CreditsRegisteredIndex).Text()
	creditsEarned := row.Find(CGPACellSelector).Eq(CreditsEarnedIndex).Text()
	cgpa := row.Find(CGPACellSelector).Eq(CGPAIndex).Text()
	sGrades := row.Find(CGPACellSelector).Eq(SGradesIndex).Text()
	aGrades := row.Find(CGPACellSelector).Eq(AGradesIndex).Text()
	bGrades := row.Find(CGPACellSelector).Eq(BGradesIndex).Text()
	cGrades := row.Find(CGPACellSelector).Eq(CGradesIndex).Text()
	dGrades := row.Find(CGPACellSelector).Eq(DGradesIndex).Text()
	eGrades := row.Find(CGPACellSelector).Eq(EGradesIndex).Text()
	fGrades := row.Find(CGPACellSelector).Eq(FGradesIndex).Text()
	nGrades := row.Find(CGPACellSelector).Eq(NGradesIndex).Text()

	// Create a nested list for grades
	gradesTableData := [][]string{
		{"Grade", "Count"},
		{"S Grades", sGrades},
		{"A Grades", aGrades},
		{"B Grades", bGrades},
		{"C Grades", cGrades},
		{"D Grades", dGrades},
		{"E Grades", eGrades},
		{"F Grades", fGrades},
		{"N Grades", nGrades},
	}

	// Print the grades table
	fmt.Println()
	fmt.Printf("\nCredits Registered: %s\n", creditsRegistered)
	fmt.Printf("Credits Earned: %s\n", creditsEarned)
	fmt.Printf("CGPA: \033[32m%s\033[0m\n", cgpa) // Highlight CGPA in green
	fmt.Println()
	helpers.PrintTable(gradesTableData, 0)
	fmt.Println()

	// Print the credits and CGPA information in line format
}
