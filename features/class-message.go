package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"
	"github.com/PuerkitoBio/goquery"
)


func GetClassMessage(regNo string, cookies types.Cookies) {
	
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentClassMessage"

	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "UTC", "GET", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}

	messages, err := extractClassMessages(bodyText)
	if err != nil && debug.Debug {
		fmt.Println("Error extracting messages:", err)
		return
	}

	printClassMessagesTable(messages)
}


func extractClassMessages(bodyText []byte) ([][]string, error) {
	var messages [][]string

	
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}

	
	doc.Find(".panel.panel-default").Each(func(i int, panel *goquery.Selection) {
		var row []string

		
		course := panel.Find("span:contains('Course:')").Next().Text()
		course = strings.TrimSpace(course)

		
		message := panel.Find("b:contains('Message:')").Next().Text()
		message = strings.TrimSpace(message)

		
		if course != "" && message != "" {
			row = append(row, course, message)
			messages = append(messages, row)
		}
	})

	
	if len(messages) == 0 {
		return nil, fmt.Errorf("no class messages found")
	}

	return messages, nil
}


func printClassMessagesTable(messages [][]string) {
	fmt.Printf("| %-35s | %-100s |\n", "Course", "Message")
	fmt.Printf("|%-37s|%-102s|\n", strings.Repeat("-", 37), strings.Repeat("-", 102))

	for _, row := range messages {
		fmt.Printf("| %-35s | %-100s |\n", row[0], row[1])
	}
}
