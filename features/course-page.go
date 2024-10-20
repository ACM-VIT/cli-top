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
    "github.com/charmbracelet/glamour"
    "github.com/schollz/progressbar/v3"
)

type Semester struct {
    SemName string
    SemID   string
}

func ExecuteCoursePageDownload(regNo string, cookies types.Cookies, semesterFlag, courseFlag, facultyFlag int) {
    semDetails := helpers.GetSemDetails(cookies, regNo)
    if len(semDetails.SemIds) == 0 {
        fmt.Println("Error fetching semester details or no semesters available.")
        return
    }

    selectedSemId := helpers.SelectSemester(regNo, cookies, semesterFlag)
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

    selectedCourse, err := fetchAndSelectCourse(regNo, cookies, selectedSemester.SemID, courseFlag)
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

    selectedFaculty, err := helpers.SelectFaculty(faculties, facultyFlag)
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

    for i := range filteredMaterials {
        filteredMaterials[i].Index = i + 1
    }

    helpers.GenerateCourseMaterialsTable(filteredMaterials)

    selectedMaterials, err := helpers.SelectCourseMaterials(filteredMaterials)
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

    selectedCourse := types.Course{}
    if courseFlag > 0 && courseFlag <= len(courses) {
        selectedCourse = courses[courseFlag-1]
        if debug.Debug {
            fmt.Printf("Selected Course: %s (ID: %s)\n", selectedCourse.Name, selectedCourse.ID)
        }
    } else {
        helpers.GenerateCourseDetailsTable(courses)
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

        selectedCourse = courses[index-1]

        if debug.Debug {
            fmt.Printf("Selected Course: %s (ID: %s)\n", selectedCourse.Name, selectedCourse.ID)
        }
    }

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

    uniqueFaculties := helpers.RemoveDuplicateFaculties(allFaculties)

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

        slotInfo := strings.TrimSpace(cells.Eq(6).Text())
        slotInfo = strings.ReplaceAll(slotInfo, "┼", "+")
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
            Slot:     slotInfo,
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
        indexStr := strings.TrimSpace(cells.Eq(0).Text())
        index, _ := strconv.Atoi(indexStr)
        date := strings.TrimSpace(cells.Eq(1).Text())
        dayOrderSlot := strings.TrimSpace(cells.Eq(2).Text())
        dayOrderSlot = strings.ReplaceAll(dayOrderSlot, "┼", "+")
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
            Index:              index,
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
                    Index:              len(materials) + 1,
                    Date:               "",
                    DayOrderSlot:       "",
                    Topic:              category,
                    ReferenceMaterials: refMaterials,
                }
                materials = append(materials, material)
            }
        })
    })

    return materials, nil
}

func downloadMaterials(regNo string, cookies types.Cookies, selectedSemester Semester, selectedCourse types.Course, selectedFaculty types.Faculty, allMaterials []types.CourseMaterial, selectedMaterials []types.CourseMaterial) error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        if debug.Debug {
            fmt.Println("Error getting user's home directory:", err)
        }
        return err
    }

    downloadsDir := filepath.Join(homeDir, "Downloads", "Course Page Downloads")

    courseParts := helpers.SplitCourseNameFull(selectedCourse.Name)
    var courseFolderName string
    if len(courseParts) >= 3 {
        courseFolderName = fmt.Sprintf("%s_%s_%s", courseParts[0], courseParts[1], courseParts[2])
    } else if len(courseParts) == 2 {
        courseFolderName = fmt.Sprintf("%s_%s", courseParts[0], courseParts[1])
    } else {
        courseFolderName = courseParts[0]
    }
    courseFolderName = sanitizeFilename(courseFolderName)

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
    facultyNamePart := strings.ReplaceAll(facultyParts[0], " ", "-")
    if len(facultyParts) >= 2 {
        facultyFolderName = fmt.Sprintf("%s_%s_%s", slotID, facultyNamePart, facultyParts[1])
    } else {
        facultyFolderName = fmt.Sprintf("%s_%s", slotID, facultyNamePart)
    }
    facultyFolderName = sanitizeFilename(facultyFolderName)

    fullDirPath := filepath.Join(downloadsDir, courseFolderName, facultyFolderName)

    err = os.MkdirAll(fullDirPath, os.ModePerm)
    if err != nil {
        if debug.Debug {
            fmt.Println("Error creating directory:", err)
        }
        return err
    }

    if len(selectedMaterials) == 0 {
        return downloadSelectedMaterials(regNo, cookies, selectedFaculty, allMaterials, fullDirPath)
    } else {
        return downloadSelectedMaterials(regNo, cookies, selectedFaculty, selectedMaterials, fullDirPath)
    }
}

func downloadSelectedMaterials(regNo string, cookies types.Cookies, selectedFaculty types.Faculty, materials []types.CourseMaterial, fullDirPath string) error {
    totalFiles := 0
    for _, material := range materials {
        totalFiles += len(material.ReferenceMaterials)
    }

    bar := progressbar.Default(int64(totalFiles), "Downloading materials")

    for _, material := range materials {
        topicName := sanitizeFilename(material.Topic)
        for _, refMaterial := range material.ReferenceMaterials {
            downloadURL := "https://vtop.vit.ac.in/vtop/downloadPdf"
            payloadMap := map[string]string{
                "_csrf":        cookies.CSRF,
                "authorizedID": regNo,
                "semSubId":     selectedFaculty.SemSubID,
                "classId":      selectedFaculty.ClassID,
                "materialId":   refMaterial.MaterialID,
                "materialDate": refMaterial.MaterialDate,
            }
            formData := helpers.FormatBodyData(payloadMap)
            body, err := helpers.FetchReq(regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")
            if err != nil {
                if debug.Debug {
                    fmt.Println("Error downloading material:", err)
                }
                bar.Add(1)
                continue
            }
            if !isSuccessfulPdfDownload(body) {
                if debug.Debug {
                    fmt.Println("Failed to download material, response may indicate an error")
                }
                bar.Add(1)
                continue
            }
            refMaterialName := sanitizeFilename(refMaterial.Name)
            filename := fmt.Sprintf("%s_%s.pdf", topicName, refMaterialName)
            filePath := filepath.Join(fullDirPath, filename)
            // Check for duplicate files
            if _, err := os.Stat(filePath); os.IsNotExist(err) {
                err = saveFile(body, filePath)
                if err != nil {
                    if debug.Debug {
                        fmt.Println("Error saving file:", err)
                    }
                    bar.Add(1)
                    continue
                }
            }
            bar.Add(1)
        }
    }
    fmt.Println("\nCourse materials downloaded successfully")
    fmt.Printf("\033[34m\033[4m\033]8;;file://%s\033\\%s\033]8;;\033\\\033[0m to open the folder.\n", fullDirPath, "Click Here")
    return nil
}

func isSuccessfulPdfDownload(body []byte) bool {
    return strings.HasPrefix(string(body), "%PDF-")
}

func sanitizeFilename(name string) string {
    replacer := strings.NewReplacer(
        "/", "_",
        "\\", "_",
        ":", "",
        "*", "_",
        "?", "",
        "\"", "_",
        "<", "_",
        ">", "_",
        "|", "_",
        "\u2013", "-",
        "\u2014", "-",
        "\u2018", "'",
        "\u2019", "'",
        "\u201C", "\"",
        "\u201D", "\"",
    )
    return replacer.Replace(name)
}

func saveFile(data []byte, filePath string) error {
    err := os.WriteFile(filePath, data, 0644)
    if err != nil {
        return err
    }
    return nil
}
