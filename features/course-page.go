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
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Semester struct {
	SemName string
	SemID   string
}

func ExecuteCoursePageDownload(regNo string, cookies types.Cookies) {
	semesterDetails, err := fetchSemesterDetails(regNo, cookies)
	if err != nil {
		fmt.Println("Error fetching semester details:", err)
		return
	}

	selectedSemester, err := selectSemester(semesterDetails)
	if err != nil {
		fmt.Println("Error selecting semester:", err)
		return
	}

	selectedCourse, err := fetchAndSelectCourse(regNo, cookies, selectedSemester.SemID)
	if err != nil {
		fmt.Println("Error selecting course:", err)
		return
	}

	selectedSlot, err := fetchAndSelectSlot(regNo, cookies, selectedSemester.SemID, selectedCourse.ID)
	if err != nil {
		fmt.Println("Error selecting slot:", err)
		return
	}

	selectedFaculty, err := fetchAndSelectFaculty(regNo, cookies, selectedSemester.SemID, selectedCourse.ID, selectedSlot.ID)
	if err != nil {
		fmt.Println("Error selecting faculty:", err)
		return
	}

	err = downloadMaterials(regNo, cookies, selectedSemester, selectedCourse, selectedFaculty)
	if err != nil {
		fmt.Println("Error downloading materials:", err)
		return
	}

	fmt.Println("Course materials downloaded successfully.")
}

func fetchSemesterDetails(regNo string, cookies types.Cookies) (types.SemesterDetails, error) {
	coursePageURL := "https://vtop.vit.ac.in/vtop/academics/common/StudentCoursePage"
	nocache := fmt.Sprintf("%d", time.Now().UnixMilli())

	payloadMap := map[string]string{
		"verifyMenu":   "true",
		"authorizedID": regNo,
		"_csrf":        cookies.CSRF,
		"nocache":      nocache,
	}

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, coursePageURL, "", formData, "POST", "form")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching course page:", err)
		}
		return types.SemesterDetails{}, err
	}

	if debug.Debug {
		fmt.Println("---- Semester Response Body Start ----")
		fmt.Println(string(body))
		fmt.Println("---- Semester Response Body End ----")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing course page HTML:", err)
		}
		return types.SemesterDetails{}, err
	}

	var semNames []string
	var semIds []string

	doc.Find("select#semesterSubId option").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		value, exists := s.Attr("value")
		if exists && value != "" {
			text := strings.TrimSpace(s.Text())
			semNames = append(semNames, text)
			semIds = append(semIds, value)
		}
	})

	if len(semNames) == 0 {
		if debug.Debug {
			fmt.Println("No semesters found in the response.")
		}
		return types.SemesterDetails{}, fmt.Errorf("no semesters found")
	}

	if len(semNames) > 10 {
		semNames = semNames[:10]
		semIds = semIds[:10]
	}

	semesterDetails := types.SemesterDetails{
		SemNames: semNames,
		SemIds:   semIds,
	}

	if debug.Debug {
		fmt.Printf("Extracted Semesters: %v\n", semesterDetails.SemNames)
		fmt.Printf("Extracted Semester IDs: %v\n", semesterDetails.SemIds)
	}

	return semesterDetails, nil
}

func selectSemester(semesterDetails types.SemesterDetails) (Semester, error) {
	for i, semName := range semesterDetails.SemNames {
		fmt.Printf("%d) %s\n", i+1, semName)
	}

	fmt.Print("Select a Semester by entering the number: ")
	var index int
	_, err := fmt.Scanln(&index)
	if err != nil {
		return Semester{}, fmt.Errorf("invalid input: %v", err)
	}

	if index < 1 || index > len(semesterDetails.SemNames) {
		return Semester{}, fmt.Errorf("invalid semester selection")
	}

	selectedSemester := Semester{
		SemName: semesterDetails.SemNames[index-1],
		SemID:   semesterDetails.SemIds[index-1],
	}

	if debug.Debug {
		fmt.Printf("Selected Semester: %s (ID: %s)\n", selectedSemester.SemName, selectedSemester.SemID)
	}

	return selectedSemester, nil
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
	body, err := helpers.FetchReq(regNo, cookies, getCourseURL, "", formData, "POST", "form")
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

	for i, course := range courses {
		fmt.Printf("%d) %s\n", i+1, course.Name)
	}

	fmt.Print("Select a Course by entering the number: ")
	var index int
	_, err = fmt.Scanln(&index)
	if err != nil {
		return types.Course{}, fmt.Errorf("invalid input: %v", err)
	}

	if index < 1 || index > len(courses) {
		return types.Course{}, fmt.Errorf("invalid course selection")
	}

	selectedCourse := courses[index-1]

	if debug.Debug {
		fmt.Printf("Selected Course: %s (ID: %s)\n", selectedCourse.Name, selectedCourse.ID)
	}

	return selectedCourse, nil
}

