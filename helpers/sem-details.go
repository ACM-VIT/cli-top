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

	"golang.org/x/net/html"
)

type selectCandidate struct {
	id      string
	name    string
	class   string
	options []types.Semester
}

// Initialize a single reader instance for the package
var reader = bufio.NewReader(os.Stdin)

func FindAndSaveSemIds(body []byte) ([]types.Semester, error) {
	var (
		candidates []selectCandidate
		currentSel *selectCandidate
		inOption   bool
		optionVal  string
		textBuf    strings.Builder
	)

	z := html.NewTokenizer(bytes.NewReader(body))
	for {
		switch z.Next() {
		case html.ErrorToken:
			if len(candidates) == 0 {
				if debug.Debug {
					fmt.Println("No semesters found in document.")
				}
				return nil, fmt.Errorf("no semesters found")
			}
			chosen := pickSemesterSelect(candidates)
			if len(chosen.options) == 0 {
				return nil, fmt.Errorf("no semesters found")
			}
			return chosen.options, nil
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			switch string(tagName) {
			case "select":
				if currentSel != nil {
					candidates = append(candidates, *currentSel)
				}
				currentSel = &selectCandidate{}
				if hasAttr {
					for {
						key, val, more := z.TagAttr()
						switch string(key) {
						case "id":
							currentSel.id = string(val)
						case "name":
							currentSel.name = string(val)
						case "class":
							currentSel.class = string(val)
						}
						if !more {
							break
						}
					}
				}
			case "option":
				if currentSel == nil {
					break
				}
				inOption = true
				optionVal = ""
				textBuf.Reset()
				if hasAttr {
					for {
						key, val, more := z.TagAttr()
						if string(key) == "value" {
							optionVal = string(val)
						}
						if !more {
							break
						}
					}
				}
			}
		case html.TextToken:
			if inOption {
				textBuf.Write(z.Text())
			}
		case html.EndTagToken:
			tagName, _ := z.TagName()
			switch string(tagName) {
			case "option":
				if inOption && currentSel != nil {
					text := strings.TrimSpace(textBuf.String())
					if optionVal != "" && text != "" {
						currentSel.options = append(currentSel.options, types.Semester{SemID: optionVal, SemName: text})
					}
				}
				inOption = false
			case "select":
				if currentSel != nil {
					candidates = append(candidates, *currentSel)
					currentSel = nil
				}
			}
		}
	}
}

func pickSemesterSelect(candidates []selectCandidate) selectCandidate {
	for _, c := range candidates {
		if len(c.options) == 0 {
			continue
		}
		if strings.Contains(c.id, "semesterSubId") || strings.Contains(c.name, "semesterSubId") {
			return c
		}
		if strings.Contains(c.class, "form-select") {
			return c
		}
	}
	for _, c := range candidates {
		if len(c.options) > 0 {
			return c
		}
	}
	return selectCandidate{}
}

// GetSemDetails fetches semester details
func GetSemDetails(cookies types.Cookies, regNo string) ([]types.Semester, error) {
	if cached, ok := getCachedSemesters(regNo); ok {
		return cached, nil
	}
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

	allSems, err = FindAndSaveSemIds(bodyText)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching semester details", err)
		}
		return allSems, err
	}
	ReverseSlice(allSems)
	storeSemesters(regNo, allSems)
	return allSems, nil
}

func GetSemDetailsBackup(cookies types.Cookies, regNo string) ([]types.Semester, error) {
	if cached, ok := getCachedSemesters(regNo); ok {
		return cached, nil
	}
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentCoursePage"
	var allSems []types.Semester
	bodyText, err := FetchReq(regNo, cookies, url, "", "", "POST", "")
	if err != nil {
		return allSems, err
	}
	allSems, err = FindAndSaveSemIds(bodyText)
	if err != nil {
		return allSems, err
	}
	ReverseSlice(allSems)
	storeSemesters(regNo, allSems)
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
