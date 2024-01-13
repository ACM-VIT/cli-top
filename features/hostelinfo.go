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

	"github.com/PuerkitoBio/goquery"
	// "golang.org/x/net/html"
)

func FetchReq(regNo string, cookies types.Cookies, url string) ([]byte, error) {
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

func PrintHostelInfo(regNo string, cookies types.Cookies, url string) {
	body, err := FetchReq(regNo, cookies, url)
	if err != nil {
		log.Fatal("Error fetching HTML:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	// fmt.Println("+-----------------------------+------------------------------------------------------+")
	fmt.Println("Student Accommodation Info")
	fmt.Println("+-----------------------------+------------------------------------------------------+")

	table := doc.Find("div.table-responsive table.table tbody tr")
	lastFiveRows := table.Slice(-5, table.Length())

	lastFiveRows.Each(func(j int, rowSelection *goquery.Selection) {
		header := rowSelection.Find("td").Eq(0).Text()
		value := rowSelection.Find("td").Eq(1).Text()

		fmt.Printf("| %-27s | %-52s |\n", strings.TrimSpace(header), strings.TrimSpace(value))
	})

	fmt.Println("+-----------------------------+------------------------------------------------------+")
}