package features

import (
	"bufio"
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"archive/zip"

	"github.com/PuerkitoBio/goquery"
	"github.com/schollz/progressbar/v3"
)

const (
	CourseOptionSelector = "select#courseCode option"
	SlotOptionSelector   = "select#slotId option"
	CourseTableSelector  = "table"
	CourseRowSelector    = "tbody tr"
	CourseCellSelector   = "td"
)

var newHttpClient *http.Client

func init() {
	newHttpClient = &http.Client{
		Timeout: time.Duration(60) * time.Second,
	}
}

func ExecuteCoursePageDownload(regNo string, cookies types.Cookies, semesterFlag int, courseFlag int, facultyFlag string, fuzzyFlag int) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	selectedSemester, err := helpers.SelectSemester(regNo, cookies, semesterFlag)
	if err != nil {
		if err.Error() == "selection canceled by user" {
			fmt.Println("Selection canceled")
			return
		}
		if debug.Debug {
			fmt.Println(err)
		}
		fmt.Println("Error selecting semester:", err)
		return
	}

	selectedCourse, err := fetchAndSelectCourse(regNo, cookies, selectedSemester.SemID, courseFlag)
	if err != nil {
		fmt.Println("Error selecting course:", err)
		return
	}

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

	selectedFaculty, err := selectFaculty(faculties, facultyFlag)
	if err != nil {
		// Check if this is a selection canceled error or a real error
		if err.Error() == "selection canceled by user" {
			fmt.Println("Selection canceled")
			return
		}
		fmt.Println("Error selecting faculty:", err)
		return
	}

	htmlContent, err := fetchCourseMaterialsPage(regNo, cookies, selectedFaculty)
	if err != nil {
		fmt.Println("Error fetching course materials page:", err)
		return
	}

	materials, err := parseCourseMaterialsPage(htmlContent)
	if err != nil {
		fmt.Println("Error parsing course materials:", err)
		return
	}

	if len(materials) == 0 {
		fmt.Println("No course materials with reference materials available for download.")
		return
	}

	displayCourseMaterials(materials)

	selectedMaterials, err := selectCourseMaterials(materials)
	if err != nil {
		fmt.Println("Error selecting materials:", err)
		return
	}

	err = downloadMaterialsIndividually(regNo, cookies, selectedCourse, selectedFaculty, materials, selectedMaterials)
	if err != nil {
		fmt.Printf("Error downloading materials: %v\n", err)
		return
	}

	fmt.Println("\nDownload complete!")
}

func fetchAndSelectCourse(regNo string, cookies types.Cookies, semSubId string, courseFlag int) (types.Course, error) {
	getCourseURL := "https://vtop.vit.ac.in/vtop/getCourseForCoursePage"
	payloadMap := map[string]string{
		"_csrf":         cookies.CSRF,
		"paramReturnId": "getCourseForCoursePage",
		"semSubId":      semSubId,
		"authorizedID":  regNo,
		"x":             time.Now().UTC().Format(time.RFC1123),
	}

	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(newHttpClient, regNo, cookies, getCourseURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return types.Course{}, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return types.Course{}, err
	}

	var courses []types.Course
	doc.Find(CourseOptionSelector).Each(func(_ int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists && value != "" {
			text := strings.TrimSpace(s.Text())
			courses = append(courses, types.Course{ID: value, Name: text})
		}
	})

	if len(courses) == 0 {
		return types.Course{}, fmt.Errorf("no courses found for the selected semester")
	}

	nestedList := [][]string{{"COURSE NAME"}}
	for _, course := range courses {
		nestedList = append(nestedList, []string{course.Name})
	}

	result := helpers.TableSelector("Course", nestedList, strconv.Itoa(courseFlag))
	if result.ExitRequest {
		return types.Course{}, fmt.Errorf("selection canceled by user")
	}

	if !result.Selected || result.Index < 1 || result.Index > len(courses) {
		return types.Course{}, fmt.Errorf("invalid course selection")
	}

	selectedCourse := courses[result.Index-1]
	return selectedCourse, nil
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

	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(newHttpClient, regNo, cookies, getSlotURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var slots []string
	doc.Find(SlotOptionSelector).Each(func(_ int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists && value != "" {
			slots = append(slots, value)
		}
	})

	if len(slots) == 0 {
		return nil, fmt.Errorf("no slots found for the selected course")
	}

	return slots, nil
}

func fetchFacultiesForAllSlotsConcurrently(regNo string, cookies types.Cookies, semSubId string, classId string, slotIds []string) ([]types.Faculty, error) {
	var wg sync.WaitGroup
	concurrency := getOptimizedConcurrency()
	sem := make(chan struct{}, concurrency)

	facultySlices := make([][]types.Faculty, len(slotIds))
	errorsOccurred := false
	var mu sync.Mutex

	for i, slotId := range slotIds {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, slotId string) {
			defer wg.Done()
			defer func() { <-sem }()
			faculties, err := fetchFaculties(newHttpClient, regNo, cookies, semSubId, classId, slotId)
			if err != nil {
				if debug.Debug {
					fmt.Printf("Error fetching faculties for slot %s: %v\n", slotId, err)
				}
				mu.Lock()
				errorsOccurred = true
				mu.Unlock()
				return
			}
			facultySlices[i] = faculties
		}(i, slotId)
	}

	wg.Wait()

	if errorsOccurred {
		return nil, fmt.Errorf("some faculties could not be fetched")
	}

	var allFaculties []types.Faculty
	for _, slice := range facultySlices {
		allFaculties = append(allFaculties, slice...)
	}

	uniqueFaculties := helpers.RemoveDuplicateFaculties(allFaculties)
	helpers.SortFacultiesAlphabetically(uniqueFaculties)

	return uniqueFaculties, nil
}

