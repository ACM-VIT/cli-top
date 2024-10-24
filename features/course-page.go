package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/schollz/progressbar/v3"
)

func ExecuteCoursePageDownload(regNo string, cookies types.Cookies, semesterFlag, courseFlag, facultyFlag int) {
    selectedSemester, err := helpers.SelectSemester(regNo, cookies, semesterFlag)
    if err != nil && debug.Debug {
        fmt.Println(err)
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

    var filteredMaterials []types.CourseMaterial
    for _, material := range materials {
        if len(material.ReferenceMaterials) > 0 {
            filteredMaterials = append(filteredMaterials, material)
        }
    }

    if len(filteredMaterials) == 0 {
        fmt.Println("No course materials with reference materials available for download.")
        return
    }

    displayCourseMaterials(filteredMaterials)

    selectedMaterials, err := selectCourseMaterials(filteredMaterials)
    if err != nil {
        fmt.Println("Error selecting materials:", err)
        return
    }

    err = downloadMaterials(regNo, cookies, selectedSemester, selectedCourse, selectedFaculty, filteredMaterials, selectedMaterials)
    if err != nil {
        fmt.Println("Error downloading materials:", err)
        return
    }
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

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, getCourseURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return types.Course{}, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
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
		return types.Course{}, fmt.Errorf("no courses found for the selected semester")
	}

	nestedList := [][]string{{"COURSE NAME"}}
	for _, course := range courses {
		nestedList = append(nestedList, []string{course.Name})
	}

	selectedIndex := helpers.TableSelector("Course", nestedList, courseFlag)
	if selectedIndex == -1 || selectedIndex < 1 || selectedIndex > len(courses) {
		return types.Course{}, fmt.Errorf("invalid course selection")
	}

	selectedCourse := courses[selectedIndex-1]
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

	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, getSlotURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
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
		return nil, fmt.Errorf("no slots found for the selected course")
	}

	return slots, nil
}

