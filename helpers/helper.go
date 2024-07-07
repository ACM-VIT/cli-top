package helpers

import (
	"cli-top/debug"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

//payload := []byte("_csrf=154a792d-e0d1-42fb-8300-c4211db46510&semesterSubId=VL20232405&authorizedID=22BCI0272&x=" + time.Now().UTC().Format(time.RFC1123))

func getTextContent(n *html.Node) string {
	var textContent string

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			textContent += c.Data
		} else if c.Type == html.ElementNode {
			textContent += getTextContent(c)
		}
	}

	return textContent
}

func printTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-12s | %-20s | %-27s | %-16s | %-10s | %-18s |\n",
		"S.No.", "Course Code", "Slot No.", "Faculty Name", "Classes Attended", "Percentage", "75% Alert"))
	builder.WriteString("|-------|--------------|----------------------|-----------------------------|------------------|------------|--------------------|\n")

}

func strToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil && debug.Debug {
		fmt.Println("Error converting string to integer:", err)

	}

	return num
}

func FindOptionWithTagValue(doc *goquery.Document, targetValue string) string {
	return doc.Find("option[value='" + targetValue + "']").Text()
}

func RemoveEmptyStrings(data []string) []string {
	var cleanedData []string
	for _, item := range data {
		if item != "" {
			cleanedData = append(cleanedData, item)
		}
	}
	return cleanedData
}
