package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"strings"
)

// Function to fetch and extract faculty messages using session cookies and regNo
func GetClassMessage(regNo string, cookies types.Cookies) {
	// URL for the class messages
	url := "https://vtop.vit.ac.in/vtop/academics/common/StudentClassMessage"

	// Fetch the response body using helper functions (assuming helpers.FetchReq is implemented)
	bodyText, err := helpers.FetchReq(regNo, cookies, url, "", "UTC", "GET", "")
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}

	// Convert []byte to string before passing it to the extractClassMessages function
	messages, err := extractClassMessages(string(bodyText))
	if err != nil && debug.Debug {
		fmt.Println("Error extracting messages:", err)
		return
	}

	// Print the extracted messages in a table format
	printClassMessagesTable(messages)
}

// Function to extract class messages using string manipulation (no external packages)
func extractClassMessages(bodyText string) ([][]string, error) {
	var messages [][]string

	// Look for the starting and ending points of the relevant HTML for each message
	parts := strings.Split(bodyText, `<div class="panel panel-default">`)
	for _, part := range parts {
		if strings.Contains(part, `Course:`) && strings.Contains(part, `Message:`) {
			var row []string

			// Extract course name
			courseStart := strings.Index(part, `<span style="font-family: sans-serif;">`)
			courseEnd := strings.Index(part[courseStart:], "</span>")
			if courseStart != -1 && courseEnd != -1 {
				course := strings.TrimSpace(part[courseStart+len(`<span style="font-family: sans-serif;">`) : courseStart+courseEnd])
				row = append(row, course)
			}

			// Extract message content
			messageStart := strings.Index(part, `<b style="color:black;font-family: sans-serif;">Message:</b>`)
			messageEnd := strings.Index(part[messageStart:], "</span>")
			if messageStart != -1 && messageEnd != -1 {
				message := strings.TrimSpace(part[messageStart+len(`<b style="color:black;font-family: sans-serif;">Message:</b> <span style="font-family: sans-serif;">`) : messageStart+messageEnd])
				row = append(row, message)
			}

			// Add extracted row to messages list if both course and message were found
			if len(row) == 2 {
				messages = append(messages, row)
			}
		}
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no class messages found")
	}

	return messages, nil
}

// Function to print the class messages in a table format
func printClassMessagesTable(messages [][]string) {
	fmt.Printf("| %-35s | %-100s |\n", "Course", "Message")
	fmt.Printf("|%-37s|%-102s|\n", strings.Repeat("-", 37), strings.Repeat("-", 102))

	for _, row := range messages {
		fmt.Printf("| %-35s | %-100s |\n", row[0], row[1])
	}
}
