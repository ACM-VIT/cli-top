package features

import (
	//"bytes"
	//"io"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"regexp"

	//"net/http"
	"strings"
	//"time"

	"github.com/PuerkitoBio/goquery"
)


func GetClassMessage(regNo string, cookies types.Cookies) {

	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentClassMessage"
	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "UTC", "POST", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}

	messages, err := extractClassMessages(bodyText)
	if err != nil && debug.Debug {
		fmt.Println("Error extracting messages:", err)
		return
	}
	fmt.Println()	
	helpers.PrintTable(messages)
	fmt.Println()
}


func extractClassMessages(bodyText []byte) ([][]string, error) {
	var messages [][]string
	messages = append(messages, []string{"Course", "Message"})
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}

    re := regexp.MustCompile(`^[A-Z0-9]+ - | - Online Course`)

    doc.Find("h5").Each(func(i int, h5 *goquery.Selection) {
        var row []string
        h5.Find("span").Each(func(i int, span *goquery.Selection) {
			trimmedText := strings.TrimSpace(span.Text())
            cleanedText := re.ReplaceAllString(trimmedText, "")
            cleanedText = strings.ReplaceAll(cleanedText, "\n", " ")
            if len(cleanedText) > 60 {
                cleanedText = cleanedText[:60] + "..."
            }
            row = append(row, cleanedText)
        })
		if len(row) == 2 {
			messages = append(messages, row)
		}
    })

	if len(messages) == 0 {
		return nil, fmt.Errorf("no class messages found")
	}
	return messages, nil
}
