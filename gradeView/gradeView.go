package gradeView

import (
	"VTOP-CLI/types"
	"bytes"
	"fmt"
	"io"
	"log"
	//"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	//"golang.org/x/net/html"
)

type SemesterDetails struct {
	SemNames []string
	SemIds   []string
}

func fetchReq(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
	//fmt.Println(payload)
	// Create a new request with POST method and payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	// Set headers or cookies if needed
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func fetchReq2(regNo string, cookies types.Cookies, url string, semID string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request

	payload := fmt.Sprintf("authorizedID=%s&_csrf=%s&semesterSubId=%s&x=%s", regNo, cookies.CSRF, semID, time.Now().UTC().Format(time.RFC1123)) //fmt.Println(payload)
	//fmt.Println(payload)
	// Create a new request with POST method and payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	// Set headers or cookies if needed
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

//payload := []byte("_csrf=154a792d-e0d1-42fb-8300-c4211db46510&semesterSubId=VL20232405&authorizedID=22BCI0272&x=" + time.Now().UTC().Format(time.RFC1123))

func GetSemDetails(cookies types.Cookies, regNo string) SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"


	//fmt.Println(regNo, cookies)
	bodyText, err := fetchReq(regNo, cookies, url, "")
	if err != nil {
		log.Fatal(err)
	}
	


	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(string(bodyText))

	// Create a slice to store the extracted data
	var SemNames []string
	var SemIds []string

	var tempIds []string
	// Find and save the semester IDs
	findAndSaveSemIds(doc, "form-select", &tempIds)
	//fmt.Printf("%q", tempIds)
	SemIds = removeEmptyStrings(tempIds)
	//fmt.Printf("%q", SemIds)

	// fmt.Printf("%q\n", SemIds)

	for _, semId := range SemIds {
		optionText := findOptionWithTagValue(doc, semId)

		if optionText != "" {
			SemNames = append(SemNames, optionText)

		} else {
			// Handle the case where no <option> tag is found with the specified SemId
			SemNames = append(SemNames, "Unknown")
		}
	}

	finalSemIds := make([]string, len(SemIds))
	finalSemNames := make([]string, len(SemNames))
	for i := 0; i < len(SemIds); i++ {
		finalSemIds[i] = SemIds[i]
		finalSemNames[i] = SemNames[i]
	}

	SemIds = finalSemIds
	SemNames = finalSemNames

	return SemesterDetails{
		SemNames: SemNames,
		SemIds:   SemIds,
	}

}

func PrintSemDetails(regNo string, cookies types.Cookies) {
	semDetails := GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available.")
		return
	}
	
	// generateSemDetailsMarkdownTable(semDetails)

	markdownTable := generateSemDetailsMarkdownTable(semDetails)

	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		fmt.Println("Error rendering Markdown:", err)
		return
	}
	fmt.Println(rendered)
	
}

func findAndSaveSemIds(doc *goquery.Document, targetClass string, result *[]string) {
	

	// fmt.Println("check1")
	doc.Find("select." + targetClass + " option").Each(func(i int, s *goquery.Selection) {
		// fmt.Println("check")

		value, exists := s.Attr("value")
		//fmt.Println(value)
		if exists {
			*result = append(*result, value)
			// fmt.Println("yes")
		}
	})
}

func generateSemDetailsMarkdownTable(semDetails SemesterDetails)string {
	
	// var c int
	// var r int
	// tot := int(math.Ceil(float64(len(semDetails.SemIds)) / float64(10)))

	// for {
		
	// 	fmt.Printf("\nEnter page no 1-%d to view sem details : ", tot)
	// 	fmt.Scanln(&c)

	// 	if c > tot || c < 1 {
	// 		fmt.Println("Invalid choice.")
	// 		break
	// 	}

	// 	pageLogic(c,tot, semDetails)
	// 	r = c

	// 	for {
	// 		fmt.Print("\nEnter 'n' for next page, 'p' for previous page, or 'g' for Grade-View : ")
	// 		var input string
	// 		fmt.Scanln(&input)

	// 		switch input {
	// 		case "n":
	// 			if r < tot {
	// 				r++
	// 				pageLogic(r,tot, semDetails)
	// 			} else {
	// 				fmt.Println("Invalid choice. Please enter again.")
	// 			}
	// 		case "p":
	// 			if r > 1 {
	// 				r--
	// 				pageLogic(r, tot,semDetails)
	// 			} else {
	// 				fmt.Println("Invalid choice. Please enter again.")
	// 			}
	// 		case "g":
	// 			fmt.Println("Grade-View selected.")
	// 			return
	// 		default:
	// 			fmt.Println("Invalid input. Please try again.")
	// 		}
	// 	}
	// }
	var buf bytes.Buffer

	buf.WriteString("| Index | SemId          | SemName                   |\n")
	buf.WriteString("|-------|----------------|---------------------------|\n")

	for i := 0; i < len(semDetails.SemIds); i++ {
		index := fmt.Sprintf("%d", i+1)
		semId := semDetails.SemIds[i]
		semName := semDetails.SemNames[i]

		buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-25s |\n", index, semId, semName))
	}

	return buf.String()
}


// func pageLogic(c int,tot int, semDetails SemesterDetails){
// 	var buf bytes.Buffer
	
// 	fmt.Printf("\n--- PAGE %d/%d ---\n\n", c, tot)
	
