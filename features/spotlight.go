//features spotlight.go

package features

import (
	"fmt"
	"io"
	"log"
	"strings"
	"vtop-cli/types"

	"golang.org/x/net/html"
)

//Function to get Spotlight Details
func GetSpotlightDetails(cookies types.Cookies) {
	url := "https://vtop.vit.ac.in/vtop/content"
	fmt.Println("hello")
	bodyText, err := fetchReq("", "GET", cookies, url, "")
	if err != nil {
		log.Fatal(err)
	}
	
	// Use goquery to parse the HTML
	// doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(bodyText))
	extractText(string(bodyText))

}

func extractText(bodyText string) string {
	fmt.Println(bodyText)
	reader := strings.NewReader(bodyText)

	// Read from the reader and print its content as a string
	data, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("Error:", err)
 
	}

	fmt.Println(string(data))
	tokenizer := html.NewTokenizer(reader)

	var result strings.Builder

	for {
		tokenType := tokenizer.Next()
		// fmt.Println("hellowuwubdba")
		switch tokenType {
		case html.ErrorToken:
			return result.String()
		case html.TextToken:
			text := strings.TrimSpace(html.UnescapeString(string(tokenizer.Text())))
			if text != "" {
				result.WriteString(text + " ")
			}
		}
	}
}

