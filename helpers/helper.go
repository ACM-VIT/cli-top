package helpers

import (
	"cli-top/types"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	"golang.org/x/net/html"
)

//payload := []byte("_csrf=154a792d-e0d1-42fb-8300-c4211db46510&semesterSubId=VL20232405&authorizedID=22BCI0272&x=" + time.Now().UTC().Format(time.RFC1123))

func FindAndSaveSemIds(doc *goquery.Document, targetClass string, result *[]string) {
	// Find all <option> elements within <select> tags with the specified class
	//fmt.Println(targetClass, doc)
	//selection := doc.Find("select.form-select option")
	//fmt.Println("Number of elements found:", selection.Length())

	doc.Find("select." + targetClass + " option").Each(func(i int, s *goquery.Selection) {

		value, exists := s.Attr("value")
		//fmt.Println(value)
		if exists {
			*result = append(*result, value)
		}
	})
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
	if err != nil {
		fmt.Println("Error converting string to integer:", err)

	}

	return num
}

func SelectSemester(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	fmt.Print("\nEnter the index of the semester to view attendance: ")
	fmt.Scanln(&choice)
	semDet := GetSemDetails(cookies, regNo)

	if choice < 1 || choice > len(semDet.SemIds) {
		fmt.Println("Invalid choice.")
	} else {
		for i, id := range semDet.SemIds {
			if i+1 == choice {
				// fmt.Println("Selected sem id : ",id)
				// fmt.Println("Selected sem name : ",semDet.SemNames[i])
				selectedSemId = id
				selectedSemName = semDet.SemNames[i]
			}
		}
	}

	// Format the string with glamour
	formattedSelection := fmt.Sprintf("\n# You selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)

	// Render and print the formatted string
	renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
	if err != nil {
		log.Fatal("Error creating glamour renderer:", err)
	}

	output, err := renderer.Render(formattedSelection)
	if err != nil {
		log.Fatal("Error rendering formatted string:", err)
	}

	fmt.Print(output)

	fmt.Println()

	return selectedSemId

}
