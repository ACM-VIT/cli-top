// features/profile.go

package features

import (
	"bytes"
	types "cli-top/types"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func fetchStudentDetails(cookies types.Cookies, regNo string) (types.StudentDetails, error) {
	client := &http.Client{}
	url := "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView"
	data := strings.NewReader(fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=@(new Date().getTime())", regNo, cookies.CSRF))

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return types.StudentDetails{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/118.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://vtop.vit.ac.in")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/content?")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("TE", "trailers")

	resp, err := client.Do(req)
	if err != nil {
		return types.StudentDetails{}, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
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
		log.Fatal(err)
	}

	markdownTable := generateStudentDetailsMarkdownTable(studentDetails)

	rendered, err := glamour.Render(markdownTable, "dark")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(rendered)
}

func generateStudentDetailsMarkdownTable(details types.StudentDetails) string {
	var buf bytes.Buffer

	buf.WriteString("| Field            | Information                                                    |\n")
	buf.WriteString("|------------------|----------------------------------------------------------------|\n")

	buf.WriteString(fmt.Sprintf("| Register Number  | %-62s |\n", details.RegisterNumber))
	buf.WriteString(fmt.Sprintf("| Program & Branch | %-62s |\n", details.ProgramBranch))
	buf.WriteString(fmt.Sprintf("| VIT Email        | [%-62s]() \n", details.VITEmail))
	buf.WriteString(fmt.Sprintf("| School Name      | %-62s |\n", details.SchoolName))

	return buf.String()
}