func fetchFaculties(client *http.Client, regNo string, cookies types.Cookies, semSubId string, classId string, slotId string) ([]types.Faculty, error) {
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

	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(client, regNo, cookies, getFacultyURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(body), "HTTP Status 404") {
		return nil, fmt.Errorf("received 404 Not Found when fetching faculty details")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`processViewStudentCourseDetail\(['"]([^'"]+)['"],\s*['"]([^'"]+)['"],\s*['"]([^'"]+)['"]\)`)

	var faculties []types.Faculty

	doc.Find("table tbody tr").Each(func(_ int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 9 {
			return
		}

		slotInfo := strings.TrimSpace(cells.Eq(6).Text())
		slotInfo = helpers.ReplaceCrossWithPlus(slotInfo)
		facultyInfo := strings.TrimSpace(cells.Eq(7).Text())

		viewButton := cells.Eq(8).Find("button")
		onclick, exists := viewButton.Attr("onclick")
		if !exists {
			return
		}

		matches := re.FindStringSubmatch(onclick)
		if len(matches) != 4 {
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
			Slot:     slotInfo,
		}

		faculties = append(faculties, faculty)
	})

	if len(faculties) == 0 {
		return nil, fmt.Errorf("no faculties found for slot ID: %s", slotId)
	}

	return faculties, nil
}

func selectFaculty(faculties []types.Faculty, facultyFlag string) (types.Faculty, error) {
	nestedList := [][]string{{"NAME", "SLOT"}}
	for _, faculty := range faculties {
		cleanName := removeNumberPrefix(faculty.Name)
		nestedList = append(nestedList, []string{
			strings.TrimSpace(cleanName),
			faculty.Slot,
		})
	}

	if runtime.GOOS == "windows" {
		clearSingleNewline()
	}

	result := helpers.TableSelectorFuzzy("Faculty", nestedList, facultyFlag, helpers.NewFuzzySearch)

	if result.ExitRequest {
		return types.Faculty{}, fmt.Errorf("selection canceled by user")
	}

	if !result.Selected || result.Index <= 0 || result.Index > len(faculties) {
		return types.Faculty{}, fmt.Errorf("invalid faculty selection")
	}

	return faculties[result.Index-1], nil
}

