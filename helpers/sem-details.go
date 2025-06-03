package helpers

import (
	"bufio"
	"bytes"
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Initialize a single reader instance for the package
var reader = bufio.NewReader(os.Stdin)

// FindAndSaveSemIds finds and saves semester IDs from the document
func FindAndSaveSemIds(doc *goquery.Document) ([]types.Semester, error) {
	var allsems []types.Semester

	// Try multiple different selectors to find semester options
	selectors := []string{
		"select.form-select option",
		"select#semesterSubId option",
		"select[name='semesterSubId'] option",
		"select option",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			var tempSem types.Semester
			var exists bool
			tempSem.SemID, exists = s.Attr("value")
			tempSem.SemName = strings.TrimSpace(s.Text())
			if exists && tempSem.SemID != "" && tempSem.SemName != "" {
				allsems = append(allsems, tempSem)
			}
		})

		// If we found semesters with this selector, break the loop
		if len(allsems) > 0 {
			break
		}
	}

	if len(allsems) == 0 {
		// If debug is enabled, output the HTML to help diagnose the issue
		if debug.Debug {
			fmt.Println("Document structure:")
			html, _ := doc.Html()
			fmt.Println(html)
		}
		return nil, fmt.Errorf("no semesters found")
	}
	return allsems, nil
}

// GetSemDetails fetches semester details
func GetSemDetails(cookies types.Cookies, regNo string) ([]types.Semester, error) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		return nil, fmt.Errorf("please login first using the cli-top login command")
	}
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"
	var allSems []types.Semester
	bodyText, err := FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching semester details", err)
		}
		return allSems, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing the HTML document:", err)
		}
		return allSems, err
	}
	allSems, err = FindAndSaveSemIds(doc)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching semester details", err)
		}
		return allSems, err
	}
	ReverseSlice(allSems)
	return allSems, nil
}

func GetSemDetailsBackup(cookies types.Cookies, regNo string) ([]types.Semester, error) {
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentCoursePage"
	var allSems []types.Semester
	bodyText, err := FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil {
		return allSems, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		return allSems, err
	}
	allSems, err = FindAndSaveSemIds(doc)
	if err != nil {
		return allSems, err
	}
	ReverseSlice(allSems)
	return allSems, nil
}

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) (types.Semester, error) {
	semDetails, err := GetSemDetails(cookies, regNo)
	var selectedSem types.Semester
	if err != nil {
		if debug.Debug {
			fmt.Println("Error featching sem details:", err)
		}
		semDetails, err = GetSemDetailsBackup(cookies, regNo)
		if err != nil {
			if debug.Debug {
				fmt.Println("Error fetching semester details in backup", err)
			}
			return selectedSem, err
		}
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

	choice := TableSelector("semester", nested_sem_list, strconv.Itoa(sem_choice))
	if choice.ExitRequest {
		return selectedSem, fmt.Errorf("selection canceled by user")
	}
	if !choice.Selected || choice.Index < 1 || choice.Index > len(semDetails) {
		return selectedSem, fmt.Errorf("invalid semester selection")
	}
	selectedSem = semDetails[choice.Index-1]

	_ = clearInputBuffer()
	return selectedSem, nil
}

func clearInputBuffer() error {
	for {

		if reader.Buffered() == 0 {
			return nil
		}

		b, err := reader.ReadByte()
		if err != nil {
			if debug.Debug {
				fmt.Println("Error clearing input buffer:", err)
			}
			return err
		}

		if b == '\n' {
			return nil
		}
	}
}
