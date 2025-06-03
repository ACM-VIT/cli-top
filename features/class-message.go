package features

import (
	"bytes"
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	MessageHeadingSelector = "h5"
	MessageBodySelector    = "div.panel-body"
)

func GetClassMessage(regNo string, cookies types.Cookies) {
	if !helpers.ValidateLogin(cookies) {
		return
	}
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

	if len(messages) == 1 {
		fmt.Println("No class messages found")
		return
	}
	fmt.Println()
	helpers.PrintTable(messages, 1)
	fmt.Println()
}

func extractClassMessages(bodyText []byte) ([][]string, error) {
	var messages [][]string
	messages = append(messages, []string{"Course", "Message"})
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyText))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}

	re := regexp.MustCompile(`^[A-Z0-9]+ - | - Online Course`)

	doc.Find(MessageHeadingSelector).Each(func(i int, h5 *goquery.Selection) {
		var row []string
		h5.Find("span").Each(func(i int, span *goquery.Selection) {
			trimmedText := strings.TrimSpace(span.Text())
			cleanedText := re.ReplaceAllString(trimmedText, "")
			cleanedText = strings.ReplaceAll(strings.TrimSpace(cleanedText), "\n", " ")

			var messageLines []string
			for len(cleanedText) > 60 {
				cutIndex := 60

				if spaceIndex := strings.LastIndex(cleanedText[:cutIndex], " "); spaceIndex != -1 {
					cutIndex = spaceIndex
				}
				messageLines = append(messageLines, cleanedText[:cutIndex])
				cleanedText = strings.TrimSpace(cleanedText[cutIndex:])
			}
			messageLines = append(messageLines, cleanedText)

			row = append(row, strings.Join(messageLines, "\n"))
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