func removeNumberPrefix(facultyName string) string {
	parts := strings.SplitN(facultyName, " - ", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return facultyName
}

func fetchCourseMaterialsPage(regNo string, cookies types.Cookies, selectedFaculty types.Faculty) (string, error) {
	url := "https://vtop.vit.ac.in/vtop/processViewStudentCourseDetail"
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"semSubId":     selectedFaculty.SemSubID,
		"erpId":        selectedFaculty.ErpID,
		"classId":      selectedFaculty.ClassID,
		"authorizedID": regNo,
		"x":            time.Now().UTC().Format(time.RFC1123),
	}
	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(newHttpClient, regNo, cookies, url, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func parseCourseMaterialsPage(htmlContent string) ([]types.CourseMaterial, error) {
	var materials []types.CourseMaterial
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		multiHeader := false
		headerRows := table.Find("thead tr")
		if headerRows.Length() > 0 {
			headerRows.Each(func(i int, row *goquery.Selection) {
				row.Find("th, td").Each(func(j int, cell *goquery.Selection) {
					if colspan, exists := cell.Attr("colspan"); exists {
						if c, err := strconv.Atoi(colspan); err == nil && c > 1 {
							multiHeader = true
						}
					}
				})
			})
		}

		if multiHeader && headerRows.Length() > 1 {
			topicGroupStart := -1
			topicColSpan := 0
			firstHeaderRow := headerRows.First()
			firstHeaderRow.Find("th, td").Each(func(j int, cell *goquery.Selection) {
				text := strings.ToLower(strings.TrimSpace(cell.Text()))
				if strings.Contains(text, "topic") {
					topicGroupStart = j
					if colspan, exists := cell.Attr("colspan"); exists {
						if c, err := strconv.Atoi(colspan); err == nil {
							topicColSpan = c
						}
					}
				}
			})
			if topicGroupStart == -1 {
				return
			}
			if topicColSpan == 0 {
				topicColSpan = 1
			}
			secondHeaderRow := headerRows.Eq(1)
			mNoIndex, topicContentIndex, tNoIndex, moduleTitleIndex := -1, -1, -1, -1
			secondHeaderRow.Find("th, td").Each(func(j int, cell *goquery.Selection) {
				cellText := strings.ToLower(strings.TrimSpace(cell.Text()))
				if mNoIndex == -1 && strings.Contains(cellText, "m.no") {
					mNoIndex = j
				}
				if topicContentIndex == -1 && strings.Contains(cellText, "topic content") {
					topicContentIndex = j
				}
				if tNoIndex == -1 && strings.Contains(cellText, "t.no") {
					tNoIndex = j
				}
				if moduleTitleIndex == -1 && strings.Contains(cellText, "module title") {
					moduleTitleIndex = j
				}
			})
			if mNoIndex == -1 {
				mNoIndex = 0
			}
			if topicContentIndex == -1 {
				if topicColSpan > 4 {
					topicContentIndex = 4
				} else {
					topicContentIndex = topicColSpan - 1
				}
			}
			if tNoIndex == -1 {
				tNoIndex = 3
			}
			table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
				cells := row.Find("td")
				if cells.Length() < (3 + topicColSpan + 1) {
					return
				}

				index, _ := strconv.Atoi(strings.TrimSpace(cells.Eq(0).Text()))
				dateVal := strings.TrimSpace(cells.Eq(1).Text())
				dayOrderSlotVal := strings.TrimSpace(cells.Eq(2).Text())
				dayOrderSlotVal = helpers.ReplaceCrossWithPlus(dayOrderSlotVal)

				// Extract module number and topic number
				moduleNumberVal := ""
				if topicGroupStart+mNoIndex < cells.Length() {
					moduleNumberVal = strings.TrimSpace(cells.Eq(topicGroupStart + mNoIndex).Text())
				}

				topicNumberVal := ""
				if topicGroupStart+tNoIndex < cells.Length() {
					topicNumberVal = strings.TrimSpace(cells.Eq(topicGroupStart + tNoIndex).Text())
				}

				// Get the topic content value
				topicContentVal := ""
				if topicGroupStart+topicContentIndex < cells.Length() {
					topicContentVal = strings.TrimSpace(cells.Eq(topicGroupStart + topicContentIndex).Text())
				}

				topicVal := ""
				if topicContentVal != "" {
					// Use the topic content instead of module title
					topicVal = fmt.Sprintf("%s - %s", moduleNumberVal, topicContentVal)
				} else {
					mNoCell := cells.Eq(topicGroupStart + mNoIndex)
					topicContentCell := cells.Eq(topicGroupStart + topicContentIndex)
					topicVal = strings.TrimSpace(mNoCell.Text()) + " - " + strings.TrimSpace(topicContentCell.Text())
				}

				if topicVal == "" {
					topicVal = "Unnamed"
				}

				refCell := cells.Eq(cells.Length() - 1)
				var refMaterials []types.ReferenceMaterial
				refCell.Find("button[name='getDownloadSemPdf']").Each(func(k int, btn *goquery.Selection) {
					materialID, _ := btn.Attr("data-matid")
					materialDate, _ := btn.Attr("data-mdate")
					name := strings.TrimSpace(btn.Find("span").Text())
					refMaterials = append(refMaterials, types.ReferenceMaterial{
						Name:         name,
						MaterialID:   materialID,
						MaterialDate: materialDate,
					})
				})
				webLink := ""
				refCell.Find("a[target='_blank']").Each(func(k int, a *goquery.Selection) {
					href, exists := a.Attr("href")
					if exists && strings.HasPrefix(href, "http") {
						webLink = href
					}
				})
				if len(refMaterials) > 0 || webLink != "" {
					material := types.CourseMaterial{
						Index:              index,
						Date:               dateVal,
						DayOrderSlot:       dayOrderSlotVal,
						Topic:              topicVal,
						ReferenceMaterials: refMaterials,
						WebLink:            webLink,
						MNo:                moduleNumberVal,
						TNo:                topicNumberVal,
					}
					materials = append(materials, material)
				}
			})
		} else {
			var headerCells []*goquery.Selection
			if table.Find("thead").Length() > 0 {
				table.Find("thead tr").First().Find("th, td").Each(func(j int, cell *goquery.Selection) {
					headerCells = append(headerCells, cell)
				})
			} else {
				table.Find("tr").First().Find("th, td").Each(func(j int, cell *goquery.Selection) {
					headerCells = append(headerCells, cell)
				})
			}
			if len(headerCells) == 0 {
				return
			}

			sNoIndex, dateIndex, dayOrderSlotIndex, topicIndex, refMaterialIndex, moduleNumberIndex, topicNumberIndex := -1, -1, -1, -1, -1, -1, -1
			for j, cell := range headerCells {
				text := strings.ToLower(strings.TrimSpace(cell.Text()))
				if sNoIndex == -1 && strings.Contains(text, "Sl.No.") {
					sNoIndex = j
				}
				if dateIndex == -1 && strings.Contains(text, "date") {
					dateIndex = j
				}
				if dayOrderSlotIndex == -1 && (strings.Contains(text, "day") || strings.Contains(text, "slot")) {
					dayOrderSlotIndex = j
				}
				if topicIndex == -1 && strings.Contains(text, "topic") {
					topicIndex = j
				}
				if refMaterialIndex == -1 && (strings.Contains(text, "reference") || strings.Contains(text, "material")) {
					refMaterialIndex = j
				}
				if moduleNumberIndex == -1 && strings.Contains(text, "m.no") {
					moduleNumberIndex = j
				}
				if topicNumberIndex == -1 && strings.Contains(text, "t.no") {
					topicNumberIndex = j
				}
			}

			if sNoIndex == -1 && dateIndex == -1 && dayOrderSlotIndex == -1 && topicIndex == -1 && refMaterialIndex == -1 {
				return
			}

			table.Find("tr").Each(func(i int, row *goquery.Selection) {
				if i == 0 {
					return
				}
				cells := row.Find("td")
				if cells.Length() < 1 {
					return
				}

				slNo := 1
				var err error
				if sNoIndex >= 0 && cells.Length() > sNoIndex {
					slNo, err = strconv.Atoi(strings.TrimSpace(cells.Eq(sNoIndex).Text()))
					if err != nil {
						slNo = 1
					}
				}
				dateVal := ""
				if dateIndex >= 0 && cells.Length() > dateIndex {
					dateVal = strings.TrimSpace(cells.Eq(dateIndex).Text())
				}
				dayOrderSlotVal := ""
				if dayOrderSlotIndex >= 0 && cells.Length() > dayOrderSlotIndex {
					dayOrderSlotVal = strings.TrimSpace(cells.Eq(dayOrderSlotIndex).Text())
					dayOrderSlotVal = helpers.ReplaceCrossWithPlus(dayOrderSlotVal)
				}

				// Extract module number and topic number
				moduleNumberVal := ""
				if moduleNumberIndex >= 0 && cells.Length() > moduleNumberIndex {
					moduleNumberVal = strings.TrimSpace(cells.Eq(moduleNumberIndex).Text())
				}

				topicNumberVal := ""
				if topicNumberIndex >= 0 && cells.Length() > topicNumberIndex {
					topicNumberVal = strings.TrimSpace(cells.Eq(topicNumberIndex).Text())
				}

				topicVal := ""
				if topicIndex >= 0 && cells.Length() > topicIndex {
					topicContentVal := strings.TrimSpace(cells.Eq(topicIndex).Text())
					if moduleNumberVal != "" && topicNumberVal != "" {
						// Format with module and topic numbers
						topicVal = fmt.Sprintf("%s - %s - %s", moduleNumberVal, topicNumberVal, topicContentVal)
					} else {
						topicVal = topicContentVal
					}
					if topicVal == "" {
						topicVal = "Unnamed"
					}
				}
				if refMaterialIndex >= 0 && cells.Length() > refMaterialIndex {
					refCell := cells.Eq(refMaterialIndex)
					var refMaterials []types.ReferenceMaterial
					refCell.Find("button[name='getDownloadSemPdf']").Each(func(k int, btn *goquery.Selection) {
						materialID, _ := btn.Attr("data-matid")
						materialDate, _ := btn.Attr("data-mdate")
						name := strings.TrimSpace(btn.Find("span").Text())
						refMaterials = append(refMaterials, types.ReferenceMaterial{
							Name:         name,
							MaterialID:   materialID,
							MaterialDate: materialDate,
						})
					})
					webLink := ""
					refCell.Find("a[target='_blank']").Each(func(k int, a *goquery.Selection) {
						href, exists := a.Attr("href")
						if exists && strings.HasPrefix(href, "http") {
							webLink = href
						}
					})
					if len(refMaterials) > 0 || webLink != "" {
						material := types.CourseMaterial{
							Index:              slNo,
							Date:               dateVal,
							DayOrderSlot:       dayOrderSlotVal,
							Topic:              topicVal,
							ReferenceMaterials: refMaterials,
							WebLink:            webLink,
							MNo:                moduleNumberVal,
							TNo:                topicNumberVal,
						}
						materials = append(materials, material)
					}
				}
			})
		}
	})

	return materials, nil
}

