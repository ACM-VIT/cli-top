package features

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cli-top/helpers"
	"cli-top/types"

	"github.com/PuerkitoBio/goquery"
)

const (
	SyllabusCategorySelector     = "div.row[style*='cursor: pointer']"
	SyllabusCategoryNameSelector = "div.col-6"
	SyllabusTableSelector        = "table.example"
	SyllabusTableAltSelector     = "table[id^='tableData']"
	SyllabusRowsSelector         = "tbody tr"
	SyllabusCellSelector         = "td"
	SyllabusCourseCodeIndex      = 1
	SyllabusCourseTitleIndex     = 2
)

type Category struct {
	ID   string
	Name string
}

type Course struct {
	Code  string
	Title string
}

func sanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
	return re.ReplaceAllString(filename, "_")
}

func DownloadSyllabus(courseCode, courseName string, cookies types.Cookies, authorizedID, outputDir string) (string, error) {
	downloadURL := "https://vtop.vit.ac.in/vtop/courseSyllabusDownload1"
	payload := fmt.Sprintf("_csrf=%s&_csrf=%s&authorizedID=%s&courseCode=%s", cookies.CSRF, cookies.CSRF, authorizedID, courseCode)

	bodyBytes, err := helpers.FetchReq("", cookies, downloadURL, "", payload, "POST", "")
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	ct := http.DetectContentType(bodyBytes)
	var pdfBytes []byte
	if strings.HasPrefix(ct, "application/pdf") {
		pdfBytes = bodyBytes
	} else if strings.HasPrefix(ct, "application/zip") || strings.HasPrefix(ct, "application/x-zip-compressed") {
		zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
		if err != nil {
			return "", fmt.Errorf("failed to read zip file: %w", err)
		}
		found := false
		for _, file := range zipReader.File {
			if strings.HasSuffix(strings.ToLower(file.Name), ".pdf") {
				rc, err := file.Open()
				if err != nil {
					return "", fmt.Errorf("failed to open pdf file in zip: %w", err)
				}
				pdfBytes, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return "", fmt.Errorf("failed to read pdf file from zip: %w", err)
				}
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("no pdf file found in zip")
		}
	} else {
		return "", fmt.Errorf("unexpected content type: %s", ct)
	}
	filename := fmt.Sprintf("%s_%s.pdf", courseName, courseCode)
	sanitizedFilename := sanitizeFilename(filename)
	outputPath := filepath.Join(outputDir, sanitizedFilename)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, pdfBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}
	fmt.Printf("Successfully downloaded syllabus for course %s. File saved at: %s\n", courseCode, outputPath)
	return outputPath, nil
}