// 	if 10*(c-1)+9 < len(semDetails.SemIds){
			 
// 		buf.WriteString("| Index | SemId          | SemName                             |\n")
// 		buf.WriteString("|-------|----------------|-------------------------------------|\n")
// 		for i := 10*(c-1); i < 10*(c-1)+10; i++ {
// 			index := fmt.Sprintf("%d", i+1)
// 			semId := semDetails.SemIds[i]
// 			semName := semDetails.SemNames[i]
	
// 			// Table row
// 			buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-35s |\n", index, semId, semName))
// 		}
		
// 	}else{
// 		buf.WriteString("| Index | SemId          | SemName                             |\n")
// 		buf.WriteString("|-------|----------------|-------------------------------------|\n")
// 		for i := 10*(c-1); i < len(semDetails.SemIds); i++ {
// 			 index := fmt.Sprintf("%d", i+1)
// 			 semId := semDetails.SemIds[i]
// 			 semName := semDetails.SemNames[i]
	
// 			 // Table row
			
// 			 buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-35s |\n", index, semId, semName))
// 		}

// 	}
// 	fmt.Println(buf.String())
// 	fmt.Printf("--- END OF PAGE %d/%d ---",c,tot)
// 	fmt.Println(" ")
// }


func GetGrade(regNo string, cookies types.Cookies, semId string) {

	url := "https://vtop.vit.ac.in/vtop/examinations/examGradeView/doStudentGradeView"

	sel_id := Marks(regNo, cookies, 0)
	//fmt.Println(sel_id)
	bodyText, err := fetchReq2(regNo, cookies, url, sel_id)
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	
	findAndSaveGrade(doc)
	
}

func Marks(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	fmt.Print("\nEnter the index of the semester to view grade: ")
	fmt.Scanln(&choice)
	semDet := GetSemDetails(cookies, regNo)

	if choice < 1 || choice > len(semDet.SemIds) {
		fmt.Println("Invalid choice.")
	} else {
		for i, id := range semDet.SemIds {
			if i+1 == choice {
				selectedSemId = id
				selectedSemName = semDet.SemNames[i]
			}
		}
	}

	// Format the string with glamour
	formattedSelection := fmt.Sprintf("\n# You selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)

	// Render and print the formatted string
	renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
	if err != nil {
		log.Fatal("Error creating glamour renderer:", err)
	}

	output, err := renderer.Render(formattedSelection)
	if err != nil {
		log.Fatal("Error rendering formatted string:", err)
	}

	fmt.Print(output)

	fmt.Println()

	return selectedSemId

}

func findAndSaveGrade(doc *goquery.Document) {
    var markdownTable strings.Builder
	var  count int
    targetClass := "table.table-hover.table-bordered"
    table := doc.Find(targetClass)

    if table.Length() > 0 {
        printTable("Header", nil, &markdownTable)

        // Find rows within tbody
        table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
			style, exists := rowSelection.Attr("style")
			if exists && style == "background-color: #C0D8C0;" {
				count = i
				
			}
			//fmt.Println(count)
            row := []string{} // Initialize a new slice for each row

            // Find cells (td) within each row
            rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {            //tr=11, td=11 , th=9+4 
                text := strings.TrimSpace(cell.Text())
                row = append(row, text)
            })

            if len(row) > 2 {
                printFormattedRow(row, &markdownTable,count)
            }
        })
    } else {
        fmt.Println("Data not found")
    }

    fmt.Println(markdownTable.String())
	doc.Find("span[style='font-size: 18px; font-weight: bold;']").Each(func(i int, s *goquery.Selection) {
		gpa := s.Text()
		fmt.Println("\x1b[32;1mGreen\x1b[0m : Course not included in GPA/CGPA\n")
		fmt.Println(gpa)
	})
}


func removeEmptyStrings(data []string) []string {
	var cleanedData []string
	for _, item := range data {
		if item != "" {
			cleanedData = append(cleanedData, item)
		}
	}
	return cleanedData
}

func findOptionWithTagValue(doc *goquery.Document, targetValue string) string {
	return doc.Find("option[value='" + targetValue + "']").Text()
}

func strToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		fmt.Println("Error converting string to integer:", err)

	}

	return num
}

func printTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-10s | %-50s | %-25s | %-21s | %-5s | %-6s |\n",
 		"S.No.", "Course Code", "Course Title", "Course Type", "Credits"," Total","Grade"))
 	builder.WriteString("|       |             |                                                    |                           |-----------------------|        |        |\n")
 	//builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")

 	builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-7s| %-6s |\n",
 		"", "", "", "", "L", "P","J","C", "",""))
 	builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")
}

func printFormattedRow(row []string, builder *strings.Builder,count int) {
	if strToInt(row[0]) == count-1 {
		builder.WriteString(fmt.Sprintf("| \x1b[32m%-5s\x1b[0m | \x1b[32m%-11s\x1b[0m | \x1b[32m%-50s\x1b[0m | \x1b[32m%-25s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-6s\x1b[0m | \x1b[32m%-6s\x1b[0m |\n",
	  	row[0], row[1], row[2], row[3], row[4],row[5], row[6], row[7],row[9],row[10]))
	}else{
		builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-6s | %-6s |\n",
	 	row[0], row[1], row[2], row[3], row[4],row[5], row[6], row[7],row[9],row[10]))
	}
}



