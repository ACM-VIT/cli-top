package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func fetchStudentDetails(cookies types.Cookies, regNo string) (types.StudentDetails, error) {
	if cookies.CSRF == "" || cookies.JSESSIONID == "" || cookies.SERVERID == "" {
		return types.StudentDetails{}, fmt.Errorf("Please login first using the cli-top login command")
	}
	url := "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView"
	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())

	body, err := helpers.FetchReq(regNo, cookies, url, "", payload, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Println("Error fetching student details:", err)
		}
		return types.StudentDetails{}, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		if debug.Debug {
			fmt.Println("Error parsing response body:", err)
		}
		return types.StudentDetails{}, err
	}

	registerNumber := doc.Find("label[for='no']").Text()
	programAndBranch := doc.Find("label[for='branchno']").Text()
	vitEmail := doc.Find("label[for='vmail']").Text()
	schoolName := doc.Find("label[for='schoolno']").Text()

	if registerNumber == "" || programAndBranch == "" || vitEmail == "" || schoolName == "" {
		return types.StudentDetails{}, fmt.Errorf("unable to fetch student details, check login config")
	}

	return types.StudentDetails{
		RegisterNumber: registerNumber,
		ProgramBranch:  programAndBranch,
		VITEmail:       vitEmail,
		SchoolName:     schoolName,
	}, nil
}

func Profile(cookies types.Cookies, regNo string) {
	studentDetails, err := fetchStudentDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error fetching profile: %v\n", err)
		} else {
			fmt.Println(err)
			fmt.Println()
			return
		}
	}

	tableData := [][]string{
		{"Field", "Information"},
		{"Register Number", studentDetails.RegisterNumber},
		{"Program & Branch", studentDetails.ProgramBranch},
		{"VIT Email", studentDetails.VITEmail},
		{"School Name", studentDetails.SchoolName},
	}
	fmt.Println()
	helpers.PrintTable(tableData, 0)
	fmt.Println()
}