func getCurriculumCategories(regNo string, cookies types.Cookies) ([]Category, error) {
	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
	endpoint := "https://vtop.vit.ac.in/vtop/academics/common/Curriculum"
	body, err := helpers.FetchReq(regNo, cookies, endpoint, "", payload, "POST", "")
	if err != nil {
		return nil, fmt.Errorf("error fetching curriculum page: %w", err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error parsing curriculum page HTML: %w", err)
	}
	var categories []Category
	doc.Find("div.card.categoty-card").Each(func(i int, s *goquery.Selection) {
		onclick, exists := s.Find(SyllabusCategorySelector).Attr("onclick")
		if !exists {
			return
		}
		re := regexp.MustCompile(`categoryOnClick\('([^']+)'\)`)
		matches := re.FindStringSubmatch(onclick)
		if len(matches) < 2 {
			return
		}
		catID := matches[1]
		catName := strings.TrimSpace(s.Find(SyllabusCategoryNameSelector).Text())
		if catName != "" && catID != "" {
			categories = append(categories, Category{ID: catID, Name: catName})
		}
	})
	if len(categories) == 0 {
		return nil, fmt.Errorf("no syllabus categories found")
	}

	return categories, nil
}

func getCoursesForCategory(regNo string, cookies types.Cookies, categoryID string) ([]Course, error) {
	payload := fmt.Sprintf("_csrf=%s&categoryId=%s&authorizedID=%s&x=%s", cookies.CSRF, categoryID, regNo, time.Now().UTC().Format(time.RFC1123))
	endpoint := "https://vtop.vit.ac.in/vtop/academics/common/curriculumCategoryView"
	body, err := helpers.FetchReq(regNo, cookies, endpoint, "", payload, "POST", "")
	if err != nil {
		return nil, fmt.Errorf("error fetching category view: %w", err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error parsing category view HTML: %w", err)
	}
	table := doc.Find(SyllabusTableSelector)
	if table.Length() == 0 {
		table = doc.Find(SyllabusTableAltSelector)
	}
	if table.Length() == 0 {
		return nil, fmt.Errorf("no course table found in category view")
	}
	var courses []Course
	table.Find(SyllabusRowsSelector).Each(func(i int, s *goquery.Selection) {
		tds := s.Find(SyllabusCellSelector)
		if tds.Length() < 3 {
			return
		}
		code := ""
		s.Find(SyllabusCellSelector).Eq(SyllabusCourseCodeIndex).Find("button").Each(func(i int, btn *goquery.Selection) {
			if val, exists := btn.Attr("data-coursecode"); exists {
				code = strings.TrimSpace(val)
			}
		})
		if code == "" {
			code = strings.TrimSpace(tds.Eq(SyllabusCourseCodeIndex).Text())
		}
		title := strings.TrimSpace(tds.Eq(SyllabusCourseTitleIndex).Text())
		if code != "" && title != "" {
			courses = append(courses, Course{Code: code, Title: title})
		}
	})
	if len(courses) == 0 {
		return nil, fmt.Errorf("no courses found in the selected category")
	}
	return courses, nil
}

func ExecuteSyllabusDownload(regNo string, cookies types.Cookies, courseSearch string) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	categories, err := getCurriculumCategories(regNo, cookies)
	if err != nil {
		helpers.HandleError("fetching syllabus categories", err)
		return
	}

	// Get all courses from all categories
	var allCourses []struct {
		Course   Course
		Category Category
	}

	for _, category := range categories {
		courses, err := getCoursesForCategory(regNo, cookies, category.ID)
		if err != nil {
			fmt.Printf("Error fetching courses for category %s: %v\n", category.Name, err)
			continue
		}

		for _, course := range courses {
			allCourses = append(allCourses, struct {
				Course   Course
				Category Category
			}{
				Course:   course,
				Category: category,
			})
		}
	}

	if len(allCourses) == 0 {
		fmt.Println("No courses found in any category.")
		return
	}

	// Prepare table for fuzzy search selection
	var courseTable [][]string
	courseTable = append(courseTable, []string{"Course Title", "Course Code", "Category"})
	for _, courseData := range allCourses {
		courseTable = append(courseTable, []string{
			courseData.Course.Title,
			courseData.Course.Code,
			courseData.Category.Name,
		})
	}

	// Use fuzzy search for course selection with the provided courseSearch query flag
	selectedCourseIndex := helpers.TableSelectorFuzzy("Course", courseTable, courseSearch, helpers.FuzzySearchWithAcronym)
	if selectedCourseIndex.ExitRequest || !selectedCourseIndex.Selected {
		fmt.Println("Selection canceled")
		return
	}

	selectedCourseData := allCourses[selectedCourseIndex.Index-1] // Adjust for header row

	// Create the Syllabus directory inside CLI-TOP Downloads
	outputDir, err := helpers.GetOrCreateDownloadDir("Syllabus")
	if err != nil {
		helpers.HandleError("creating syllabus download directory", err)
		return
	}

	downloadedPath, err := DownloadSyllabus(
		selectedCourseData.Course.Code,
		selectedCourseData.Course.Title,
		cookies,
		regNo,
		outputDir,
	)
	if err != nil {
		helpers.HandleError("downloading syllabus", err)
		return
	}

	pathDir := filepath.Dir(downloadedPath)

	helpers.OpenFolder(pathDir)
}
