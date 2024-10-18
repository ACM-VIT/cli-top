package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

type Semester struct {
	SemName string
	SemID   string
}

func ExecuteCoursePageDownload(regNo string, cookies types.Cookies) {
	semDetails := helpers.GetSemDetails(cookies, regNo)
	if len(semDetails.SemIds) == 0 {
		fmt.Println("Error fetching semester details or no semesters available.")
		return
	}

	selectedSemId := helpers.SelectSemester(regNo, cookies, 0)
	if selectedSemId == "" {
		fmt.Println("Error selecting semester.")
		return
	}

	selectedSemName := "Unknown"
	for i, id := range semDetails.SemIds {
		if id == selectedSemId {
			selectedSemName = semDetails.SemNames[i]
			break
		}
	}

	selectedSemester := Semester{
		SemName: selectedSemName,
		SemID:   selectedSemId,
	}

	selectedCourse, err := fetchAndSelectCourse(regNo, cookies, selectedSemester.SemID)
	if err != nil {
		fmt.Println("Error selecting course:", err)
		return
	}

	selectedCourseName := selectedCourse.Name
	formattedSelection := fmt.Sprintf("\n# You selected Course: %s\n", selectedCourseName)
	renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
	if err != nil && debug.Debug {
		fmt.Println("Error creating glamour renderer:", err)
	}
	output, err := renderer.Render(formattedSelection)
	if err != nil && debug.Debug {
		fmt.Println("Error rendering formatted string:", err)
	}
	fmt.Print(output)

	slotIds, err := fetchSlotIds(regNo, cookies, selectedSemester.SemID, selectedCourse.ID)
	if err != nil {
		fmt.Println("Error fetching slots:", err)
		return
	}

	if len(slotIds) == 0 {
		fmt.Println("No slots available for the selected course.")
		return
	}

	faculties, err := fetchFacultiesForAllSlotsConcurrently(regNo, cookies, selectedSemester.SemID, selectedCourse.ID, slotIds)
	if err != nil {
		fmt.Println("Error fetching faculties:", err)
		return
	}

	if len(faculties) == 0 {
		fmt.Println("No faculties found for the selected course across all slots.")
		return
	}

	selectedFaculty, err := helpers.SelectFaculty(faculties)
	if err != nil {
		fmt.Println("Error selecting faculty:", err)
		return
	}

	err = downloadMaterials(regNo, cookies, selectedSemester, selectedCourse, selectedFaculty)
	if err != nil {
		fmt.Println("Error downloading materials:", err)
		return
	}
}

func fetchAndSelectCourse(regNo string, cookies types.Cookies, semSubId string) (types.Course, error) {
	getCourseURL := "https://vtop.vit.ac.in/vtop/getCourseForCoursePage"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getCourseForCoursePage",
		"semSubId":      semSubId,
		"authorizedID":  regNo,
		"x":             fmt.Sprintf("%d", time.Now().Unix()),
	}

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, getCourseURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching courses:", err)
		}
		return types.Course{}, err
	}

	if debug.Debug {
		fmt.Println("---- Courses Response Body Start ----")
		fmt.Println(string(body))
		fmt.Println("---- Courses Response Body End ----")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing courses HTML:", err)
		}
		return types.Course{}, err
	}

	var courses []types.Course
	doc.Find("select#courseCode option").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		value, exists := s.Attr("value")
		if exists && value != "" {
			text := strings.TrimSpace(s.Text())
			courses = append(courses, types.Course{ID: value, Name: text})
		}
	})

	if len(courses) == 0 {
		if debug.Debug {
			fmt.Println("No courses found for the selected semester.")
		}
		return types.Course{}, fmt.Errorf("no courses found for the selected semester")
	}

	markdownTable := GenerateCourseDetailsMarkdownTable(courses)

	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil && debug.Debug {
		fmt.Println("Error rendering Markdown:", err)
	}

	fmt.Println("\nAvailable Courses:")
	fmt.Println(rendered)

	fmt.Print("Select a Course by entering the number: ")
	var index int
	_, err = fmt.Scanln(&index)
	if err != nil {
		if debug.Debug {
			fmt.Println("Invalid input for course selection:", err)
		}
		return types.Course{}, fmt.Errorf("invalid input for course selection")
	}

	if index < 1 || index > len(courses) {
		fmt.Println("Invalid selection. Please enter a valid number.")
		return types.Course{}, fmt.Errorf("invalid course selection")
	}

	selectedCourse := courses[index-1]

	if debug.Debug {
		fmt.Printf("Selected Course: %s (ID: %s)\n", selectedCourse.Name, selectedCourse.ID)
	}

	return selectedCourse, nil
}

