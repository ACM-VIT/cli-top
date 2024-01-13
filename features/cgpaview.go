package features

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	// "net/url"
	"strings"
	"time"
	"vtop-cli/types"
	// "github.com/charmbracelet/glamour"
	"github.com/olekukonko/tablewriter"


	"github.com/PuerkitoBio/goquery"
	// "golang.org/x/net/html"
)

func FetchReq1(regNo string, cookies types.Cookies, url string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request

	payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
	//fmt.Println(payload)
	// Create a new request with POST method and payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	// Set headers or cookies if needed
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	// fmt.Println("response body:",string(body))

	return body, nil
}

func PrintCgpa(regNo string, cookies types.Cookies, url string) {
	body, err := FetchReq1(regNo, cookies, url)
	if err != nil {
		log.Fatal("Error fetching CGPA data:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	fmt.Println("CGPA Details")

	// Extract and print data from the specified HTML structure
	table := doc.Find("div.table-responsive table.table tbody tr")
	row := table.First()

	// Extract data from each column
	creditsRegistered := row.Find("td").Eq(0).Text()
	creditsEarned := row.Find("td").Eq(1).Text()
	cgpa := row.Find("td").Eq(2).Text()
	sGrades := row.Find("td").Eq(3).Text()
	aGrades := row.Find("td").Eq(4).Text()
	bGrades := row.Find("td").Eq(5).Text()
	cGrades := row.Find("td").Eq(6).Text()
	dGrades := row.Find("td").Eq(7).Text()
	eGrades := row.Find("td").Eq(8).Text()
	fGrades := row.Find("td").Eq(9).Text()
	nGrades := row.Find("td").Eq(10).Text()

	// Create a table
	tableData := [][]string{
		{"Credits Registered", creditsRegistered},
		{"Credits Earned", creditsEarned},
		{"CGPA", fmt.Sprintf("\033[32m%s\033[0m", cgpa)}, // Highlight CGPA in green
		{"S Grades", sGrades},
		{"A Grades", aGrades},
		{"B Grades", bGrades},
		{"C Grades", cGrades},
		{"D Grades", dGrades},
		{"E Grades", eGrades},
		{"F Grades", fGrades},
		{"N Grades", nGrades},
	}

	tableObj := tablewriter.NewWriter(log.Writer())

	// Append data to the table
	for _, data := range tableData {
		tableObj.Append(data)
	}

	// Render the table
	tableObj.Render()
}
