package features

import (
	"bufio"
	"cli-top/helpers"
	"cli-top/types"
	"math/rand"
	"os"
	"regexp"
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
	getCoursesListEndpoint         = "academics/common/getCoursesListForCurriculmCategory"
	getCoursesDetailEndpoint       = "academics/common/getCoursesDetailForRegistration"
	curriculumCategorySelector     = "select#curriculumCategory option"
	curriculumDropdownSelector     = "select#curriculumCategory"
	courseListSelector             = "select#courseId option"
	courseDetailTableSelector      = "div#courseDetailFragement table.table-bordered"
	courseDetailRowSelector        = "tbody tr"
	courseDetailCellSelector       = "td"
	csrfVarRegexPattern            = `var csrfValue = "([^"]+)"`
	authIDVarRegexPattern          = `var id="([^"]+)"`
	associatedFunctionHintCourses  = "getCoursesListForCurriculmCategory"
	associatedFunctionHintDetails  = "getCoursesDetail"
)

type CourseAllocationDetail struct {
	Code    string
	Title   string
	Type    string
	Venue   string
	Slot    string
	Faculty string
}

func ExecuteInteractiveCourseAllocationView(regNo string, cookies types.Cookies, courseAllocationPageURL string, categoryQuery string, courseQuery string) {
	if !helpers.ValidateLogin(cookies) {
		helpers.Println("User not logged in or session expired.")
		return
	}

	if courseAllocationPageURL == "" {
		courseAllocationPageURL = DefaultCourseAllocationPageURL
	}

	initialPayloadMap := map[string]string{
		"verifyMenu":   "true",
		"authorizedID": regNo,
		"_csrf":        cookies.CSRF,
		"nocache":      strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10),
	}
	initialFormData := helpers.FormatBodyData(initialPayloadMap)
	initialPageHTMLBytes, _, err := helpers.FetchReqClient(helpers.GetHTTPClient(), cookies, courseAllocationPageURL, "", initialFormData, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		return
	}
	initialPageHTML := string(initialPageHTMLBytes)

	if len(initialPageHTML) < 500 || strings.Contains(initialPageHTML, "Session Timed Out") || strings.Contains(strings.ToLower(initialPageHTML), "login") {
		return
	}

	pageCsrfToken, pageAuthID := extractScriptParams(initialPageHTML)
	currentAjaxCsrfToken := cookies.CSRF
	if pageCsrfToken != "" {
		currentAjaxCsrfToken = pageCsrfToken
	}
	currentAjaxAuthID := regNo
	if pageAuthID != "" {
		currentAjaxAuthID = pageAuthID
	}

	initialDoc, err := goquery.NewDocumentFromReader(strings.NewReader(initialPageHTML))
	if err != nil {
		helpers.Printf("Error parsing initial page HTML: %v\n", err)
		return
	}

	if initialDoc.Find(curriculumDropdownSelector).Length() == 0 {
		helpers.Println("\nCourse allocation details are not available at this time")
		return
	}

	ajaxBaseURL := ""
	if strings.Contains(courseAllocationPageURL, "/vtop/") {
		ajaxBaseURL = strings.Split(courseAllocationPageURL, "/vtop/")[0] + "/vtop/"
	} else {
		ajaxBaseURL = "https://vtop.vit.ac.in/vtop/"
	}

	for {
		selectedCategory, categoryAction := selectCurriculumCategory(initialDoc, categoryQuery)
		switch categoryAction {
		case actionExitApp:
			return
		case actionSetupFailed:
			return
		case actionError:
			continue
		case actionSelected:
		CourseLoop:
			for {
				selectedCourse, courseAction := selectCourseFromCategory(selectedCategory, currentAjaxCsrfToken, currentAjaxAuthID, ajaxBaseURL, regNo, cookies, courseQuery)
				switch courseAction {
				case actionExitApp:
					return
				case actionGoBack:
					break CourseLoop
				case actionError:
					continue
				case actionSelected:
					detailsAction := displayCourseAllocationDetails(selectedCourse, currentAjaxCsrfToken, currentAjaxAuthID, ajaxBaseURL, regNo, cookies)
					if detailsAction == actionExitApp {
						return
					}
				}
			}
		}
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

func selectCurriculumCategory(initialDoc *goquery.Document, categoryQuery string) (types.Category, string) {
	var categories []types.Category
	initialDoc.Find(curriculumCategorySelector).Each(func(_ int, s *goquery.Selection) {
		val, exists := s.Attr("value")
		if exists && val != "" {
			categories = append(categories, types.Category{ID: val, Name: strings.TrimSpace(s.Text())})
		}
	})
	if len(categories) == 0 {
		return types.Category{}, actionSetupFailed
	}
	tableData := [][]string{{"CATEGORY NAME"}}
	for _, cat := range categories {
		tableData = append(tableData, []string{cat.Name})
	}
	selectionResult := helpers.TableSelectorFuzzy("Category", tableData, categoryQuery, helpers.NewFuzzySearch)
	if selectionResult.ExitRequest {
		return types.Category{}, actionExitApp
	}
	if !selectionResult.Selected || selectionResult.Index < 1 || selectionResult.Index > len(categories) {
		if helpers.ShouldMuteUI() {
			return types.Category{}, actionExitApp
		}
		helpers.Println("Invalid selection.")
		return types.Category{}, actionError
	}
	return categories[selectionResult.Index-1], actionSelected
}

func selectCourseFromCategory(category types.Category, csrfToken, authID, baseURL, regNo string, cookies types.Cookies, courseQuery string) (types.Course, string) {
	courseListParams := map[string]string{
		"_csrf": csrfToken, "cccategory": category.ID,
		"authorizedID": authID, "x": time.Now().UTC().Format(time.RFC1123),
	}
	formDataCourses := helpers.FormatBodyData(courseListParams)
	courseListHTMLBytes, _, err := helpers.FetchReqClient(helpers.GetHTTPClient(), cookies, baseURL+getCoursesListEndpoint, "", formDataCourses, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		helpers.Printf("Error fetching course list for category %s: %v\n", category.Name, err)
		return types.Course{}, actionError
	}
	courseListHTML := string(courseListHTMLBytes)
	if len(courseListHTML) < 10 && (strings.Contains(strings.ToLower(courseListHTML), "error")) ||
		strings.Contains(courseListHTML, "Session Timed Out") || strings.Contains(strings.ToLower(courseListHTML), "login required") {
		helpers.Printf("Error or session issue fetching courses for %s.\n", category.Name)
		return types.Course{}, actionError
	}

	courseListDoc, err := goquery.NewDocumentFromReader(strings.NewReader(courseListHTML))
	if err != nil {
		helpers.Printf("Error parsing course list for category %s: %v\n", category.Name, err)
		return types.Course{}, actionError
	}
	var courses []types.Course
	courseListDoc.Find(courseListSelector).Each(func(_ int, s *goquery.Selection) {
		code, exists := s.Attr("value")
		if exists && code != "" {
			courses = append(courses, types.Course{ID: code, Name: strings.TrimSpace(s.Text())})
		}
	})
	if len(courses) == 0 {
		helpers.Printf("No courses found for category: %s.\n", category.Name)
		return types.Course{}, actionGoBack
	}
	tableData := [][]string{{"COURSE (CODE - TITLE)"}}
	for _, c := range courses {
		tableData = append(tableData, []string{c.Name})
	}
	selectionResult := helpers.TableSelectorFuzzy("Course", tableData, courseQuery, helpers.NewFuzzySearch)
	if selectionResult.ExitRequest {
		return types.Course{}, actionExitApp
	}
	if !selectionResult.Selected || selectionResult.Index < 1 || selectionResult.Index > len(courses) {
		if helpers.ShouldMuteUI() {
			return types.Course{}, actionExitApp
		}
		return types.Course{}, actionGoBack
	}
	return courses[selectionResult.Index-1], actionSelected
}

func displayCourseAllocationDetails(course types.Course, csrfToken, authID, baseURL, regNo string, cookies types.Cookies) string {
	time.Sleep(time.Duration(100+rand.Intn(150)) * time.Millisecond)
	courseDetailParams := map[string]string{
		"_csrf": csrfToken, "courseCode": course.ID,
		"authorizedID": authID, "x": time.Now().UTC().Format(time.RFC1123),
	}
	formDataDetails := helpers.FormatBodyData(courseDetailParams)
	courseDetailHTMLBytes, _, err := helpers.FetchReqClient(helpers.GetHTTPClient(), cookies, baseURL+getCoursesDetailEndpoint, "", formDataDetails, "POST", "application/x-www-form-urlencoded")
	if err != nil {
		helpers.Printf("Error fetching course details for %s: %v\n", course.Name, err)
		return actionError
	}
	courseDetailHTML := string(courseDetailHTMLBytes)
	if len(courseDetailHTML) < 10 && (strings.Contains(strings.ToLower(courseDetailHTML), "error")) ||
		strings.Contains(courseDetailHTML, "Session Timed Out") || strings.Contains(strings.ToLower(courseDetailHTML), "login required") {
		helpers.Printf("Error or session issue fetching details for %s.\n", course.Name)
		return actionError
	}

	courseDetailDoc, err := goquery.NewDocumentFromReader(strings.NewReader(courseDetailHTML))
	if err != nil {
		return actionError
	}
	var detailsList []CourseAllocationDetail
	table := courseDetailDoc.Find(courseDetailTableSelector)
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
		})
		if len(detailsList) > 0 {
			tableData := [][]string{{"FACULTY", "VENUE", "SLOT", "TYPE"}}
			for _, d := range detailsList {
				tableData = append(tableData, []string{d.Faculty, d.Venue, d.Slot, d.Type})
			}
			helpers.PrintTable(tableData, 0)
		} else {
			//helpers.Println("Details table found, but no rows matched expected structure (4 cells).")
		}
	}
	if helpers.ShouldMuteUI() {
		return actionExitApp
	}
	helpers.Println("\nPress 'b' to go back to course list, or 'q' to exit.")
	reader := bufio.NewReader(os.Stdin)
	for {
		helpers.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "b" {
			return actionGoBack
		}
		if input == "q" {
			return actionExitApp
		}
		helpers.Println("Invalid input. 'b' for back, 'q' for quit.")
	}
}
