package features

import (
	"bufio"
	"bytes"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	actionSelected    = "SELECTED"
	actionGoBack      = "BACK"
	actionExitApp     = "EXIT"
	actionError       = "ERROR_OCCURRED"
	actionSetupFailed = "SETUP_FAILED"
)

const (
	DefaultCourseAllocationPageURL = "https://vtop.vit.ac.in/vtop/academics/common/StudentRegistrationScheduleAllocation"
	getCategoriesEndpoint          = "https://vtop.vit.ac.in/vtop/academics/common/getCoursesListForCurriculmCategory"
	getCourseDetailsEndpoint       = "https://vtop.vit.ac.in/vtop/academics/common/getCoursesDetailForRegistration"
	curriculumCategorySelector     = "select#curriculumCategory option"
	curriculumDropdownSelector     = "select#curriculumCategory"
	courseListSelector             = "select#courseId option"
	courseDetailTableSelector      = "div#courseDetailFragement table.table-bordered"
	courseDetailRowSelector        = "tbody tr"
	courseDetailCellSelector       = "td"
)

type CourseAllocationDetail struct {
	Code    string
	Title   string
	Type    string
	Venue   string
	Slot    string
	Faculty string
}

var courseAllocationHttpClient *http.Client

func init() {
	courseAllocationHttpClient = &http.Client{
		Timeout: time.Duration(60) * time.Second,
	}
}

func ExecuteInteractiveCourseAllocationView(regNo string, cookies types.Cookies, courseAllocationPageURL string) {
	if !helpers.ValidateLogin(cookies) {
		fmt.Println("User not logged in or session expired.")
		return
	}

	if courseAllocationPageURL == "" {
		courseAllocationPageURL = DefaultCourseAllocationPageURL
	}

	// Fetch initial page to get categories
	initialPayloadMap := map[string]string{
		"verifyMenu":   "true",
		"authorizedID": regNo,
		"_csrf":        cookies.CSRF,
		"nocache":      strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10),
	}
	initialFormData := helpers.FormatBodyDataClient(initialPayloadMap)
	initialPageHTMLBytes, _, err := helpers.FetchReqClient(courseAllocationHttpClient, regNo, cookies, courseAllocationPageURL, "", initialFormData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		fmt.Println("Error fetching course allocation page:", err)
		return
	}

	initialDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(initialPageHTMLBytes))
	if err != nil {
		fmt.Printf("Error parsing initial page HTML: %v\n", err)
		return
	}

	if initialDoc.Find(curriculumDropdownSelector).Length() == 0 {
		fmt.Printf("The course allocation activity is inactive\n")
		return
	}

	// Get categories and let user select
	selectedCategory, err := fetchAndSelectCategory(initialDoc)
	if err != nil {
		if err.Error() == "selection canceled by user" {
			fmt.Println("Selection canceled")
			return
		}
		fmt.Println("Error selecting category:", err)
		return
	}

	// Get courses for selected category
	selectedCourse, err := fetchAndSelectCourseAllocation(regNo, cookies, selectedCategory)
	if err != nil {
		if err.Error() == "selection canceled by user" {
			fmt.Println("Selection canceled")
			return
		}
		fmt.Println("Error selecting course:", err)
		return
	}

	// Display course allocation details
	err = displayCourseDetails(regNo, cookies, selectedCourse)
	if err != nil {
		fmt.Println("Error displaying course details:", err)
		return
	}
}

func extractScriptParams(htmlContent string) (csrfToken string, authID string) {
	scriptRegex := regexp.MustCompile(`(?s)<script.*?>(.*?)</script>`)
	csrfRegex := regexp.MustCompile(csrfVarRegexPattern)
	authIDRegex := regexp.MustCompile(authIDVarRegexPattern)
	scripts := scriptRegex.FindAllStringSubmatch(htmlContent, -1)
	foundCsrf, foundAuthID := false, false
	for _, scriptMatch := range scripts {
		if len(scriptMatch) < 2 {
			continue
		}
		scriptContent := scriptMatch[1]
		isRelevant := strings.Contains(scriptContent, associatedFunctionHintCourses) || strings.Contains(scriptContent, associatedFunctionHintDetails)
		if !foundCsrf {
			m := csrfRegex.FindStringSubmatch(scriptContent)
			if len(m) > 1 && (isRelevant || csrfToken == "") {
				csrfToken = m[1]
				foundCsrf = true
			}
		}
		if !foundAuthID {
			m := authIDRegex.FindStringSubmatch(scriptContent)
			if len(m) > 1 && (isRelevant || authID == "") {
				authID = m[1]
				foundAuthID = true
			}
		}
		if foundCsrf && foundAuthID && isRelevant {
			break
		}
	}
	return
}

