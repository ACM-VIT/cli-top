package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
)

func extractIDFromHTML(html string) (string, error) {
	// Define a regular expression to match the assignment of id variable
	re := regexp.MustCompile(`let id\s*=\s*"(.*?)";`)

	// Find the first match
	match := re.FindStringSubmatch(html)
	if len(match) != 2 {
		return "", fmt.Errorf("unable to extract id from HTML")
	}

	return match[1], nil
}

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/content", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Set the request headers
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("cookie", "JSESSIONID=2A4163F31950BCAFAC67F09840405FF5; _ga=GA1.1.1763604292.1699285434; _hp2_id.3863792970=%7B%22userId%22%3A%224954569497889160%22%2C%22pageviewId%22%3A%226755249576479608%22%2C%22sessionId%22%3A%224822149987821438%22%2C%22identity%22%3Anull%2C%22trackerVersion%22%3A%224.0%22%7D; SSESS8ae6d8dd4c6fcb35daaa3386129c796f=6bW9uis65_FQlanzBTpnxZEmYNgGl9AFDbwGPSCfT1s; _ga_VY3TWN1FJ7=GS1.1.1704387383.3.1.1704387416.27.0.0; SERVERID=s1")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Convert the body text to a string
	html := string(bodyText)

	// Extract the id from the HTML
	id, err := extractIDFromHTML(html)
	if err != nil {
		log.Fatal(err)
	}

	// Print the extracted id as RegNo
	fmt.Printf("RegNo: %s\n", id)
}