func fetchFacultiesForAllSlotsConcurrently(regNo string, cookies types.Cookies, semSubId string, classId string, slotIds []string) ([]types.Faculty, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	facultySlices := make([][]types.Faculty, len(slotIds))
	errorsOccurred := false
	var mu sync.Mutex

	for i, slotId := range slotIds {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, slotId string) {
			defer wg.Done()
			defer func() { <-sem }()
			faculties, err := fetchFaculties(regNo, cookies, semSubId, classId, slotId)
			if err != nil {
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
		return nil, err
	}

	if strings.Contains(string(body), "HTTP Status 404") {
		return nil, fmt.Errorf("received 404 Not Found when fetching faculty details")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`processViewStudentCourseDetail\(['"]([^'"]+)['"],\s*['"]([^'"]+)['"],\s*['"]([^'"]+)['"]\)`)

	var faculties []types.Faculty

	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
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

func selectFaculty(faculties []types.Faculty, facultyFlag int) (types.Faculty, error) {
    for {
        fmt.Print("")

        input := ""

        if input == "" {
            nestedList := [][]string{{"SLOT", "NAME"}}
            for _, faculty := range faculties {
                cleanName := removeNumberPrefix(faculty.Name)
                nestedList = append(nestedList, []string{
                    faculty.Slot,
                    cleanName,
                })
            }

            selectedIndex := helpers.TableSelector("Faculty", nestedList, facultyFlag)
            if selectedIndex == -1 || selectedIndex < 1 || selectedIndex > len(faculties) {
                return types.Faculty{}, fmt.Errorf("invalid faculty selection")
            }

            return faculties[selectedIndex-1], nil
        }

        if index, err := strconv.Atoi(input); err == nil {
            if index >= 1 && index <= len(faculties) {
                return faculties[index-1], nil
            } else {
                fmt.Println("Invalid index number. Please enter a valid index.")
                continue
            }
        }

        matchingResults := [][]string{}
        for i, faculty := range faculties {
            cleanName := removeNumberPrefix(faculty.Name)
            if helpers.FuzzyMatch(input, cleanName) {
                matchingResults = append(matchingResults, []string{
                    fmt.Sprintf("%d", i+1), 
                    faculty.Slot,
                    cleanName,
                })
            }
        }

        if len(matchingResults) == 0 {
            fmt.Println("No matching faculty found for your query. Please try again.")
            continue
        }

        if len(matchingResults) == 1 {
            selectedIndex, _ := strconv.Atoi(matchingResults[0][0])
            return faculties[selectedIndex-1], nil
        }

        fmt.Println("\nMultiple matches found. Please select an index from the results below:")
        nestedList := append([][]string{{"INDEX", "SLOT", "NAME"}}, matchingResults...)

        selectedIndex := helpers.TableSelector("Faculty", nestedList, facultyFlag)
        if selectedIndex == -1 || selectedIndex < 1 || selectedIndex > len(faculties) {
            fmt.Println("Invalid selection. Please enter a valid index number.")
            continue
        }

        _, err := fmt.Scanln(&input) 
        if err != nil {
            if err.Error() == "unexpected newline" {
                input = "" 
            } else {
                if debug.Debug {
                    fmt.Println("Error reading input:", err)
                }
                return types.Faculty{}, err
            }
        }

        input = strings.TrimSpace(input)

        if input == "exit" {
            return types.Faculty{}, fmt.Errorf("selection canceled by user")
        }

        return faculties[selectedIndex-1], nil
    }
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
	formData := helpers.FormatBodyData(payloadMap)
	body, err := helpers.FetchReq(regNo, cookies, url, "", formData, "POST", "application/x-www-form-urlencoded")
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

	doc.Find("table.table-bordered.table-hover tbody tr").Each(func(_ int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() < 5 {
			return
		}
		date := strings.TrimSpace(cells.Eq(1).Text())
		dayOrderSlot := strings.TrimSpace(cells.Eq(2).Text())
		dayOrderSlot = helpers.ReplaceCrossWithPlus(dayOrderSlot)
		topic := strings.TrimSpace(cells.Eq(3).Text())
		if topic == "" {
			topic = "Unnamed"
		}
		refMaterialsTd := cells.Eq(4)

		var refMaterials []types.ReferenceMaterial
		refMaterialsTd.Find("button[name='getDownloadSemPdf']").Each(func(_ int, btn *goquery.Selection) {
			materialID, _ := btn.Attr("data-matid")
			materialDate, _ := btn.Attr("data-mdate")
			name := strings.TrimSpace(btn.Find("span").Text())
			refMaterials = append(refMaterials, types.ReferenceMaterial{
				Name:         name,
				MaterialID:   materialID,
				MaterialDate: materialDate,
			})
		})

		materials = append(materials, types.CourseMaterial{
			Date:               date,
			DayOrderSlot:       dayOrderSlot,
			Topic:              topic,
			ReferenceMaterials: refMaterials,
		})
	})

	doc.Find("table").Each(func(_ int, table *goquery.Selection) {
		if table.HasClass("table-bordered") && table.HasClass("table-hover") {
			return
		}

		table.Find("tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}
			category := strings.TrimSpace(cells.Eq(0).Text())
			buttonsTd := cells.Eq(1)
			var refMaterials []types.ReferenceMaterial
			buttonsTd.Find("button[name='getDownloadSemPdf']").Each(func(_ int, btn *goquery.Selection) {
				materialID, _ := btn.Attr("data-matid")
				materialDate, _ := btn.Attr("data-mdate")
				name := strings.TrimSpace(btn.Find("span").Text())
				refMaterials = append(refMaterials, types.ReferenceMaterial{
					Name:         name,
					MaterialID:   materialID,
					MaterialDate: materialDate,
				})
			})
			if len(refMaterials) > 0 {
				material := types.CourseMaterial{
					Topic:              category,
					ReferenceMaterials: refMaterials,
				}
				materials = append(materials, material)
			}
		})
	})

	return materials, nil
}

func displayCourseMaterials(materials []types.CourseMaterial) {
	nestedList := [][]string{{"DATE", "DAY ORDER/SLOT", "TOPIC", "REF MATERIALS"}}
	for _, material := range materials {
		refCount := strconv.Itoa(len(material.ReferenceMaterials))
		nestedList = append(nestedList, []string{
			material.Date,
			material.DayOrderSlot,
			material.Topic,
			refCount,
		})
	}
	helpers.PrintTable(nestedList, 1)
}

func selectCourseMaterials(materials []types.CourseMaterial) ([]types.CourseMaterial, error) {
	for {
		fmt.Print("Enter the index numbers of the topics to download (e.g., 1,2,3), or 0 for bulk download: ")

		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			if err.Error() == "unexpected newline" {
				fmt.Println("No input provided.")
				continue
			}
			if debug.Debug {
				fmt.Println("Error reading input:", err)
			}
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

		indicesStr := strings.Split(input, ",")
		indexSet := make(map[int]struct{})
		var invalidIndices []string

		for _, idxStr := range indicesStr {
			idxStr = strings.TrimSpace(idxStr)
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				invalidIndices = append(invalidIndices, idxStr)
				continue
			}
			if idx < 1 || idx > len(materials) {
				invalidIndices = append(invalidIndices, idxStr)
				continue
			}
			indexSet[idx] = struct{}{}
		}

		if len(invalidIndices) > 0 {
			fmt.Println("Invalid indices:", strings.Join(invalidIndices, ", "))
		}

		if len(indexSet) == 0 {
			fmt.Println("No valid indices selected.")
			continue
		}

		var selectedMaterials []types.CourseMaterial
		for idx := range indexSet {
			selectedMaterials = append(selectedMaterials, materials[idx-1])
		}

		return selectedMaterials, nil
	}
}


