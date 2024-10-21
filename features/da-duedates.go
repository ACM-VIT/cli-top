package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func PrintDAdates(regNo string, cookies types.Cookies){
	listOfSubjects := getAllSubs(regNo, cookies)
	var table_data [][]string
    table_data = append(table_data, []string{"Name", "Title", "Date", "Days Left"})
    currentDate := time.Now()
	for _, detail := range listOfSubjects {
		code := detail[2]
		name := detail[0]
        doc := getOneSub(regNo, cookies, code)
		lastest_work:=lastestda(doc)
		if len(lastest_work) > 0 {
			title :=lastest_work[0]
			datestr := lastest_work[1]
			date,err := time.Parse("02-Jan-2006", datestr)
			if err != nil {
				fmt.Println("Error parsing date:", err)
				continue	
			}
			daysLeft := int(date.Sub(currentDate).Hours() / 24) + 1
            daysString := strconv.Itoa(daysLeft)
            color := "\033[32m" 
            if daysLeft < 3 {
                color = "\033[31m" // Red
            } else if daysLeft < 7 {
                color = "\033[33m" // Yellow
            }
            reset := "\033[0m"
            daysString = color + daysString + reset
			table_data = append(table_data, []string{name, title, datestr, daysString})
		}
	}
	//fmt.Println(table_data)
    if len(table_data) == 0 {
        fmt.Println("YAYYYY!! No DA's due")
        return
    }
	helpers.PrintTable(table_data)
    fmt.Println()
}

func allSubDetails(doc *goquery.Document) [][]string {
    var details [][]string

    // Use CSS selectors to find and extract data
    doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
        // Extract data from each column
        td := s.Find("td")
        code := td.Eq(1).Text()
        subject := td.Eq(2).Text()
        name := td.Eq(3).Text()
        // Add more lines as needed for other columns

        // Create a slice for the current row
        detail := []string{name, subject, code}

        // Append the row to the details slice
        details = append(details, detail)
    })
    fmt.Println()
    return details
}

func lastestda(doc *goquery.Document) []string {
	var details []string
	//found := false

	doc.Find("tr.fixedContent.tableContent").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// Extract data from each column
		td := s.Find("td")
		name := td.Eq(1).Text()
		span := td.Eq(4).Find("span")
		date := span.Text()
		// Add more lines as needed for other columns
		
		style, exists := span.Attr("style")
        if exists && strings.Contains(style, "color: green;") {
            details = append(details, name)
			details = append(details, date)
			//found = true
            return false // Break out of the loop
        }
        return true // Continue the loop
    })

    // if !found {
    //     fmt.Println("Did not find the element")
	// }
	return details
}

func getAllSubs(regNo string, cookies types.Cookies) [][]string {
    url:="https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment"
	semDetails := helpers.GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("No semesters found")
		return nil 
	}
	semID := semDetails.SemIds[len(semDetails.SemIds)-1]
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

    return allSubDetails(doc)
}

func getOneSub(regNo string, cookies types.Cookies, code string) *goquery.Document {
    url:="https://vtop.vit.ac.in/vtop/examinations/processDigitalAssignment"
    payloadMap := map[string]string{
        "_csrf":         cookies.CSRF,
        "paramReturnId": "getCourseForCoursePage",
        "classId":      code,
        "authorizedID":  regNo,
        "x":             fmt.Sprintf("%d", time.Now().Unix()),
    }
    formData := helpers.FormatBodyData(payloadMap)
    subBody, err := helpers.FetchReq(regNo, cookies, url, "",formData, "POST", "")
    if err != nil && debug.Debug {
        fmt.Println(err)
    }
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(subBody)))
    if err != nil && debug.Debug {
        fmt.Println(err)
    }
    return doc
}