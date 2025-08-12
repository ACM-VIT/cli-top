package features

import (
	"archive/zip"
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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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

	selectedCourse, err := fetchAndSelectCourse(regNo, cookies, courseFlag)
	if err != nil {
		fmt.Println("Error selecting course:", err)
		return
	}

	materials, faculty, err := fetchFacultieswithMaterials(regNo, cookies, selectedCourse.ID, selectedCourse.Name, facultyFlag)
	if err != nil {
		fmt.Println("Error fetching faculties:", err)
		return
	}

	displayCourseMaterials(materials)

	selectedMaterials, err := selectCourseMaterials(materials)
	if err != nil {
		fmt.Println("Error selecting materials:", err)
		return
	}

	err = downloadMaterialsHope(regNo, cookies, selectedCourse, materials, selectedMaterials, faculty)
	if err != nil {
		fmt.Printf("Error downloading materials: %v\n", err)
		return
	}

	fmt.Println("\nDownload complete!")
}

func fetchAndSelectCourse(regNo string, cookies types.Cookies, courseFlag int) (types.Course, error) {
	getCourseURL := "https://vtop.vit.ac.in/vtop/academics/common/CoursePageConsolidated"
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"authorizedID": regNo,
		"x":            time.Now().UTC().Format(time.RFC1123),
		"verifyMenu":   "true",
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

	type courseWithSem struct {
		Semester string
		Course   types.Course
	}

	var courses []courseWithSem
	semSet := make(map[string]struct{})
	doc.Find("select#courseId option").Each(func(_ int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists && value != "" {
			text := strings.TrimSpace(s.Text())
			parts := strings.SplitN(text, " - ", 2)
			semester := ""
			if len(parts) > 1 {
				semester = strings.TrimSpace(parts[0])
			}
			if semester == "" {
				semester = "Unknown Semester"
			}
			semSet[semester] = struct{}{}
			courses = append(courses, courseWithSem{
				Semester: semester,
				Course:   types.Course{ID: value, Name: text},
			})
		}
	})
	// // Merge semester names from semDetails into semSet, but remove duplicates by normalizing.
	// semDetails, err := helpers.GetSemDetails(cookies, regNo)
	// if err == nil && len(semDetails) > 0 {
	// 	for _, sem := range semDetails {
	// 		// Normalize by removing trailing " - VLR" if present
	// 		normalized := strings.TrimSuffix(sem.SemName, " - VLR")
	// 		semSet[normalized] = struct{}{}
	// 	}
	// }
	// if err != nil {
	// 	return types.Course{}, err
	// }
	// // Also normalize keys already in semSet (from goquery)
	// normalizedSemSet := make(map[string]struct{})
	// for sem := range semSet {
	// 	normalized := strings.TrimSuffix(sem, " - VLR")
	// 	normalizedSemSet[normalized] = struct{}{}
	// }
	// semSet = normalizedSemSet

	type semInfo struct {
		Raw    string
		Year   int
		Season int // Fall=0, Winter=1
	}
	var semInfos []semInfo
	for sem := range semSet {
		year := 0
		season := 1
		if strings.HasPrefix(sem, "Fall") {
			season = 0
		}
		// Extract year (e.g., "Fall Semester 2025-26")
		yearParts := strings.Fields(sem)
		if len(yearParts) >= 3 {
			yearStr := yearParts[2]
			yearStr = strings.Split(yearStr, "-")[0]
			if y, err := strconv.Atoi(yearStr); err == nil {
				year = y
			}
		}
		semInfos = append(semInfos, semInfo{Raw: sem, Year: year, Season: season})
	}
	sort.Slice(semInfos, func(i, j int) bool {
		if semInfos[i].Year != semInfos[j].Year {
			return semInfos[i].Year < semInfos[j].Year
		}
		return semInfos[i].Season < semInfos[j].Season
	})

	var semesters []string
	for _, si := range semInfos {
		semesters = append(semesters, si.Raw)
	}

	// fallSemester := "Fall Semester 2025-26"
	// fallIdx := -1
	// i := 0
	// for _, sem := range semesters {
	// 	if sem == fallSemester {
	// 		fallIdx = i
	// 		break
	// 	}
	// 	i++
	// }

	// Ask user to select semester

	var selectedSemester string

	if len(semesters) > 1 {
		semTable := [][]string{{"SEMESTER"}}
		for _, sem := range semesters {
			semTable = append(semTable, []string{sem})
		}
		semResult := helpers.TableSelector("Semester", semTable, "")
		if semResult.ExitRequest {
			return types.Course{}, fmt.Errorf("selection canceled by user")
		}
		if !semResult.Selected || semResult.Index < 1 || semResult.Index > len(semesters) {
			return types.Course{}, fmt.Errorf("invalid semester selection")
		}
		selectedSemester = semesters[semResult.Index-1]
	} else {
		selectedSemester = semesters[0]
	}

	// if fallIdx != -1 {
	// 	 if semResult.Index-1 < fallIdx {
	// 		coursePageOldAfterSemSelection(regNo, cookies, selectedSemester, courseFlag, facultyFlag)
	// 	 }
	// }

	var filteredCourses []types.Course
	for _, c := range courses {
		if c.Semester == selectedSemester {
			filteredCourses = append(filteredCourses, c.Course)
		}
	}
	if len(filteredCourses) == 0 {
		return types.Course{}, fmt.Errorf("no courses found for selected semester")
	}

	// Display filtered courses for selection (only code and name)
	courseTable := [][]string{{"COURSE CODE", "COURSE NAME"}}
	for _, course := range filteredCourses {
		parts := strings.Split(course.Name, " - ")
		code := ""
		name := ""
		if len(parts) >= 3 {
			code = strings.TrimSpace(parts[1])
			name = strings.TrimSpace(parts[2])
		} else if len(parts) == 2 {
			code = strings.TrimSpace(parts[1])
			name = ""
		} else {
			code = ""
			name = course.Name
		}
		courseTable = append(courseTable, []string{code, name})
	}
	courseResult := helpers.TableSelector("Course", courseTable, strconv.Itoa(courseFlag))
	if courseResult.ExitRequest {
		return types.Course{}, fmt.Errorf("selection canceled by user")
	}
	if !courseResult.Selected || courseResult.Index < 1 || courseResult.Index > len(filteredCourses) {
		return types.Course{}, fmt.Errorf("invalid course selection")
	}
	selectedCourse := filteredCourses[courseResult.Index-1]
	return selectedCourse, nil
}

