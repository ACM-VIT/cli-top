package helpers

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func StripAnsiCodes(str string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(str, "")
}

func TableSelector(subject string, nestedList [][]string, choice int) int {
	if choice != 0 {
		return choice
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("")
		PrintTable(nestedList, 1)
		fmt.Println("")
		fmt.Print("Choose a ", subject, ": ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid number.")
			continue
		}
		if choice < 1 || choice > len(nestedList)-1 {
			fmt.Println("Invalid choice.")
			continue
		}
		fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[choice][0])
		return choice
	}
}

func TableSelectorFuzzy(subject string, nestedList [][]string, choice string) int {

	if choice != "" {
		var matchedResults [][]string
		for _, v := range nestedList {
			combinedData := strings.Join(v[1:], " ")

			if strictFuzzyMatch(choice, combinedData) {
				matchedResults = append(matchedResults, []string{v[0], v[1], strings.Join(v[2:], " ")})
			}
		}

		if len(matchedResults) > 0 {
			fmt.Println("\nMatching results:")
			PrintTable(matchedResults, 1)

			for {
				fmt.Print("Choose an index from the matching results: ")
				var indexInput string
				_, err := fmt.Scanln(&indexInput)
				if err != nil {
					fmt.Println("\nInvalid input. Please enter a valid number.")
					continue
				}

				indexInput = strings.TrimSpace(indexInput)
				index, err := strconv.Atoi(indexInput)
				if err != nil || index < 1 || index > len(matchedResults) {
					fmt.Println("Invalid selection. Please enter a valid number.")
					continue
				}

				fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, strings.Join(matchedResults[index-1][1:], " "))
				return index - 1
			}
		} else {
			fmt.Println("No matching result found. Please try again.")
			return -1
		}
	}

	fmt.Println("")
	PrintTable(nestedList, 1)
	fmt.Println("")

	for {
		fmt.Print("Choose a ", subject, ": ")
		var userChoice string
		_, err := fmt.Scanln(&userChoice)
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid ", subject)
			continue
		}

		userChoice = strings.TrimSpace(userChoice)
		if userChoice == "" {
			fmt.Println("Please enter a valid choice.")
			continue
		}

		for i, v := range nestedList {
			combinedData := strings.Join(v[1:], " ")
			if strictFuzzyMatch(userChoice, combinedData) {
				fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, combinedData)
				return i
			}
		}

		fmt.Println("No matching result found. Please try again.")
	}
}

func strictFuzzyMatch(input string, data string) bool {
	input = strings.TrimSpace(input)

	return strings.Contains(data, input)
}

