package helpers

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"
	"vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func PrintSemDetails(regNo string, cookies types.Cookies) {
	semDetails := GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available.")
		return
	}

	// Generate Markdown table text
	markdownTable := GenerateSemDetailsMarkdownTable(semDetails)

	// Render Markdown using glamour
	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		fmt.Println("Error rendering Markdown:", err)
		return
	}

	// Print the rendered Markdown
	fmt.Println(rendered)
}

func GetSemDetails(cookies types.Cookies, regNo string) types.SemesterDetails {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())

	bodyText, err := FetchReq(regNo, cookies, url, payload, "POST")
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}

	// Create a slice to store the extracted data
	var SemNames []string
	var SemIds []string
	var tempIds []string

	// Find and save the semester IDs
	FindAndSaveSemIds(doc, "form-select", &tempIds)
	SemIds = RemoveEmptyStrings(tempIds)

	for _, semId := range SemIds {
		optionText := FindOptionWithTagValue(doc, semId)

		if optionText != "" {
			SemNames = append(SemNames, optionText)

		} else {
			// Handle the case where no <option> tag is found with the specified SemId
			SemNames = append(SemNames, "Unknown")
		}
	}

	// Reverse SemIds and SemNames
	reverseSemIds := make([]string, len(SemIds))
	reverseSemNames := make([]string, len(SemNames))
	for i := 0; i < len(SemIds); i++ {
		reverseIndex := len(SemIds) - 1 - i
		reverseSemIds[i] = SemIds[reverseIndex]
		reverseSemNames[i] = SemNames[reverseIndex]
	}

	// Save the reversed values back to SemDetails
	SemIds = reverseSemIds
	SemNames = reverseSemNames

	// Return the encapsulated struct
	return types.SemesterDetails{
		SemNames: SemNames,
		SemIds:   SemIds,
	}
}

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) string {
	selectedSemId := ""
	selectedSemName := ""

	var choice int
	fmt.Print("\nEnter the index of the semester to view courses: ")
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

	formattedSelection := fmt.Sprintf("\n# You selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)

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

type CourseDetail struct {
	Index         int
	CourseCode    string
	CourseDetails string
	ClassID       string
}

func GetCourseDetails(cookies types.Cookies, regNo string, semID string) []CourseDetail {
	var courseDetails []CourseDetail

	url := "https://vtop.vit.ac.in/vtop/getCourseForCoursePage"
	payload := fmt.Sprintf("_csrf=%s&paramReturnId=getCourseForCoursePage&semSubId=%s&authorizedID=%s&x=%d", cookies.CSRF, semID, regNo, time.Now().UnixNano())
	resp, err := FetchReq(regNo, cookies, url, payload, "POST")
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp)))
	if err != nil {
		log.Fatal(err)
	}

	var builder strings.Builder
	builder.WriteString("| Index | Course Code     | Course Details                                                      |\n")
	builder.WriteString("|-------|-----------------|---------------------------------------------------------------------|\n")

	index := 1
	doc.Find("#getCourseForCoursePage select#courseCode option").Each(func(i int, s *goquery.Selection) {
		value := s.AttrOr("value", "")
		text := s.Text()
		if value != "" && text != "--Choose Course --" {
			courseCode := strings.Split(text, " - ")[0]
			courseDetailsText := strings.TrimSpace(strings.TrimPrefix(text, courseCode+" - "))

			builder.WriteString(fmt.Sprintf("| %-5d | %-15s | %-25s |\n", index, courseCode, courseDetailsText))
			courseDetail := CourseDetail{
				Index:         index,
				CourseCode:    courseCode,
				CourseDetails: courseDetailsText,
				ClassID:       value,
			}
			courseDetails = append(courseDetails, courseDetail)
			index++
		}
	})

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
	)

	output, _ := r.Render(builder.String())
	fmt.Println(output)
	return courseDetails
}

func SelectCourse(regNo string, cookies types.Cookies, semID string) string {
	CourseDetails := GetCourseDetails(cookies, regNo, semID)
	selectedClassID := ""

	var choice int
	fmt.Print("\nEnter the index of the course to view faculties: ")
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(CourseDetails) {
		fmt.Println("Invalid choice.")
	} else {
		chosenCourse := CourseDetails[choice-1]
		selectedClassID = chosenCourse.ClassID
		formattedSelection := fmt.Sprintf("\nYou selected ClassID: %s, CourseName: %s\n", selectedClassID, chosenCourse.CourseDetails)

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
	}

	return selectedClassID
}

func GetFacultyDetails(regNo string, cookies types.Cookies, classID string) string {
	url := "https://vtop.vit.ac.in/vtop/getSlotIdForCoursePage"
	payload := fmt.Sprintf("_csrf=%s&classId=%s&praType=source&paramReturnId=getSlotIdForCoursePage&semSubId=%s&authorizedID=%s&x=%d", cookies.CSRF, classID, regNo, time.Now().UnixNano())
	resp, err := FetchReq(regNo, cookies, url, payload, "POST")
	if err != nil {
		log.Fatal(err)
	}

	// doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp)))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	fmt.Print(string(resp))
	return ""

}

func GenerateSemDetailsMarkdownTable(semDetails types.SemesterDetails) string {
	var buf bytes.Buffer

	// Table header
	buf.WriteString("| Index | SemId          | SemName                   |\n")
	buf.WriteString("|-------|----------------|---------------------------|\n")

	// Iterate through SemIds and SemNames using a for loop
	for i := 0; i < len(semDetails.SemIds); i++ {
		index := fmt.Sprintf("%d", i+1)
		semId := semDetails.SemIds[i]
		semName := semDetails.SemNames[i]

		// Table row
		buf.WriteString(fmt.Sprintf("| %-5s | %-14s | %-25s |\n", index, semId, semName))
	}

	return buf.String()
}

func FindAndSaveSemIds(doc *goquery.Document, targetClass string, result *[]string) {
	doc.Find("select." + targetClass + " option").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists {
			*result = append(*result, value)
		}
	})
}

func RemoveEmptyStrings(data []string) []string {
	var cleanedData []string
	for _, item := range data {
		if item != "" {
			cleanedData = append(cleanedData, item)
		}
	}
	return cleanedData
}

func FindOptionWithTagValue(doc *goquery.Document, targetValue string) string {
	return doc.Find("option[value='" + targetValue + "']").Text()
}