func fetchFacultieswithMaterials(regNo string, cookies types.Cookies, courseID string, courseName string, facultyFlag string) ([]types.CourseMaterial, types.Faculty, error) {
	getFacultyMaterialURL := "https://vtop.vit.ac.in/vtop/academics/CoursePageConsolidated/getCourseDetail"
	// Extract course type from courseName (assumed format: "Semester - CourseCode - CourseTitle - ...")
	parts := strings.Split(courseName, " - ")
	if len(parts) < 3 {
		return nil, types.Faculty{}, fmt.Errorf("invalid course name format")
	}
	courseType := strings.TrimSpace(parts[len(parts)-3])
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"CourseId":     courseID,
		"CoursType":    courseType,
		"authorizedID": regNo,
		"x":            time.Now().UTC().Format(time.RFC1123),
	}
	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(newHttpClient, regNo, cookies, getFacultyMaterialURL, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, types.Faculty{}, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, types.Faculty{}, err
	}

	// Temporary struct to hold faculty name (raw) and the material
	type facultyMaterial struct {
		Faculty  string
		Material types.CourseMaterial
	}
	var allMaterials []facultyMaterial
	facultySet := make(map[string]struct{})

	rows := doc.Find("table#materialTable tbody tr")
	rows.Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 5 {
			return
		}

		// Extract faculty info from cell index 3, inside its div.mt-1
		facultyDiv := cells.Eq(3).Find("div.mt-1")
		facultySpans := facultyDiv.Find("span")
		if facultySpans.Length() < 2 {
			return
		}
		rawFaculty := strings.TrimSpace(facultySpans.Eq(0).Text())
		date := strings.TrimSpace(facultySpans.Eq(1).Text())
		if rawFaculty == "" {
			return
		}
		facultySet[rawFaculty] = struct{}{}

		// Get the material info
		indexStr := strings.TrimSpace(cells.Eq(0).Text())
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			index = i + 1
		}
		// Topic from cell 2: try the nested span inside div.mt-1
		materialDiv := cells.Eq(2).Find("div.mt-1")
		var topic string
		var mNo string
		if materialDiv.Length() > 0 {
			// Try to find the span with style containing "#2E86C1"
			materialDiv.Find("span").Each(func(i int, s *goquery.Selection) {
				if style, exists := s.Attr("style"); exists && strings.Contains(style, "#2E86C1") {
					topic = strings.TrimSpace(s.Text())
					return
				}
			})
			// Fallback if no span with the color was found
			if topic == "" {
				topic = strings.TrimSpace(materialDiv.Find("span").First().Text())
			}
			// Extract MNo from span with style containing "#28B463"
			materialDiv.Find("span").Each(func(i int, s *goquery.Selection) {
				if style, exists := s.Attr("style"); exists && strings.Contains(style, "#28B463") {
					mNo = strings.TrimSpace(s.Text())
				}
			})
		} else {
			topic = strings.TrimSpace(cells.Eq(2).Text())
			mNo = ""
		}

		// Download button is in cell 4
		downloadBtn := cells.Eq(4).Find("button[name='downloadmat']")
		materialID, _ := downloadBtn.Attr("data-fileid")
		material := types.CourseMaterial{
			Index: index,
			Date:  date,
			Topic: topic,
			ReferenceMaterials: []types.ReferenceMaterial{
				{
					Name:       topic,
					MaterialID: materialID,
				},
			},
			MNo: mNo,
		}
		allMaterials = append(allMaterials, facultyMaterial{
			Faculty:  rawFaculty,
			Material: material,
		})
	})

	// Build unique faculty list from facultySet.
	// We assume each raw faculty is in the format "ERPID - Faculty Name - SCOPE"
	var facultyList []types.Faculty
	for raw := range facultySet {
		parts := strings.Split(raw, " - ")
		if len(parts) < 3 {
			continue
		}
		erpID := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		facultyItem := types.Faculty{
			Name:         name,
			ErpID:        erpID,
			SemesterName: courseName, // using courseName as provided
			CourseName:   courseName,
		}
		facultyList = append(facultyList, facultyItem)
	}
	sort.Slice(facultyList, func(i, j int) bool {
		return facultyList[i].Name < facultyList[j].Name
	})

	// If nothing uploaded for this course, error out explicitly
	if len(facultyList) == 0 {
		return nil, types.Faculty{}, fmt.Errorf("no material uploaded")
	}

	// Prompt the user to select a faculty
	nestedList := [][]string{{"FACULTY"}}
	for _, f := range facultyList {
		nestedList = append(nestedList, []string{f.Name})
	}
	result := helpers.TableSelectorFuzzy("Faculty", nestedList, facultyFlag, helpers.NewFuzzySearch)
	if result.ExitRequest {
		return nil, types.Faculty{}, fmt.Errorf("selection canceled by user")
	}
	if !result.Selected || result.Index < 1 || result.Index > len(facultyList) {
		return nil, types.Faculty{}, fmt.Errorf("invalid faculty selection")
	}
	selectedFaculty := facultyList[result.Index-1]

	// Filter materials for the selected faculty.
	var materials []types.CourseMaterial
	for _, fm := range allMaterials {
		// Parse the raw faculty string to extract the name.
		fParts := strings.Split(fm.Faculty, " - ")
		if len(fParts) < 2 {
			continue
		}
		fName := strings.TrimSpace(fParts[1])
		if fName == selectedFaculty.Name {
			materials = append(materials, fm.Material)
		}
	}
	if len(materials) == 0 {
		// No rows matched selected faculty – treat as no uploads
		return nil, types.Faculty{}, fmt.Errorf("no material uploaded")
	}
	return materials, selectedFaculty, nil
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
		header = []string{"DATE", "MODULE", "TOPIC", "REF COUNT", "WEB MATERIAL"}
	} else {
		header = []string{"DATE", "MODULE", "TOPIC", "REF COUNT"}
	}

	// Increase topic column width from 30 to 50
	nestedList := [][]string{header}
	for _, material := range materials {
		refCount := strconv.Itoa(len(material.ReferenceMaterials))
		topic := helpers.TruncateWithEllipsis(material.Topic, 50)
		module := material.MNo
		if showWebColumn {
			webCol := ""
			if strings.TrimSpace(material.WebLink) != "" {
				webCol = helpers.MakeANSILink("Open", material.WebLink)
			}
			nestedList = append(nestedList, []string{
				material.Date,
				module,
				topic,
				refCount,
				webCol,
			})
		} else {
			nestedList = append(nestedList, []string{
				material.Date,
				module,
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

func downloadMaterialsHope(regNo string, cookies types.Cookies, selectedCourse types.Course, allMaterials []types.CourseMaterial, selectedMaterials []types.CourseMaterial, faculty types.Faculty) error {
	// Concurrency primitives
	var wg sync.WaitGroup
	errChan := make(chan error, 1024) // large enough buffer for errors

	coursePageDir, err := helpers.GetOrCreateDownloadDir("Course Page")
	if err != nil {
		return fmt.Errorf("failed to create course page directory: %w", err)
	}

	// Build folder structure (UNCHANGED)
	courseParts := helpers.SplitCourseNameFull(selectedCourse.Name)
	semesterName := fmt.Sprintf("%s_%s", courseParts[0], courseParts[1])
	courseName := courseParts[3]
	fullDirPath := filepath.Join(coursePageDir, semesterName, courseName, faculty.Name)

	if mkErr := os.MkdirAll(fullDirPath, os.ModePerm); mkErr != nil {
		return mkErr
	}

	if selectedMaterials == nil {
		selectedMaterials = allMaterials
	}

	if helpers.IsRateLimitExceeded() {
		fmt.Println("Rate limit exceeded. Please try again later.")
		return fmt.Errorf("rate limit exceeded")
	}

	if _, statErr := os.Stat(fullDirPath); os.IsNotExist(statErr) {
		return fmt.Errorf("download directory does not exist: %s", fullDirPath)
	}

	// Count total reference materials for progress bar
	totalRefMaterials := 0
	for _, material := range selectedMaterials {
		totalRefMaterials += len(material.ReferenceMaterials)
	}

	bar := progressbar.NewOptions(
		totalRefMaterials,
		progressbar.OptionSetDescription("Downloading materials..."),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionClearOnFinish(),
	)
	var barMu sync.Mutex
	addProgress := func() {
		barMu.Lock()
		_ = bar.Add(1)
		barMu.Unlock()
	}

	// Tunables
	concurrency := 4 // allow a bit more concurrency while being gentle on server
	sem := make(chan struct{}, concurrency)

	// Helper to decide if a file is likely large/needs longer timeout
	isLargeCandidate := func(name string) bool {
		low := strings.ToLower(name)
		return strings.Contains(low, "ppt") ||
			strings.Contains(low, "presentation") ||
			strings.Contains(low, "slide") ||
			strings.HasSuffix(low, ".pptx") ||
			strings.HasSuffix(low, ".ppt") ||
			strings.HasSuffix(low, ".zip") ||
			strings.HasSuffix(low, ".mp4") ||
			strings.HasSuffix(low, ".mkv")
	}

	// Launch per-reference-material goroutines (better throughput; path remains unchanged)
	for _, mat := range selectedMaterials {
		material := mat // capture
		for _, rm := range material.ReferenceMaterials {
			refMat := rm // capture

			sem <- struct{}{}
			wg.Add(1)

			go func(material types.CourseMaterial, refMat types.ReferenceMaterial) {
				defer wg.Done()
				defer func() { <-sem }()
				defer addProgress()

				// Endpoint & payload (UNCHANGED)
				downloadURL := "https://vtop.vit.ac.in/vtop/downloadCourseMaterialFacultyPdf"
				payloadMap := map[string]string{
					"_csrf":        cookies.CSRF,
					"authorizedID": regNo,
					"fileId":       refMat.MaterialID,
				}
				formData := helpers.FormatBodyDataClient(payloadMap)

				// Retry logic with exponential backoff + jitter + last long attempt
				maxAttempts := 5
				large := isLargeCandidate(refMat.Name)

				var body []byte
				var headers http.Header
				var lastErr error
				var success bool

				for attempt := 1; attempt <= maxAttempts; attempt++ {
					timeout := 2 * time.Minute
					if large {
						timeout = 5 * time.Minute
					}

					client := &http.Client{
						Timeout: timeout,
						Transport: &http.Transport{
							MaxIdleConns:        10,
							MaxIdleConnsPerHost: 5,
							IdleConnTimeout:     30 * time.Second,
							DisableKeepAlives:   true,
						},
					}

					var fetchErr error
					body, headers, fetchErr = helpers.FetchReqClient(client, regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")
					if fetchErr != nil {
						lastErr = fetchErr
					} else if len(body) == 0 {
						lastErr = fmt.Errorf("empty response")
					} else {
						// Validate content
						if (large && len(body) > 4096) || isSuccessfulDownload(body) {
							success = true
							break
						}
						lastErr = fmt.Errorf("invalid file content")
					}

					backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
					jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
					time.Sleep(backoff + jitter)
				}

				// Final "fresh client" attempt with longer timeout & cache buster
				if !success {
					timeout := 5 * time.Minute
					if large {
						timeout = 10 * time.Minute
					}
					freshClient := &http.Client{
						Timeout: timeout,
						Transport: &http.Transport{
							DisableKeepAlives: true,
						},
					}
					randomParam := fmt.Sprintf("?nocache=%d", time.Now().UnixNano())
					b2, h2, fetchErr := helpers.FetchReqClient(
						freshClient, regNo, cookies, downloadURL+randomParam, "",
						formData, "POST", "application/x-www-form-urlencoded",
					)
					if fetchErr == nil && len(b2) > 0 && ((large && len(b2) > 4096) || isSuccessfulDownload(b2)) {
						body, headers = b2, h2
						success = true
					} else if fetchErr != nil {
						lastErr = fetchErr
					} else {
						lastErr = fmt.Errorf("final attempt received invalid/empty content")
					}
				}

				if !success {
					errChan <- fmt.Errorf("failed to download %q (fileId=%s): %v", material.Topic, refMat.MaterialID, lastErr)
					return
				}

				// Build safe filename:
				// 1) Treat only allowed extensions as real extensions.
				// 2) Fix trailing numeric dotted suffixes (e.g. "1.1" -> "1-1").
				// 3) Ensure final extension is one of allowed Office/PDF types.
				baseName := helpers.SanitizeFilename(refMat.Name)
				currExt := strings.ToLower(filepath.Ext(baseName))

				detected := strings.ToLower(helpers.GetFileExtension(refMat.Name, body, headers))
				if detected == "" {
					detected = inferExtFromBody(body)
				}
				finalExt := pickAllowedExt(currExt, detected, body)
				if !strings.HasPrefix(finalExt, ".") {
					finalExt = "." + finalExt
				}

				// Only strip current extension if it is an allowed one; otherwise keep it in the base name.
				nameNoExt := baseName
				if _, ok := allowedOfficeExt[currExt]; ok {
					nameNoExt = strings.TrimSuffix(baseName, currExt)
				}

				// Convert trailing dotted numeric suffix to hyphenated form (e.g., "Topic 1.2" -> "Topic 1-2")
				nameNoExt = fixNumericSuffix(strings.TrimSpace(nameNoExt))

				// Prepend module number if available
				modulePrefix := ""
				if strings.TrimSpace(material.MNo) != "" {
					modulePrefix = "Module-" + material.MNo + "_"
				}
				baseFileName := modulePrefix + nameNoExt + finalExt
				filePath := filepath.Join(fullDirPath, baseFileName)

				// Ensure no overwrite: if file exists, append _1, _2, etc.
				uniqueFilePath := filePath
				if _, err := os.Stat(uniqueFilePath); err == nil {
					// File exists, find a unique name
					for suffix := 1; ; suffix++ {
						altName := fmt.Sprintf("%s_%d%s", modulePrefix+nameNoExt, suffix, finalExt)
						uniqueFilePath = filepath.Join(fullDirPath, altName)
						if _, err := os.Stat(uniqueFilePath); os.IsNotExist(err) {
							break
						}
					}
				}

				if saveErr := helpers.SaveFile(body, uniqueFilePath); saveErr != nil {
					errChan <- fmt.Errorf("error saving %q to %s: %v", material.Topic, uniqueFilePath, saveErr)
					return
				}
			}(material, refMat)
		}
	}

	// Waiter for all downloads
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for e := range errChan {
		fmt.Println("Error:", e)
	}

	helpers.OpenFolder(fullDirPath)

	return nil
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

// Allowed extensions: only PDF and Office formats
var allowedOfficeExt = map[string]struct {
}{

	".pdf":  {},
	".pptx": {},
	".ppt":  {},
	".docx": {},
	".doc":  {},
	".xlsx": {},
	".xls":  {},
}

// pickAllowedExt chooses a safe extension from detected or current, restricted to allowedOfficeExt.
// Falls back to inferring from body, then .pdf.
func pickAllowedExt(currExt, detected string, body []byte) string {
	currExt = strings.ToLower(currExt)
	detected = strings.ToLower(detected)
	if _, ok := allowedOfficeExt[detected]; ok {
		return detected
	}
	if _, ok := allowedOfficeExt[currExt]; ok {
		return currExt
	}
	inf := strings.ToLower(inferExtFromBody(body))
	if _, ok := allowedOfficeExt[inf]; ok {
		return inf
	}
	return ".pdf"
}

// inferExtFromBody tries to infer a reasonable extension from the file signature/content,
// restricted to PDF and Office formats only.
func inferExtFromBody(body []byte) string {
	if len(body) < 4 {
		return ""
	}
	sig := string(body[:4])
	switch sig {
	case "%PDF":
		return ".pdf"
	case "PK\x03\x04":
		// OOXML (zip): decide based on internal folder names
		readerAt := bytes.NewReader(body)
		size := int64(len(body))
		if zr, err := zip.NewReader(readerAt, size); err == nil {
			for _, f := range zr.File {
				n := f.Name
				if strings.HasPrefix(n, "ppt/") {
					return ".pptx"
				}
				if strings.HasPrefix(n, "word/") {
					return ".docx"
				}
				if strings.HasPrefix(n, "xl/") {
					return ".xlsx"
				}
			}
		}
		return ""
	case "\xD0\xCF\x11\xE0":
		// Legacy OLE Compound File: inspect for stream names
		window := body
		if len(window) > 16384 {
			window = body[:16384]
		}
		if bytes.Contains(window, []byte("PowerPoint Document")) {
			return ".ppt"
		}
		if bytes.Contains(window, []byte("WordDocument")) {
			return ".doc"
		}
		if bytes.Contains(window, []byte("Workbook")) || bytes.Contains(window, []byte("Book")) {
			return ".xls"
		}
		return ".ppt" // default to most common legacy type
	default:
		// Unknown: do not return non-office/image/zip extensions
		return ""
	}
}

// fixNumericSuffix converts a trailing dotted numeric suffix into a hyphenated form.
// Examples:
//
//	"Lecture 1.1"   -> "Lecture 1-1"
//	"Topic A 2.3.4" -> "Topic A 2-3-4"
func fixNumericSuffix(name string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)+)$`)
	if m := re.FindStringSubmatchIndex(name); len(m) >= 4 {
		start, end := m[2], m[3]
		tail := name[start:end]
		tail = strings.ReplaceAll(tail, ".", "-")
		return name[:start] + tail
	}
	return name
}