func PrintTable(nestedList [][]string, indexStatus int) int {
	if len(nestedList) == 0 {
		fmt.Println("Ummm are you sure you are printing the right thing?")
		return 1
	}

	// Convert all headers to uppercase
	for i, v := range nestedList[0] {
		nestedList[0][i] = strings.ToUpper(v)
	}

	// Determine the maximum number of columns
	maxCols := len(nestedList[0])

	normalizedList := make([][]string, 0, len(nestedList))

	// Normalize the number of columns in each row and append to the new list
	for _, row := range nestedList {
		if len(row) < maxCols {
			// Add empty strings to rows with fewer columns
			for len(row) < maxCols {
				row = append(row, "")
			}
		} else if len(row) > maxCols {
			// Truncate rows with more columns
			row = row[:maxCols]
		}
		normalizedList = append(normalizedList, row)
	}

	// Prepend 'INDEX' to the header row
	if indexStatus == 1 {
		normalizedList[0] = append([]string{"INDEX"}, normalizedList[0]...)

		// Add index numbers to the remaining rows
		for i := 1; i < len(normalizedList); i++ {
			index := fmt.Sprintf("%d", i)
			normalizedList[i] = append([]string{index}, normalizedList[i]...)
		}
	}

	// Compute the maximum width for each column
	maxwidth := make([]int, len(normalizedList[0]))
	for i, v := range normalizedList[0] {
		maxwidth[i] = len(StripAnsiCodes(v))
	}
	for _, row := range normalizedList {
		for j, v := range row {
			// Handle multiline cells and find the longest line
			for _, line := range strings.Split(v, "\n") {
				lineLength := len(StripAnsiCodes(line))
				if lineLength > maxwidth[j] {
					maxwidth[j] = lineLength
				}
			}
		}
	}

	// Define the right-align function for int values
	rightAlign := func(s string, width int) string {
		return fmt.Sprintf("%*s", width+len(s)-len(StripAnsiCodes(s)), s)
	}

	// Define the left-align function
	leftAlign := func(s string, width int) string {
		return fmt.Sprintf("%-*s", width+len(s)-len(StripAnsiCodes(s)), s)
	}

	// Build the format string for rows, including 3 leading spaces
	formatRow := "   " // Add 3 leading spaces
	for _, width := range maxwidth {
		formatRow += fmt.Sprintf(" %%-%ds │", width)
	}
	formatRow = strings.TrimSuffix(formatRow, " │") + "\n"

	// Print the header row
	headerRow := make([]interface{}, len(normalizedList[0]))
	for i, v := range normalizedList[0] {
		width := maxwidth[i]
		headerRow[i] = leftAlign(v, width) // Left-align the header
	}
	fmt.Printf(formatRow, headerRow...)

	// Build the separator line, including 3 leading spaces
	separator := "    " // Add 3 leading spaces
	for i, width := range maxwidth {
		separator += strings.Repeat("─", width)
		if i < len(maxwidth)-1 {
			separator += "─┼─"
		}
	}
	fmt.Println(separator)

	// Print the data rows
	for _, row := range normalizedList[1:] {
		// Split multiline cells and align each line properly
		lines := make([][]string, 0)
		maxLines := 1
		for _, cell := range row {
			cellLines := strings.Split(cell, "\n")
			if len(cellLines) > maxLines {
				maxLines = len(cellLines)
			}
			lines = append(lines, cellLines)
		}

		// Print each row, line by line
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			lineData := make([]interface{}, len(row))
			for colIdx, cellLines := range lines {
				line := ""
				if lineIdx < len(cellLines) {
					line = cellLines[lineIdx]
				}
				width := maxwidth[colIdx]
				if colIdx == 0 {
					if indexStatus == 1 {
						lineData[colIdx] = rightAlign(line, width)
					} else {
						lineData[colIdx] = leftAlign(line, width)
					}
				} else {
					lineData[colIdx] = leftAlign(line, width)
				}
			}
			fmt.Printf(formatRow, lineData...)
		}
	}
	return 0
}

func NewFuzzySearch(nestedList [][]string, stringFlag string) []int {
	var matchedResults []int
	for i, v := range nestedList {
		combinedData := strings.Join(v, " ")
		if FuzzyMatch(stringFlag, combinedData) {
			matchedResults = append(matchedResults, i)
		}
	}
	return matchedResults
}

func NewTableSelectorFuzzy(subject string, nestedList [][]string, choice string) int {
	if choice == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your choice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return -1
		}
		choice = strings.TrimSpace(input)
	}
	matchedResults := NewFuzzySearch(nestedList, choice)
	if len(matchedResults) > 0 {
		fmt.Println("\nMatching results:")
		PrintTable(nestedList, 1)

		for {
			fmt.Print("Choose an index from the matching results: ")
			var indexInput string
			_, err := fmt.Scanln(&indexInput)
			if err != nil {
				fmt.Println("Invalid input. Please enter a valid number.")
				continue
			}

			indexInput = strings.TrimSpace(indexInput)
			index, err := strconv.Atoi(indexInput)
			if err != nil || index < 1 || index > len(matchedResults) {
				fmt.Println("Invalid selection. Please enter a valid number.")
				continue
			}

			fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, strings.Join(nestedList[matchedResults[index-1]], " "))
			return matchedResults[index-1]
		}
	} else {
		fmt.Println("No matching result found. Please try again.")
		return -1
	}
}
