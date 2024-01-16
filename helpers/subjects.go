package helpers

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

func SubjectDetails(doc *goquery.Document) []string {
	var details []string

	// Use CSS selectors to find and extract data
	doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
		// Skip every other iteration
		if i%2 != 0 {
			return
		}

		// Extract data from each column
		td := s.Find("td")
		code := td.Eq(1).Text()
		subject := td.Eq(2).Text()
		name := td.Eq(3).Text()
		ctype := td.Eq(4).Text()
		fac := td.Eq(6).Text()
		slot := td.Eq(7).Text()
		// Add more lines as needed for other columns

		// Print or use the extracted data
		detail := fmt.Sprintf("## CourseCode: %s, CourseTitle: %s,  CourseType: %s, Faculty: %s, Slot: %s, ClassNbr: %s\n", subject, name, ctype, fac, slot, code)
		// Print or use other extracted data as needed

		details = append(details, detail)
	})
	return details
}