func selectCurriculumCategory(initialDoc *goquery.Document, csrfToken, authID, baseURL, regNo string, cookies types.Cookies) (types.Category, string) {
	var categories []types.Category
	initialDoc.Find(curriculumCategorySelector).Each(func(_ int, s *goquery.Selection) {
		val, exists := s.Attr("value")
		if exists && val != "" {
			categories = append(categories, types.Category{ID: val, Name: strings.TrimSpace(s.Text())})
		}
	})

	if len(categories) == 0 {
		return types.Category{}, fmt.Errorf("no categories found")
	}

	tableData := [][]string{{"CATEGORY NAME"}}
	for _, cat := range categories {
		tableData = append(tableData, []string{cat.Name})
	}
	selectionResult := helpers.TableSelector("Category", tableData, "")
	if selectionResult.ExitRequest {
		return types.Category{}, actionExitApp
	}
	if !selectionResult.Selected || selectionResult.Index < 1 || selectionResult.Index > len(categories) {
		fmt.Println("Invalid selection.")
		return types.Category{}, actionError
	}

	if !result.Selected || result.Index < 1 || result.Index > len(categories) {
		return types.Category{}, fmt.Errorf("invalid category selection")
	}

	return categories[result.Index-1], nil
}

func fetchAndSelectCourseAllocation(regNo string, cookies types.Cookies, category types.Category) (types.Course, error) {
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"cccategory":   category.ID,
		"authorizedID": regNo,
		"x":            time.Now().UTC().Format(time.RFC1123),
	}

	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(courseAllocationHttpClient, regNo, cookies, getCategoriesEndpoint, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return types.Course{}, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return types.Course{}, err
	}

	var courses []types.Course
	doc.Find(courseListSelector).Each(func(_ int, s *goquery.Selection) {
		code, exists := s.Attr("value")
		if exists && code != "" {
			courses = append(courses, types.Course{ID: code, Name: strings.TrimSpace(s.Text())})
		}
	})

	if len(courses) == 0 {
		return types.Course{}, fmt.Errorf("no courses found for category: %s", category.Name)
	}

	tableData := [][]string{{"COURSE (CODE - TITLE)"}}
	for _, c := range courses {
		tableData = append(tableData, []string{c.Name})
	}
	selectionResult := helpers.TableSelector("Course", tableData, "")
	if selectionResult.ExitRequest {
		return types.Course{}, actionExitApp
	}
	if !selectionResult.Selected || selectionResult.Index < 1 || selectionResult.Index > len(courses) {
		return types.Course{}, actionGoBack
	}

	if !result.Selected || result.Index < 1 || result.Index > len(courses) {
		return types.Course{}, fmt.Errorf("invalid course selection")
	}

	return courses[result.Index-1], nil
}

func displayCourseDetails(regNo string, cookies types.Cookies, course types.Course) error {
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"courseCode":   course.ID,
		"authorizedID": regNo,
		"x":            time.Now().UTC().Format(time.RFC1123),
	}

	formData := helpers.FormatBodyDataClient(payloadMap)
	body, _, err := helpers.FetchReqClient(courseAllocationHttpClient, regNo, cookies, getCourseDetailsEndpoint, "", formData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return err
	}

	var detailsList []CourseAllocationDetail
	table := doc.Find(courseDetailTableSelector)
	if table.Length() == 0 {
		return actionGoBack
	} else {
		table.Find(courseDetailRowSelector).Each(func(_ int, row *goquery.Selection) {
			cells := row.Find(courseDetailCellSelector)
			if cells.Length() == 4 {
				titleParts := strings.SplitN(course.Name, " - ", 2)
				actualTitle := course.ID
				if len(titleParts) == 2 {
					actualTitle = titleParts[1]
				} else {
					actualTitle = course.Name
				}
				detailsList = append(detailsList, CourseAllocationDetail{
					Code: course.ID, Title: actualTitle,
					Slot: strings.TrimSpace(cells.Eq(0).Text()), Venue: strings.TrimSpace(cells.Eq(1).Text()),
					Faculty: strings.TrimSpace(cells.Eq(2).Text()), Type: strings.TrimSpace(cells.Eq(3).Text()),
				})
			}
			detailsList = append(detailsList, CourseAllocationDetail{
				Code:    course.ID,
				Title:   actualTitle,
				Slot:    strings.TrimSpace(cells.Eq(0).Text()),
				Venue:   strings.TrimSpace(cells.Eq(1).Text()),
				Faculty: strings.TrimSpace(cells.Eq(2).Text()),
				Type:    strings.TrimSpace(cells.Eq(3).Text()),
			})
		}
	})

	if len(detailsList) > 0 {
		tableData := [][]string{{"FACULTY", "VENUE", "SLOT", "TYPE"}}
		for _, d := range detailsList {
			tableData = append(tableData, []string{d.Faculty, d.Venue, d.Slot, d.Type})
		}
		helpers.PrintTable(tableData, 0)
	} else {
		fmt.Println("No allocation details found for this course")
	}

	fmt.Println("\nPress Enter to continue...")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "b" {
			return actionGoBack
		}
		if input == "q" {
			return actionExitApp
		}
		fmt.Println("Invalid input. 'b' for back, 'q' for quit.")
	}
}
