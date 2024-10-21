package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strconv"
	"time"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func PrintDAdates(regNo string, cookies types.Cookies){
	url:="https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment"
	semDetails := helpers.GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("No semesters found")
		return
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
	subjectlist:=allSubDetails(doc)
	url = "https://vtop.vit.ac.in/vtop/examinations/processDigitalAssignment"
	var table_data [][]string
    currentDate := time.Now()
	k:= 1
	for _, detail := range subjectlist {
		code := detail[0]
		name := detail[2]
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
		doc, err = goquery.NewDocumentFromReader(strings.NewReader(string(subBody)))
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
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
			table_data = append(table_data, []string{strconv.Itoa(k), name, title, datestr, strconv.Itoa(daysLeft)})
			k++
		}
	}
	//fmt.Println(table_data)
	if len(table_data) == 0 {
		fmt.Println("YAYYYY!! No DA's due")
		return
	}
	printTableData(table_data)
}

func printTableData(table_data [][]string) {
    // Determine the maximum width for each column
    maxWidths := []int{4, 4, 5, 4, 9} // Initial widths based on header lengths
    for _, row := range table_data {
        for i, col := range row {
            if len(col)+2 > maxWidths[i] {
                maxWidths[i] = len(col) + 2
            }
        }
    }

    // Create format strings based on the maximum widths
    format := fmt.Sprintf("| %%-%ds | %%-%ds | %%-%ds | %%-%ds | %%%ds |\n",  // Changed last one to right-aligned
        maxWidths[0], maxWidths[1], maxWidths[2], maxWidths[3], maxWidths[4])
    
    separator := fmt.Sprintf("+%%-%ds+%%-%ds+%%-%ds+%%-%ds+%%-%ds+\n",
        maxWidths[0]+2, maxWidths[1]+2, maxWidths[2]+2, maxWidths[3]+2, maxWidths[4]+2)
    
    var builder strings.Builder
    
    // Print the top border
    builder.WriteString(fmt.Sprintf(separator, 
        strings.Repeat("-", maxWidths[0]+2),
        strings.Repeat("-", maxWidths[1]+2),
        strings.Repeat("-", maxWidths[2]+2),
        strings.Repeat("-", maxWidths[3]+2),
        strings.Repeat("-", maxWidths[4]+2)))
    
    // Print the table header
    builder.WriteString(fmt.Sprintf(format, "S.No", "Name", "Title", "Date", "Days Left"))
    
    // Print separator line
    builder.WriteString(fmt.Sprintf(separator, 
        strings.Repeat("-", maxWidths[0]+2),
        strings.Repeat("-", maxWidths[1]+2),
        strings.Repeat("-", maxWidths[2]+2),
        strings.Repeat("-", maxWidths[3]+2),
        strings.Repeat("-", maxWidths[4]+2)))
    
    // Print the table rows
    for _, row := range table_data {
        daysLeft, _ := strconv.Atoi(row[4])
        color := "\033[32m" // Green by default
        if daysLeft < 3 {
            color = "\033[31m" // Red
        } else if daysLeft < 7 {
            color = "\033[33m" // Yellow
        }
        reset := "\033[0m"
        
        // Add padding to the days left value
        paddedDaysLeft := fmt.Sprintf("%*s", maxWidths[4], row[4])
        builder.WriteString(fmt.Sprintf(format, 
            row[0], row[1], row[2], row[3], 
            color+paddedDaysLeft+reset))
    }
    
    // Print the bottom border
    builder.WriteString(fmt.Sprintf(separator, 
        strings.Repeat("-", maxWidths[0]+2),
        strings.Repeat("-", maxWidths[1]+2),
        strings.Repeat("-", maxWidths[2]+2),
        strings.Repeat("-", maxWidths[3]+2),
        strings.Repeat("-", maxWidths[4]+2)))
    
    fmt.Println(builder.String())
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
        detail := []string{code, subject, name}

        // Append the row to the details slice
        details = append(details, detail)
    })
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