func downloadMaterials(regNo string, cookies types.Cookies, selectedSemester types.Semester, selectedCourse types.Course, selectedFaculty types.Faculty, allMaterials []types.CourseMaterial, selectedMaterials []types.CourseMaterial) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	downloadsDir := filepath.Join(homeDir, "Downloads", "Course Page Downloads")

	courseParts := helpers.SplitCourseNameFull(selectedCourse.Name)
	var courseFolderName string
	if len(courseParts) >= 3 {
		courseFolderName = fmt.Sprintf("%s_%s_%s", courseParts[0], courseParts[1], courseParts[2])
	} else if len(courseParts) == 2 {
		courseFolderName = fmt.Sprintf("%s_%s", courseParts[0], courseParts[1])
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

	fullDirPath := filepath.Join(downloadsDir, courseFolderName, facultyFolderName)

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

	return downloadSelectedMaterials(regNo, cookies, selectedFaculty, selectedMaterials, fullDirPath)
}

func downloadSelectedMaterials(
	regNo string,
	cookies types.Cookies,
	selectedFaculty types.Faculty,
	materials []types.CourseMaterial,
	fullDirPath string,
) error {
	totalFiles := 0
	for _, material := range materials {
		totalFiles += len(material.ReferenceMaterials)
	}

	if totalFiles == 0 {
		fmt.Println("No files to download.")
		return nil
	}

	bar := progressbar.NewOptions(totalFiles,
		progressbar.OptionSetDescription("Downloading materials..."),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowIts(),
		progressbar.OptionClearOnFinish(),
	)

	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, 5)
	counter := 1

	for _, material := range materials {
		topicName := helpers.SanitizeFilename(material.Topic)

		for _, refMaterial := range material.ReferenceMaterials {
			sem <- struct{}{}
			wg.Add(1)

			go func(material types.CourseMaterial, refMaterial types.ReferenceMaterial, counter int) {
				defer wg.Done()
				defer func() { <-sem }()
				downloadURL := "https://vtop.vit.ac.in/vtop/downloadPdf"
				payloadMap := map[string]string{
					"_csrf":        cookies.CSRF,
					"authorizedID": regNo,
					"semSubId":     selectedFaculty.SemSubID,
					"classId":      selectedFaculty.ClassID,
					"materialId":   refMaterial.MaterialID,
					"materialDate": refMaterial.MaterialDate,
					"x":            time.Now().UTC().Format(time.RFC1123),
				}
				formData := helpers.FormatBodyData(payloadMap)

				body, err := helpers.FetchReq(regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")
				if err != nil {
					mu.Lock()
					bar.Add(1)
					mu.Unlock()
					return
				}

				if !isSuccessfulDownload(body) {
					mu.Lock()
					bar.Add(1)
					mu.Unlock()
					return
				}

				refMaterialName := helpers.SanitizeFilename(refMaterial.Name)
				ext := getFileExtension(refMaterialName, body)
				filename := fmt.Sprintf("%02d_%s_%s%s", counter, topicName, refMaterialName, ext)
				filePath := filepath.Join(fullDirPath, filename)

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					err = helpers.SaveFile(body, filePath)
					if err != nil {
						mu.Lock()
						bar.Add(1)
						mu.Unlock()
						return
					}
				}

				mu.Lock()
				bar.Add(1)
				mu.Unlock()
			}(material, refMaterial, counter)
			counter++
		}
	}

	wg.Wait()
	bar.Finish()
	fmt.Println("\nCourse materials downloaded successfully")
	fmt.Printf("\033[34m\033[4m\033]8;;file://%s\033\\%s\033]8;;\033\\\033[0m to open the folder.\n", fullDirPath, "Click Here")
	return nil
}

func isSuccessfulDownload(body []byte) bool {
	if len(body) < 4 {
		return false
	}
	signature := string(body[:4])
	switch signature {
	case "%PDF":
		return true
	case "PK\x03\x04":
		return true
	case "PK\x05\x06":
		return false
	case "PK\x07\x08":
		return false
	default:
		return false
	}
}

func getFileExtension(filename string, body []byte) string {
	ext := filepath.Ext(filename)
	if ext != "" {
		return ext
	}
	if len(body) >= 4 {
		signature := string(body[:4])
		switch signature {
		case "%PDF":
			return ".pdf"
		case "PK\x03\x04":
			lowerName := strings.ToLower(filename)
			if strings.Contains(lowerName, "docx") {
				return ".docx"
			} else if strings.Contains(lowerName, "pptx") {
				return ".pptx"
			} else if strings.Contains(lowerName, "xlsx") {
				return ".xlsx"
			}
			return ".pptx"
		default:
			return ""
		}
	}
	return ""
}