func RemoveCourseCode(courseName string) string {
	re := regexp.MustCompile(`^[A-Z]{4}\d{3}[A-Z]?\s*-\s*`)
	return re.ReplaceAllString(courseName, "")
}

func TruncateString(str string, maxLength int) string {
	if len(str) <= maxLength {
		return str
	}
	if maxLength <= 3 {
		return str[:maxLength]
	}
	return str[:maxLength-3] + "..."
}

func GenerateCourseDetailsMarkdownTable(courses []types.Course) string {
	var sb strings.Builder

	sb.WriteString("| INDEX | COURSE NAME                                       |\n")
	sb.WriteString("|-------|---------------------------------------------------|\n")

	for i, course := range courses {
		index := fmt.Sprintf("%d", i+1)

		courseName := RemoveCourseCode(course.Name)

		courseName = strings.ReplaceAll(courseName, "\n", " ")
		courseName = strings.TrimSpace(courseName)

		maxWidth := 60
		courseName = TruncateString(courseName, maxWidth)

		courseName = fmt.Sprintf("%-60s", courseName)

		sb.WriteString(fmt.Sprintf("| %-5s | %-60s |\n", index, courseName))
	}

	return sb.String()
}

func fetchSlotIds(regNo string, cookies types.Cookies, semSubId string, classId string) ([]string, error) {
	getSlotURL := "https://vtop.vit.ac.in/vtop/getSlotIdForCoursePage"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getSlotIdForCoursePage",
		"semSubId":      semSubId,
		"classId":       classId,
		"praType":       "source",
		"authorizedID":  regNo,
		"x":             time.Now().UTC().Format(time.RFC1123),
	}

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, getSlotURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching slots:", err)
		}
		return nil, err
	}

	if debug.Debug {
		fmt.Println("---- Slots Response Body Start ----")
		fmt.Println(string(body))
		fmt.Println("---- Slots Response Body End ----")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing slots HTML:", err)
		}
		return nil, err
	}

	var slots []string
	doc.Find("select#slotId option").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		value, exists := s.Attr("value")
		if exists && value != "" {
			slots = append(slots, value)
		}
	})

	if len(slots) == 0 {
		if debug.Debug {
			fmt.Println("No slots found for the selected course.")
		}
		return nil, fmt.Errorf("no slots found for the selected course")
	}

	if debug.Debug {
		fmt.Printf("Available Slots: %v\n", slots)
	}

	return slots, nil
}

func fetchFacultiesForAllSlotsConcurrently(regNo string, cookies types.Cookies, semSubId string, classId string, slotIds []string) ([]types.Faculty, error) {
	var allFaculties []types.Faculty
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, slotId := range slotIds {
		wg.Add(1)
		sem <- struct{}{}
		go func(slotId string) {
			defer wg.Done()
			faculties, err := fetchFaculties(regNo, cookies, semSubId, classId, slotId)
			if err != nil {
				if debug.Debug {
					fmt.Printf("Error fetching faculties for slot %s: %v\n", slotId, err)
				}
				<-sem
				return
			}
			mu.Lock()
			allFaculties = append(allFaculties, faculties...)
			mu.Unlock()
			<-sem
		}(slotId)
	}

	wg.Wait()

	uniqueFaculties := removeDuplicateFaculties(allFaculties)

	return uniqueFaculties, nil
}

