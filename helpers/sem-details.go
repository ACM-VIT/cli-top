package helpers

import (
	"bytes"
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func FindAndSaveSemIds(body []byte) ([]types.Semester, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse semester page: %w", err)
	}

	selectors := []string{
		"select[id*='semesterSubId']",
		"select[name*='semesterSubId']",
		"select.form-select",
		"select",
	}
	for _, selector := range selectors {
		var semesters []types.Semester
		doc.Find(selector).EachWithBreak(func(_ int, selection *goquery.Selection) bool {
			selection.Find("option").Each(func(_ int, option *goquery.Selection) {
				id := strings.TrimSpace(option.AttrOr("value", ""))
				name := strings.TrimSpace(option.Text())
				if id != "" && name != "" {
					semesters = append(semesters, types.Semester{SemID: id, SemName: name})
				}
			})
			return len(semesters) == 0
		})
		if len(semesters) > 0 {
			return semesters, nil
		}
	}

	if debug.Debug {
		fmt.Println("No semesters found in document.")
	}
	return nil, fmt.Errorf("no semesters found")
}

// GetSemDetails fetches semester details
func GetSemDetails(cookies types.Cookies, regNo string) ([]types.Semester, error) {
	if cached, ok := getCachedSemesters(regNo); ok {
		return cached, nil
	}
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		return nil, fmt.Errorf("please login first using the cli-top login command")
	}
	endpoints := []string{
		"https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance",
		"https://vtop.vit.ac.in/vtop/academics/common/StudentCoursePage",
	}
	var lastErr error
	for _, endpoint := range endpoints {
		body, err := FetchReq(regNo, cookies, endpoint, "", "", "POST", "")
		if err != nil {
			lastErr = err
			continue
		}
		semesters, err := FindAndSaveSemIds(body)
		if err != nil {
			lastErr = err
			continue
		}
		ReverseSlice(semesters)
		storeSemesters(regNo, semesters)
		return semesters, nil
	}
	if debug.Debug && lastErr != nil {
		fmt.Println("Error fetching semester details:", lastErr)
	}
	return nil, fmt.Errorf("fetch semester details: %w", lastErr)
}

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) (types.Semester, error) {
	semDetails, err := GetSemDetails(cookies, regNo)
	var selectedSem types.Semester
	if err != nil {
		return selectedSem, err
	}
	if len(semDetails) == 0 {
		if debug.Debug {
			fmt.Println("Error fetching semester details", err)
		}
		return selectedSem, fmt.Errorf("error fetching semester details or no semesters available. Try logging out and logging back in")
	}

	var nested_sem_list [][]string
	nested_sem_list = append(nested_sem_list, []string{"Semester ID", "Semester"})
	for i := 0; i < len(semDetails); i++ {
		nested_sem_list = append(nested_sem_list, []string{semDetails[i].SemID, semDetails[i].SemName})
	}

	if os.Getenv("CLI_TOP_PROXY_MODE") == "1" && sem_choice <= 0 {
		selectedSem = semDetails[len(semDetails)-1]
		return selectedSem, nil
	}

	choice := TableSelector("semester", nested_sem_list, strconv.Itoa(sem_choice))
	if choice.ExitRequest {
		return selectedSem, fmt.Errorf("selection canceled by user")
	}
	if !choice.Selected || choice.Index < 1 || choice.Index > len(semDetails) {
		return selectedSem, fmt.Errorf("invalid semester selection")
	}
	selectedSem = semDetails[choice.Index-1]

	return selectedSem, nil
}