func fetchAndSelectSlot(regNo string, cookies types.Cookies, semSubId string, classId string) (types.Slot, error) {
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
	body, err := helpers.FetchReq(regNo, cookies, getSlotURL, "", formData, "POST", "form")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching slots:", err)
		}
		return types.Slot{}, err
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
		return types.Slot{}, err
	}

	var slots []types.Slot
	doc.Find("select#slotId option").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		value, exists := s.Attr("value")
		if exists && value != "" {
			text := strings.TrimSpace(s.Text())
			slots = append(slots, types.Slot{ID: value, Name: text})
		}
	})

	if len(slots) == 0 {
		if debug.Debug {
			fmt.Println("No slots found for the selected course.")
		}
		return types.Slot{}, fmt.Errorf("no slots found for the selected course")
	}

	for i, slot := range slots {
		fmt.Printf("%d) %s\n", i+1, slot.Name)
	}

	fmt.Print("Select a Slot by entering the number: ")
	var index int
	_, err = fmt.Scanln(&index)
	if err != nil {
		return types.Slot{}, fmt.Errorf("invalid input: %v", err)
	}

	if index < 1 || index > len(slots) {
		return types.Slot{}, fmt.Errorf("invalid slot selection")
	}

	selectedSlot := slots[index-1]

	if debug.Debug {
		fmt.Printf("Selected Slot: %s (ID: %s)\n", selectedSlot.Name, selectedSlot.ID)
	}

	return selectedSlot, nil
}

func fetchAndSelectFaculty(regNo string, cookies types.Cookies, semSubId string, courseID string, slotID string) (types.Faculty, error) {
	getFacultyURL := "https://vtop.vit.ac.in/vtop/getFacultyForCoursePage"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getFacultyForCoursePage",
		"semSubId":      semSubId,
		"classId":       courseID,
		"slotId":        slotID,
		"praType":       "source",
		"authorizedID":  regNo,
		"x":             time.Now().UTC().Format(time.RFC1123),
	}

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, getFacultyURL, "", formData, "POST", "form")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching faculty details:", err)
		}
		return types.Faculty{}, err
	}

	if debug.Debug {
		fmt.Println("---- Faculties Response Body Start ----")
		fmt.Println(string(body))
		fmt.Println("---- Faculties Response Body End ----")
	}

	if strings.Contains(string(body), "HTTP Status 404") {
		return types.Faculty{}, fmt.Errorf("received 404 Not Found when fetching faculty details")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing faculty list HTML:", err)
		}
		return types.Faculty{}, err
	}

	re := regexp.MustCompile(`processViewStudentCourseDetail\('([^']+)','([^']+)','([^']+)'\)`)
	var faculties []types.Faculty

	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 9 {
			return
		}

		semName := strings.TrimSpace(cells.Eq(1).Text())
		courseCode := strings.TrimSpace(cells.Eq(2).Text())
		courseTitle := strings.TrimSpace(cells.Eq(3).Text())
		fullCourseName := fmt.Sprintf("%s - %s", courseCode, courseTitle)
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
			ID:           extractedClassID,
			Name:         facultyInfo,
			ErpID:        extractedErpID,
			ClassID:      extractedClassID,
			SemesterName: semName,
			CourseName:   fullCourseName,
			SemSubID:     extractedSemSubID,
		}

		faculties = append(faculties, faculty)
	})

	if len(faculties) == 0 {
		if debug.Debug {
			fmt.Println("No faculties found for the selected course.")
		}
		return types.Faculty{}, fmt.Errorf("no faculties found for the selected course")
	}

	for i, faculty := range faculties {
		fmt.Printf("%d) %s (ERP ID: %s)\n", i+1, faculty.Name, faculty.ErpID)
	}

	fmt.Print("Select a Faculty by entering the number: ")
	var index int
	_, err = fmt.Scanln(&index)
	if err != nil {
		return types.Faculty{}, fmt.Errorf("invalid input: %v", err)
	}

	if index < 1 || index > len(faculties) {
		return types.Faculty{}, fmt.Errorf("invalid faculty selection")
	}

	selectedFaculty := faculties[index-1]

	if debug.Debug {
		fmt.Printf("Selected Faculty: %s (ERP ID: %s)\n", selectedFaculty.Name, selectedFaculty.ErpID)
	}

	return selectedFaculty, nil
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
	body, err := helpers.FetchReq(regNo, cookies, downloadURL, "", formData, "POST", "form")
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

	if debug.Debug {
		fmt.Printf("Course materials downloaded successfully to %s\n", filePath)
	} else {
		fmt.Printf("Course materials downloaded successfully to %s\n", filePath)
	}

	return nil
}

func isSuccessfulDownload(body []byte) bool {
	return strings.HasPrefix(string(body), "PK")
}

func sanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
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