func fetchFaculties(regNo string, cookies types.Cookies, semSubId string, classId string, slotId string) ([]types.Faculty, error) {
	getFacultyURL := "https://vtop.vit.ac.in/vtop/getFacultyForCoursePage"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getFacultyForCoursePage",
		"semSubId":      semSubId,
		"classId":       classId,
		"slotId":        slotId,
		"praType":       "source",
		"authorizedID":  regNo,
		"x":             time.Now().UTC().Format(time.RFC1123),
	}

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, getFacultyURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching faculty details:", err)
		}
		return nil, err
	}

	if debug.Debug {
		fmt.Println("---- Faculties Response Body Start ----")
		fmt.Println(string(body))
		fmt.Println("---- Faculties Response Body End ----")
	}

	if strings.Contains(string(body), "HTTP Status 404") {
		if debug.Debug {
			fmt.Println("Received 404 Not Found when fetching faculty details.")
		}
		return nil, fmt.Errorf("received 404 Not Found when fetching faculty details")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing faculty list HTML:", err)
		}
		return nil, err
	}

	re := regexp.MustCompile(`processViewStudentCourseDetail\(['"]([^'"]+)['"],\s*['"]([^'"]+)['"],\s*['"]([^'"]+)['"]\)`)

	var faculties []types.Faculty

	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 9 {
			return
		}

		facultyInfo := strings.TrimSpace(cells.Eq(7).Text())

		viewButton := cells.Eq(8).Find("button")
		onclick, exists := viewButton.Attr("onclick")
		if !exists {
			return
		}

		matches := re.FindStringSubmatch(onclick)
		if len(matches) != 4 {
			if debug.Debug {
				fmt.Println("Regex did not match correctly for onclick:", onclick)
			}
			return
		}

		extractedSemSubID := matches[1]
		extractedErpID := matches[2]
		extractedClassID := matches[3]

		faculty := types.Faculty{
			ID:       extractedClassID,
			Name:     facultyInfo,
			ErpID:    extractedErpID,
			ClassID:  extractedClassID,
			SemSubID: extractedSemSubID,
		}

		faculties = append(faculties, faculty)
	})

	if len(faculties) == 0 {
		if debug.Debug {
			fmt.Println("No faculties found for slot ID:", slotId)
		}
		return nil, fmt.Errorf("no faculties found for slot ID: %s", slotId)
	}

	return faculties, nil
}

func removeDuplicateFaculties(faculties []types.Faculty) []types.Faculty {
	uniqueMap := make(map[string]types.Faculty)
	for _, faculty := range faculties {
		if _, exists := uniqueMap[faculty.ErpID]; !exists {
			uniqueMap[faculty.ErpID] = faculty
		}
	}

	var uniqueFaculties []types.Faculty
	for _, faculty := range uniqueMap {
		uniqueFaculties = append(uniqueFaculties, faculty)
	}

	return uniqueFaculties
}

func downloadMaterials(regNo string, cookies types.Cookies, selectedSemester Semester, selectedCourse types.Course, selectedFaculty types.Faculty) error {
	downloadURL := "https://vtop.vit.ac.in/vtop/academics/common/allCourseMeterialDownload"

	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"authorizedID":  regNo,
		"materialMode":  "1",
		"uploadView":    "1",
		"semesterSubId": selectedFaculty.SemSubID,
		"classId":       selectedFaculty.ClassID,
	}

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error sending download request:", err)
		}
		return err
	}

	if !isSuccessfulDownload(body) {
		if debug.Debug {
			fmt.Println("Download response does not start with 'PK', indicating an invalid ZIP file.")
		}
		return fmt.Errorf("failed to download materials, response may indicate an error")
	}

	filename := fmt.Sprintf("course_materials_%d.zip", time.Now().Unix())
	dirName := fmt.Sprintf("%s_%s_%s",
		sanitizeFilename(selectedSemester.SemName),
		sanitizeFilename(selectedCourse.Name),
		sanitizeFilename(selectedFaculty.Name),
	)

	err = os.MkdirAll(dirName, os.ModePerm)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating directory:", err)
		}
		return err
	}

	filePath := filepath.Join(dirName, filename)
	file, err := os.Create(filePath)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating file:", err)
		}
		return err
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error saving file:", err)
		}
		return err
	}

	fmt.Printf("Course materials downloaded successfully to %s\n", filePath)
	return nil
}

func isSuccessfulDownload(body []byte) bool {
	return strings.HasPrefix(string(body), "PK")
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}
