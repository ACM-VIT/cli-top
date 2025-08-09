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

	// "regexp"
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

	fmt.Println(selectedCourse)

	materials, faculty, err := fetchFacultieswithMaterials(regNo, cookies, selectedCourse.ID, selectedCourse.Name)
	if err != nil {
		fmt.Println("Error fetching faculties:", err)
		return
	}

	fmt.Println(materials)

	displayCourseMaterials(materials)

	selectedMaterials, err := selectCourseMaterials(materials)
	if err != nil {
		fmt.Println("Error selecting materials:", err)
		return
	}

	fmt.Println(selectedMaterials)
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

	// Custom sort for semesters: by year, then by season (Fall < Winter)
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

	// Ask user to select semester
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
	selectedSemester := semesters[semResult.Index-1]

	// Filter courses by selected semester
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

func fetchFacultieswithMaterials(regNo string, cookies types.Cookies, courseID string, courseName string) ([]types.CourseMaterial, types.Faculty, error) {
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
		// rawFaculty expected format: "ERPID - Faculty Name - SCOPE"
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
		} else {
			topic = strings.TrimSpace(cells.Eq(2).Text())
		}

		// Download button is in cell 4
		downloadBtn := cells.Eq(4).Find("button[name='downloadmat']")
		materialID, _ := downloadBtn.Attr("data-fileid")
		// Use topic as the material name (ignore button text)
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
	// Debug print the faculty list
	fmt.Println("DEBUG: Extracted Faculties:")
	for _, f := range facultyList {
		fmt.Printf("%+v\n", f)
	}

	// Prompt the user to select a faculty
	nestedList := [][]string{{"FACULTY"}}
	for _, f := range facultyList {
		nestedList = append(nestedList, []string{f.Name})
	}
	result := helpers.TableSelector("Faculty", nestedList, "")
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
		return nil, types.Faculty{}, fmt.Errorf("no materials found for selected faculty")
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

func downloadMaterialsHope(regNo string, cookies types.Cookies, selectedCourse types.Course, allMaterials []types.CourseMaterial, selectedMaterials []types.CourseMaterial, faculty types.Faculty) error {
	// Create a wait group to manage concurrent downloads
	var wg sync.WaitGroup
	errChan := make(chan error, len(selectedMaterials))

	coursePageDir, err := helpers.GetOrCreateDownloadDir("Course Page")
	if err != nil {
		return fmt.Errorf("failed to create course page directory: %w", err)
	}

	courseParts := helpers.SplitCourseNameFull(selectedCourse.Name)
	// Merge courseParts[0] and courseParts[1], then take courseParts[3], then faculty.Name
	semesterName := fmt.Sprintf("%s_%s", courseParts[0], courseParts[1])
	courseName := courseParts[3]

	fullDirPath := filepath.Join(coursePageDir, semesterName, courseName, faculty.Name)
	fmt.Println(fullDirPath)

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
	sem := make(chan struct{}, concurrency)

	// Download each material concurrently
	for _, material := range selectedMaterials {
		wg.Add(1)
		// Acquire slot in semaphore for each course material download
		sem <- struct{}{}
		go func(mat types.CourseMaterial) {
			defer wg.Done()
			// Iterate over each reference material for the course material
			for _, refMat := range mat.ReferenceMaterials {
				isPotentialPptx := strings.Contains(strings.ToLower(refMat.Name), "ppt") ||
					strings.Contains(strings.ToLower(refMat.Name), "presentation") ||
					strings.Contains(strings.ToLower(refMat.Name), "slide")

				downloadURL := "https://vtop.vit.ac.in/vtop/downloadCourseMaterialFacultyPdf"
				payloadMap := map[string]string{
					"_csrf":        cookies.CSRF,
					"authorizedID": regNo,
					"fileId":       refMat.MaterialID,
				}
				formData := helpers.FormatBodyDataClient(payloadMap)

				var body []byte
				// var headers http.Header
				retries := 2
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

					body, _, downloadErr = helpers.FetchReqClient(attemptClient, regNo, cookies, downloadURL, "", formData, "POST", "application/x-www-form-urlencoded")
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
					errChan <- fmt.Errorf("failed to download %s: %s", mat.Topic, lastError)
				} else {
					filePath := filepath.Join(fullDirPath, refMat.Name)

					err = helpers.SaveFile(body, filePath)
					if err != nil {
						lastError = err.Error()
					}

					fmt.Printf("Downloaded: %s\n", mat.Topic)
				}
				// Update progress bar after each ref material downloaded (success or failure)
				bar.Add(1)
			}
			<-sem
		}(material)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Handle errors
	for err := range errChan {
		fmt.Println("Error:", err)
	}

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