func displayCourseMaterials(materials []types.CourseMaterial) {
	showWebColumn := false
	for _, material := range materials {
		if strings.TrimSpace(material.WebLink) != "" {
			showWebColumn = true
			break
		}
	}

	var header []string
	if showWebColumn {
		header = []string{"DATE", "TOPIC", "REF COUNT", "WEB MATERIAL"}
	} else {
		header = []string{"DATE", "TOPIC", "REF COUNT"}
	}

	// Increase topic column width from 30 to 50
	nestedList := [][]string{header}
	for _, material := range materials {
		refCount := strconv.Itoa(len(material.ReferenceMaterials))
		topic := helpers.TruncateWithEllipsis(material.Topic, 50)
		if showWebColumn {
			webCol := ""
			if strings.TrimSpace(material.WebLink) != "" {
				webCol = helpers.MakeANSILink("Open", material.WebLink)
			}
			nestedList = append(nestedList, []string{
				material.Date,
				topic,
				refCount,
				webCol,
			})
		} else {
			nestedList = append(nestedList, []string{
				material.Date,
				topic,
				refCount,
			})
		}
	}
	fmt.Println()
	helpers.PrintTable(nestedList, 1)
}

func selectCourseMaterials(materials []types.CourseMaterial) ([]types.CourseMaterial, error) {
	for {
		fmt.Println()
		fmt.Print("Enter the index numbers of the topics to download (e.g., 1,2-5,8,5,3), or 0 for bulk download: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil && debug.Debug {
			fmt.Println("Error reading input:", err)
			return nil, err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Println("No input provided.")
			continue
		}

		if input == "0" {
			return materials, nil
		}

		selectedIndices, invalidInputs := parseIndices(input, len(materials))
		if len(invalidInputs) > 0 {
			fmt.Println("Invalid indices:", strings.Join(invalidInputs, ", "))
		}

		if len(selectedIndices) == 0 {
			fmt.Println("No valid indices selected.")
			continue
		}

		var selectedMaterials []types.CourseMaterial
		for _, idx := range selectedIndices {
			selectedMaterials = append(selectedMaterials, materials[idx-1])
		}

		return selectedMaterials, nil
	}
}

