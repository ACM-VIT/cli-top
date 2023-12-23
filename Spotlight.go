package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)
const listingsPerPage = 5

func main() {
	// Prepare form data
	formData := prepareFormData("22BDS0188")

	// Prepare the request
	req, err := prepareRequest(formData)
	if err != nil {
		log.Fatal(err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Parse and print the specific HTML structure
	parseAndPrintSpecificHTML(resp.Body)

}

func prepareFormData(authorizedID string) url.Values {
	formData := url.Values{}
	formData.Set("verifyMenu", "true")
	formData.Set("authorizedID", authorizedID)
	formData.Set("_csrf", "a557f650-9034-44e5-9d0d-c426f714ea0a")
	formData.Set("nocache", fmt.Sprintf("@%d", time.Now().UnixNano()/int64(time.Millisecond)))
	return formData
}

func prepareRequest(formData url.Values) (*http.Request, error) {
	req, err := http.NewRequest(
		"POST",
		"https://vtop.vit.ac.in/vtop/content",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, err
	}

	// Set headers (unchanged)
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Cookie", "JSESSIONID=1850C53AEB453380FC81EC410FA2B415; SERVERID=s2")
	req.Header.Set("Sec-Ch-Ua", ` "Not_A Brand";v="8", "Chromium";v="120" `)
	req.Header.Set("Accept", "/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.71 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua-Platform", "Windows")
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

func parseAndPrintSpecificHTML(body io.Reader) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	var listings []listingItem
	pageCount := 1

	// Find and store the specific HTML structure
	doc.Find("div.card-body div.row div.col-sm-3 div.d-flex.flex-column").Each(func(i int, cardSelection *goquery.Selection) {
		// Extract data from the button (unchanged)

		// Extract data from the associated list
		cardSelection.Find("ul.list-group li").Each(func(k int, listSelection *goquery.Selection) {
			listItemText := strings.TrimSpace(listSelection.Text())
			listItemEndpoint, _ := listSelection.Find("a").Attr("href")
			listings = append(listings, listingItem{
				Text:     listItemText,
				Endpoint: listItemEndpoint,
			})
		})
	})

	// Display listings for the first page
	displayListings(listings, pageCount)

	// Ask the user for input to navigate between pages
	for {
		totalPages := (len(listings)-1)/listingsPerPage + 1
		fmt.Printf("Enter page number (1-%d) or 'q' to quit: ", totalPages)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "q" {
			break
		}

		pageNumber, err := strconv.Atoi(input)
		if err != nil || pageNumber < 1 || pageNumber > ((len(listings)-1)/listingsPerPage)+1 {
			fmt.Println("Invalid input. Please enter a valid page number.")
			continue
		}

		// Display listings for the selected page
		pageCount = pageNumber
		displayListings(listings, pageCount)
	}
}

func displayListings(listings []listingItem, page int) {
	start := (page - 1) * listingsPerPage
	end := start + listingsPerPage

	fmt.Printf("--- Page %d ---\n", page)
	for i := start; i < end && i < len(listings); i++ {
		fmt.Printf("    %d. %s\n", i+1, listings[i].Text)
		if listings[i].Endpoint == "javascript:void(0);" {
			fmt.Println("       Please use the web interface (VTOP) to access this entry")
		} else {
			fmt.Printf("       Link: %s\n", listings[i].Endpoint)
		}
		fmt.Println() // Add a newline after each listing
	}

	fmt.Println("--- End of Page ---\n")
}

type listingItem struct {
	Text     string
	Endpoint string
}
