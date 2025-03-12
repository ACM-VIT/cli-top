package features

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cli-top/helpers"
	"cli-top/types"

	"github.com/PuerkitoBio/goquery"
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

func DownloadSyllabus(courseCode, courseName, csrfToken, authorizedID, sessionCookie, outputDir string) (string, error) {
	downloadURL := "https://vtop.vit.ac.in/vtop/courseSyllabusDownload1"
	payload := fmt.Sprintf("_csrf=%s&_csrf=%s&authorizedID=%s&courseCode=%s", csrfToken, csrfToken, authorizedID, courseCode)
	req, err := http.NewRequest("POST", downloadURL, strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("origin", "https://vtop.vit.ac.in")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("priority", "u=0, i")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content")
	req.Header.Set("sec-ch-ua", `"Chromium";v="134", "Not:A-Brand";v="24", "Brave";v="134"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("sec-gpc", "1")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("cookie", sessionCookie)
	client := &http.Client{Timeout: time.Minute * 2}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
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
				pdfBytes, err = ioutil.ReadAll(rc)
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
	filename := fmt.Sprintf("%s_%s.pdf", courseCode, courseName)
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
		onclick, exists := s.Find("div.row[style*='cursor: pointer']").Attr("onclick")
		if !exists {
			return
		}
		re := regexp.MustCompile(`categoryOnClick\('([^']+)'\)`)
		matches := re.FindStringSubmatch(onclick)
		if len(matches) < 2 {
			return
		}
		catID := matches[1]
		catName := strings.TrimSpace(s.Find("div.col-6").Text())
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
	table := doc.Find("table.example")
	if table.Length() == 0 {
		table = doc.Find("table[id^='tableData']")
	}
	if table.Length() == 0 {
		return nil, fmt.Errorf("no course table found in category view")
	}
	var courses []Course
	table.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() < 3 {
			return
		}
		code := ""
		s.Find("td").Eq(1).Find("button").Each(func(i int, btn *goquery.Selection) {
			if val, exists := btn.Attr("data-coursecode"); exists {
				code = strings.TrimSpace(val)
			}
		})
		if code == "" {
			code = strings.TrimSpace(tds.Eq(1).Text())
		}
		title := strings.TrimSpace(tds.Eq(2).Text())
		if code != "" && title != "" {
			courses = append(courses, Course{Code: code, Title: title})
		}
	})
	if len(courses) == 0 {
		return nil, fmt.Errorf("no courses found in the selected category")
	}
	return courses, nil
}

func OpenFolder(path string) {
	var cmd *exec.Cmd
	switch os := os.Getenv("OS"); os {
	case "Windows_NT":
		cmd = exec.Command("explorer", filepath.Dir(path))
	default:
		if _, err := exec.LookPath("open"); err == nil {
			cmd = exec.Command("open", filepath.Dir(path))
		} else if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", filepath.Dir(path))
		} else {
			fmt.Println("Please open the folder manually:", filepath.Dir(path))
			return
		}
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error opening folder: %v\n", err)
	}
}

func ExecuteSyllabusDownload(regNo string, cookies types.Cookies) {
	if !helpers.ValidateCookies(cookies) {
		fmt.Println("Please login using the cli-top login command.")
		return
	}
	categories, err := getCurriculumCategories(regNo, cookies)
	if err != nil {
		helpers.HandleError("fetching syllabus categories", err)
		return
	}
	var catTable [][]string
	catTable = append(catTable, []string{"Syllabus Category"})
	for _, cat := range categories {
		catTable = append(catTable, []string{cat.Name})
	}
	selectedCatIndex := helpers.TableSelector("Syllabus Category", catTable, 0)
	if selectedCatIndex < 1 || selectedCatIndex > len(categories) {
		fmt.Println("Invalid category selection.")
		return
	}
	selectedCategory := categories[selectedCatIndex-1]
	courses, err := getCoursesForCategory(regNo, cookies, selectedCategory.ID)
	if err != nil {
		helpers.HandleError("fetching courses", err)
		return
	}
	var courseTable [][]string
	courseTable = append(courseTable, []string{"Course Title", "Course Code"})
	for _, course := range courses {
		courseTable = append(courseTable, []string{course.Title, course.Code})
	}
	selectedCourseIndex := helpers.TableSelector("Course", courseTable, 0)
	if selectedCourseIndex < 1 || selectedCourseIndex > len(courses) {
		fmt.Println("Invalid course selection.")
		return
	}
	selectedCourse := courses[selectedCourseIndex-1]
	//fmt.Printf("You selected course: %s (%s)\n", selectedCourse.Title, selectedCourse.Code)
	outputDir := filepath.Join(helpers.GetDownloadsDir(), "Syllabus Downloads")
	cookieStr := fmt.Sprintf("JSESSIONID=%s; SERVERID=%s;", cookies.JSESSIONID, cookies.SERVERID)
	downloadedPath, err := DownloadSyllabus(selectedCourse.Code, selectedCourse.Title, cookies.CSRF, regNo, cookieStr, outputDir)
	if err != nil {
		helpers.HandleError("downloading syllabus", err)
		return
	}
	//fmt.Printf("Syllabus downloaded successfully to: %s\n", downloadedPath)
	OpenFolder(downloadedPath)
}
