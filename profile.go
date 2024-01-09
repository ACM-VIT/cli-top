package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	// Prepare form data
	formData := prepareFormData("22BCT0355")

	// Prepare the request
	req, err := prepareRequest(formData)
	if err != nil {
		log.Fatal(err)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Print the response body or process it further
	parseAndPrintLastFiveLines(resp.Body)

}

func prepareFormData(authorizedID string) url.Values {
	formData := url.Values{}
	formData.Set("verifyMenu", "true")
	formData.Set("authorizedID", authorizedID)
	formData.Set("_csrf", "bdb6bf5c-dc8c-413e-a9ef-6581de3aa967")
	formData.Set("nocache", fmt.Sprintf("@%d", time.Now().UnixNano()/int64(time.Millisecond)))
	return formData
}

func prepareRequest(formData url.Values) (*http.Request, error) {
	req, err := http.NewRequest(
		"POST",
		"https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, err
	}

	// Set headers (unchanged)
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Cookie", "JSESSIONID=7FBB4E62A9DC8A4B8968DEF7C25EFF36; SERVERID=s2")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120"`)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.71 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Origin", "https://vtop.vit.ac.in")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/content?")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=1, i")

	return req, nil
}

func parseAndPrintLastFiveLines(body io.Reader) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	// Find and print data from the HTML
	fmt.Println("Student Accommodation Information:")

	// Extract and print data from the specified HTML structure
	table := doc.Find("div.table-responsive table.table tbody tr")
	lastFiveRows := table.Slice(-5, table.Length())

	lastFiveRows.Each(func(j int, rowSelection *goquery.Selection) {
		// Extract and print data from each row
		header := rowSelection.Find("td[style*='font-weight:bold;']").Text()
		value := rowSelection.Find("td[style*='background-color']").Text()

		// Format the output with clear spacing and indentation
		fmt.Printf("%-25s: %s\n", strings.TrimSpace(header), strings.TrimSpace(value))
	})
}