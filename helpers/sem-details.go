package helpers

import (
    "cli-top/debug"
    "cli-top/types"
    "fmt"
    "strings"


    "github.com/PuerkitoBio/goquery"
)

// FindAndSaveSemIds finds and saves semester IDs from the document
func FindAndSaveSemIds(doc *goquery.Document) ([]types.Semester, error) {
    var allsems []types.Semester
    doc.Find("select.form-select option").Each(func(i int, s *goquery.Selection) {
        var tempSem types.Semester
        var exists bool
        tempSem.SemID, exists = s.Attr("value")
        tempSem.SemName = s.Text()
        if exists {
            if tempSem.SemID != "" {
                allsems = append(allsems, tempSem)
            }
        }
    })
    if len(allsems) == 0 {
        return nil, fmt.Errorf("no semesters found")
    }
    return allsems, nil
}

// GetSemDetails fetches semester details
func GetSemDetails(cookies types.Cookies, regNo string) ([]types.Semester, error) {
    url := "https://vtop.vit.ac.in/vtop/academics/common/StudentAttendance"
    var allSems []types.Semester
    bodyText, err := FetchReq(regNo, cookies, url, "", "", "POST", "")
    if err != nil {
        if debug.Debug {
            fmt.Println("Error fetching semester details:", err)
        }
        return allSems, err
    }

    if debug.Debug {
        fmt.Println("---- Response Body Start ----")
        fmt.Println(string(bodyText))
        fmt.Println("---- Response Body End ----")
    }

    doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
    if err != nil {
        if debug.Debug {
            fmt.Println("Error parsing the HTML document:", err)
        }
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
        return selectedSem, err
    }
    if len(semDetails) == 0 {
        return selectedSem, fmt.Errorf("error fetching semester details or no semesters available. Try logging out and logging back in")
    }

    clearInputBuffer()

    var nested_sem_list [][]string
    nested_sem_list = append(nested_sem_list, []string{"SemId", "SemName"})
    for i := 0; i < len(semDetails); i++ {
        nested_sem_list = append(nested_sem_list, []string{semDetails[i].SemID, semDetails[i].SemName})
    }

    choice := TableSelector("semester", nested_sem_list, sem_choice)
    if choice == -1 {
        return selectedSem, fmt.Errorf("invalid semester selection")
    }
    if choice < 1 || choice > len(semDetails) {
        return selectedSem, fmt.Errorf("invalid semester selection")
    }
    selectedSem = semDetails[choice-1]

    return selectedSem, nil
}

func clearInputBuffer() {
	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	if err != nil && debug.Debug {
		fmt.Println("Error clearing input buffer:", err)
	}
}