func parseIndices(input string, max int) ([]int, []string) {
	var indices []int
	var invalid []string
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				invalid = append(invalid, part)
				continue
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end || start < 1 || end > max {
				invalid = append(invalid, part)
				continue
			}
			for i := start; i <= end; i++ {
				indices = append(indices, i)
			}
		} else {
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 1 || idx > max {
				invalid = append(invalid, part)
				continue
			}
			indices = append(indices, idx)
		}
	}

	uniqueIndices := helpers.RemoveDuplicates(indices)
	return uniqueIndices, invalid
}

func downloadMaterialsIndividually(regNo string, cookies types.Cookies, selectedCourse types.Course, selectedFaculty types.Faculty, allMaterials []types.CourseMaterial, selectedMaterials []types.CourseMaterial) error {
	// Create the Course Page directory
	coursePageDir, err := helpers.GetOrCreateDownloadDir("Course Page")
	if err != nil {
		return fmt.Errorf("failed to create course page directory: %w", err)
	}

	courseParts := helpers.SplitCourseNameFull(selectedCourse.Name)
	var courseFolderName string
	if len(courseParts) >= 3 {
		// Extract the course code from the course name (typically the first part)
		courseCode := courseParts[0]
		// Extract the course name parts (typically everything after the first part)
		courseName := strings.Join(courseParts[1:], "_")
		courseFolderName = fmt.Sprintf("%s_%s", courseName, courseCode)
	} else if len(courseParts) == 2 {
		courseFolderName = fmt.Sprintf("%s_%s", courseParts[1], courseParts[0])
	} else if len(courseParts) == 1 {
		courseFolderName = courseParts[0]
	} else {
		courseFolderName = "Unknown_Course"
	}
	courseFolderName = helpers.SanitizeFilename(courseFolderName)

	slotID := func() string {
		if len(selectedFaculty.Slot) >= 3 && strings.HasPrefix(selectedFaculty.Slot, "L") {
			return selectedFaculty.Slot[:3]
		} else if len(selectedFaculty.Slot) >= 2 {
			return selectedFaculty.Slot[:2]
		}
		return selectedFaculty.Slot
	}()

	facultyNameNoERP := helpers.RedactERPID(selectedFaculty.Name)
	facultyParts := helpers.SplitFacultyNameFull(facultyNameNoERP)
	var facultyFolderName string
	if len(facultyParts) >= 2 {
		facultyNamePart := strings.ReplaceAll(facultyParts[0], " ", "-")
		facultyFolderName = fmt.Sprintf("%s_%s_%s", slotID, facultyNamePart, facultyParts[1])
	} else if len(facultyParts) == 1 {
		facultyNamePart := strings.ReplaceAll(facultyParts[0], " ", "-")
		facultyFolderName = fmt.Sprintf("%s_%s", slotID, facultyNamePart)
	} else {
		facultyFolderName = "Unknown_Faculty"
	}
	facultyFolderName = helpers.SanitizeFilename(facultyFolderName)

	fullDirPath := filepath.Join(coursePageDir, courseFolderName, facultyFolderName)

	err = os.MkdirAll(fullDirPath, os.ModePerm)
	if err != nil {
		return err
	}

	if selectedMaterials == nil {
		selectedMaterials = allMaterials
	}

	if helpers.IsRateLimitExceeded() {
		fmt.Println("Rate limit exceeded. Please try again later.")
		return fmt.Errorf("rate limit exceeded")
	}

	if _, err := os.Stat(fullDirPath); os.IsNotExist(err) {
		return fmt.Errorf("download directory does not exist: %s", fullDirPath)
	}

	totalRefMaterials := 0
	for _, material := range selectedMaterials {
		totalRefMaterials += len(material.ReferenceMaterials)
	}

	bar := progressbar.NewOptions(totalRefMaterials,
		progressbar.OptionSetDescription("Downloading materials..."),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionClearOnFinish(),
	)

	concurrency := 2

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex

	type FailedDownload struct {
		IndexNo       int
		TopicName     string
		RefMaterialNo int
		RefMat        types.ReferenceMaterial
		Topic         string
		Error         string
		MNo           string
		TNo           string
	}
	var failedDownloads []FailedDownload
	var failedMu sync.Mutex

	type DownloadKey struct {
		MaterialID   string
		MaterialDate string
	}
	successfulDownloads := make(map[DownloadKey]bool)
	var successMu sync.Mutex

	for _, material := range selectedMaterials {
		if material.WebLink != "" {
			fmt.Printf("Web Material available for '%s'\n", material.Topic)
			fmt.Printf("Link: %s\n", material.WebLink)
		}

		topicName := helpers.SanitizeFilename(material.Topic)
		// Ensure the topic name isn't too long to avoid path length issues
		if len(topicName) > 50 {
			topicName = topicName[:50]
		}
		refMaterialNo := 1

		for _, refMat := range material.ReferenceMaterials {
			wg.Add(1)
			sem <- struct{}{}
			currentIndexNo := material.Index
			currentTopicName := topicName
			currentRefMaterialNo := refMaterialNo
			currentRefMat := refMat
			currentTopic := material.Topic
			currentMNo := material.MNo
			currentTNo := material.TNo

			go func(indexNo int, topicName string, refMaterialNo int, refMat types.ReferenceMaterial, topic, mNo, tNo string) {
				defer wg.Done()
				defer func() { <-sem }()

				key := DownloadKey{MaterialID: refMat.MaterialID, MaterialDate: refMat.MaterialDate}
				successMu.Lock()
				alreadyDownloaded := successfulDownloads[key]
				successMu.Unlock()

				if alreadyDownloaded {
					mu.Lock()
					bar.Add(1)
					mu.Unlock()
					return
				}

				isPotentialPptx := strings.Contains(strings.ToLower(refMat.Name), "ppt") ||
					strings.Contains(strings.ToLower(refMat.Name), "presentation") ||
					strings.Contains(strings.ToLower(refMat.Name), "slide")

				downloadURL := "https://vtop.vit.ac.in/vtop/downloadPdf"
				payloadMap := map[string]string{
					"_csrf":        cookies.CSRF,
					"authorizedID": regNo,
					"semSubId":     selectedFaculty.SemSubID,
					"classId":      selectedFaculty.ClassID,
					"materialId":   refMat.MaterialID,
					"materialDate": refMat.MaterialDate,
					"x":            time.Now().UTC().Format(time.RFC1123),
				}
				formData := helpers.FormatBodyDataClient(payloadMap)

				var body []byte
				var headers http.Header
				retries := 5
				var downloadErr error
				var lastError string

				for attempt := 1; attempt <= retries; attempt++ {
					attemptClient := &http.Client{
						Timeout: time.Minute * 2,
						Transport: &http.Transport{
							MaxIdleConns:        10,
							MaxIdleConnsPerHost: 5,
							IdleConnTimeout:     30 * time.Second,
							DisableKeepAlives:   true,
						},
					}

					if isPotentialPptx {
						attemptClient.Timeout = time.Minute * 5
					}

					body, headers, downloadErr = helpers.FetchReqClient(attemptClient, regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")
					if downloadErr == nil && len(body) > 0 {
						if (isPotentialPptx && len(body) > 4096) || isSuccessfulDownload(body) {
							break
						} else {
							downloadErr = fmt.Errorf("invalid file content")
							lastError = "Invalid file content"
						}
					} else if downloadErr != nil {
						lastError = downloadErr.Error()
					} else {
						lastError = "Empty response"
					}

					if debug.Debug {
						fmt.Printf("Attempt %d: Error downloading material ID %s: %v\n", attempt, refMat.MaterialID, downloadErr)
					}

					backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
					jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
					time.Sleep(backoffTime + jitter)
				}

				if downloadErr != nil || !isSuccessfulDownload(body) {
					if debug.Debug {
						fmt.Printf("Failed to download material ID %s after %d attempts: %v\n", refMat.MaterialID, retries, downloadErr)
					}

					failedMu.Lock()
					failedDownloads = append(failedDownloads, FailedDownload{
						IndexNo:       indexNo,
						TopicName:     topicName,
						RefMaterialNo: refMaterialNo,
						RefMat:        refMat,
						Topic:         topic,
						Error:         lastError,
						MNo:           mNo,
						TNo:           tNo,
					})
					failedMu.Unlock()

					mu.Lock()
					bar.Add(1)
					mu.Unlock()
					return
				}

				ext := helpers.GetFileExtension(refMat.Name, body, headers)
				if ext == "" {
					if isPotentialPptx {
						ext = ".pptx"
					} else {
						ext = ".bin"
					}
				}

				filePath := generateFilePath(fullDirPath, indexNo, mNo, tNo, topicName, refMaterialNo, ext)

				err = helpers.SaveFile(body, filePath)
				if err != nil {
					if debug.Debug {
						fmt.Printf("Error saving file: %v\n", err)
					}

					failedMu.Lock()
					failedDownloads = append(failedDownloads, FailedDownload{
						IndexNo:       indexNo,
						TopicName:     topicName,
						RefMaterialNo: refMaterialNo,
						RefMat:        refMat,
						Topic:         topic,
						Error:         err.Error(),
						MNo:           mNo,
						TNo:           tNo,
					})
					failedMu.Unlock()
				} else {
					successMu.Lock()
					successfulDownloads[key] = true
					successMu.Unlock()
				}

				mu.Lock()
				bar.Add(1)
				mu.Unlock()
			}(currentIndexNo, currentTopicName, currentRefMaterialNo, currentRefMat, currentTopic, currentMNo, currentTNo)
			refMaterialNo++
		}
	}

	wg.Wait()

	var permanentlyFailedDownloads []FailedDownload

	if len(failedDownloads) > 0 {
		fmt.Printf("\nRetrying %d failed downloads...\n", len(failedDownloads))
		retryBar := progressbar.NewOptions(len(failedDownloads),
			progressbar.OptionSetDescription("Retrying failed downloads..."),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionClearOnFinish(),
		)

		for i, fd := range failedDownloads {
			// Check if this file has already been successfully downloaded in a previous retry
			key := DownloadKey{MaterialID: fd.RefMat.MaterialID, MaterialDate: fd.RefMat.MaterialDate}
			if successfulDownloads[key] {
				retryBar.Add(1)
				continue
			}

			isPotentialPptx := strings.Contains(strings.ToLower(fd.RefMat.Name), "ppt") ||
				strings.Contains(strings.ToLower(fd.RefMat.Name), "presentation") ||
				strings.Contains(strings.ToLower(fd.RefMat.Name), "slide")

			downloadURL := "https://vtop.vit.ac.in/vtop/downloadPdf"
			payloadMap := map[string]string{
				"_csrf":        cookies.CSRF,
				"authorizedID": regNo,
				"semSubId":     selectedFaculty.SemSubID,
				"classId":      selectedFaculty.ClassID,
				"materialId":   fd.RefMat.MaterialID,
				"materialDate": fd.RefMat.MaterialDate,
				"x":            time.Now().UTC().Format(time.RFC1123),
			}
			formData := helpers.FormatBodyDataClient(payloadMap)

			var body []byte
			var headers http.Header
			var downloadErr error
			var lastError string

			retryClient := &http.Client{
				Timeout: time.Minute * 5,
				Transport: &http.Transport{
					MaxIdleConns:        5,
					MaxIdleConnsPerHost: 2,
					IdleConnTimeout:     90 * time.Second,
					DisableKeepAlives:   true,
				},
			}

			if isPotentialPptx {
				retryClient.Timeout = time.Minute * 10
			}

			success := false

			for attempt := 1; attempt <= 5; attempt++ {
				if attempt > 1 {
					sleepTime := time.Duration(attempt*3) * time.Second
					time.Sleep(sleepTime)
				}

				body, headers, downloadErr = helpers.FetchReqClient(retryClient, regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")

				if downloadErr == nil && len(body) > 0 {
					if (isPotentialPptx && len(body) > 4096) || isSuccessfulDownload(body) {
						ext := helpers.GetFileExtension(fd.RefMat.Name, body, headers)
						if ext == "" {
							if isPotentialPptx {
								ext = ".pptx"
							} else {
								ext = ".bin"
							}
						}

						filePath := generateFilePath(fullDirPath, fd.IndexNo, fd.MNo, fd.TNo, fd.TopicName, fd.RefMaterialNo, ext)

						err = helpers.SaveFile(body, filePath)
						if err == nil {
							success = true
							successfulDownloads[key] = true
							break
						} else {
							lastError = err.Error()
						}
					} else {
						lastError = "Invalid file content"
					}
				} else if downloadErr != nil {
					lastError = downloadErr.Error()
				} else {
					lastError = "Invalid or empty file content"
				}

				if attempt == 5 {
					time.Sleep(5 * time.Second)

					var timeout time.Duration
					if isPotentialPptx {
						timeout = time.Minute * 15
					} else {
						timeout = time.Minute * 10
					}

					freshClient := &http.Client{
						Timeout: timeout,
						Transport: &http.Transport{
							DisableKeepAlives: true,
						},
					}

					randomParam := fmt.Sprintf("&nocache=%d", time.Now().UnixNano())
					body, headers, downloadErr = helpers.FetchReqClient(freshClient, regNo, cookies, downloadURL+randomParam, "", formData, "POST", "application/x-www-form-urlencoded")

					if downloadErr == nil && len(body) > 0 {
						if (isPotentialPptx && len(body) > 4096) || isSuccessfulDownload(body) {
							ext := helpers.GetFileExtension(fd.RefMat.Name, body, headers)
							if ext == "" {
								if isPotentialPptx {
									ext = ".pptx"
								} else {
									ext = ".bin"
								}
							}

							filePath := generateFilePath(fullDirPath, fd.IndexNo, fd.MNo, fd.TNo, fd.TopicName, fd.RefMaterialNo, ext)

							err = helpers.SaveFile(body, filePath)
							if err == nil {
								success = true
								successfulDownloads[key] = true
							} else {
								lastError = err.Error()
							}
						} else {
							lastError = "Invalid file content"
						}
					}
				}
			}

			if !success {
				permanentlyFailedDownloads = append(permanentlyFailedDownloads, FailedDownload{
					IndexNo:       fd.IndexNo,
					TopicName:     fd.TopicName,
					RefMaterialNo: fd.RefMaterialNo,
					RefMat:        fd.RefMat,
					Topic:         fd.Topic,
					Error:         lastError,
					MNo:           fd.MNo,
					TNo:           fd.TNo,
				})
			}

			retryBar.Add(1)

			if i < len(failedDownloads)-1 {
				time.Sleep(1 * time.Second)
			}
		}
		retryBar.Finish()
	}

	bar.Finish()

	totalFiles := totalRefMaterials
	successfulFiles := totalFiles - len(permanentlyFailedDownloads)

	fmt.Printf("\n\nDownload Summary:\n")
	fmt.Printf("Total files: %d\n", totalFiles)
	fmt.Printf("Successfully downloaded: %d\n", successfulFiles)

	if len(permanentlyFailedDownloads) > 0 {
		fmt.Printf("Failed to download: %d\n\n", len(permanentlyFailedDownloads))
		fmt.Println("The following files could not be downloaded:")

		for i, fd := range permanentlyFailedDownloads {
			fmt.Printf("%d. Topic: %s\n", i+1, fd.Topic)
			fmt.Printf("   File: %s\n", fd.RefMat.Name)
			fmt.Printf("   Error: %s\n", fd.Error)
			fmt.Println()
		}

		fmt.Println("\nYou can try downloading these files individually later.")
	} else {
		fmt.Println("All files were downloaded successfully!")
	}

	fmt.Printf("Files have been saved to: %s\n", fullDirPath)
	helpers.OpenFolder(fullDirPath)
	return nil
}

func generateFilePath(dirPath string, indexNo int, moduleNo, topicNo, topicName string, refMatNo int, ext string) string {
	var filename string

	if moduleNo != "" && topicNo != "" {
		topicContent := extractTopicContent(topicName)
		filename = fmt.Sprintf("M%s_T%s_%s_%d%s", moduleNo, topicNo, helpers.SanitizeFilename(topicContent), refMatNo, ext)
	} else {
		filename = fmt.Sprintf("%d_%s_%d%s", indexNo, topicName, refMatNo, ext)
	}

	filename = helpers.SanitizeFilename(filename)
	filePath := filepath.Join(dirPath, filename)

	// Handle long path names
	if len(filePath) > 250 {
		ext := filepath.Ext(filename)
		baseFilename := filename[:len(filename)-len(ext)]
		excessLength := len(filePath) - 250
		if excessLength >= len(baseFilename) {
			baseFilename = fmt.Sprintf("file_%d_%d", indexNo, refMatNo)
		} else {
			baseFilename = baseFilename[:len(baseFilename)-excessLength-1]
		}
		filename = baseFilename + ext
		filePath = filepath.Join(dirPath, filename)
	}

	return filePath
}

func isSuccessfulDownload(body []byte) bool {
	if len(body) < 4 {
		return false
	}

	signature := string(body[:4])
	switch signature {
	case "%PDF": // PDF
		return true
	case "PK\x03\x04": // ZIP-based formats (DOCX, XLSX, PPTX, etc.)
		if len(body) > 30 {
			readerAt := bytes.NewReader(body)
			size := int64(len(body))
			zipReader, err := zip.NewReader(readerAt, size)
			if err == nil {
				for _, f := range zipReader.File {
					if strings.HasPrefix(f.Name, "ppt/") ||
						strings.HasPrefix(f.Name, "word/") ||
						strings.HasPrefix(f.Name, "xl/") {
						return true
					}
				}
				return true
			}

			if len(body) > 4096 {
				return true
			}
		}
		return true
	case "\xD0\xCF\x11\xE0":
		return true
	case "PK\x05\x06", "PK\x07\x08":
		return false
	default:
		if len(body) >= 8 {
			if body[0] == 0xFF && body[1] == 0xD8 && body[2] == 0xFF {
				return true
			}
			if body[0] == 0x89 && body[1] == 0x50 && body[2] == 0x4E && body[3] == 0x47 {
				return true
			}
			if len(body) > 30 && bytes.Contains(body[:30], []byte("PK")) {
				return true
			}
		}

		if len(body) > 1024 {
			return true
		}

		return false
	}
}

func clearSingleNewline() {
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "cls").Run()
	}
}

// func getOptimalConcurrency() int {
// 	numCPU := runtime.NumCPU()
// 	if runtime.GOOS == "linux" {
// 		return numCPU * 2
// 	}
// 	return numCPU
// }

func getOptimizedConcurrency() int {
	return 4
}

func extractTopicContent(topic string) string {

	parts := strings.Split(topic, " - ")
	if len(parts) >= 3 {
		return parts[2]
	} else if len(parts) == 2 {
		return parts[1]
	}
	return topic
}
