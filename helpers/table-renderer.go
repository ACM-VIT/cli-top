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
	// Remove standard ANSI CSI sequences (e.g. colors)
	reCSI := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	str = reCSI.ReplaceAllString(str, "")
	// Remove ANSI hyperlink sequences while keeping the visible text.
	// Matches the pattern: ESC ]8;;<url> BEL <visible text> ESC ]8;; BEL
	reHyper := regexp.MustCompile(`\x1b]8;;.*?\a(.*?)\x1b]8;;\a`)
	str = reHyper.ReplaceAllString(str, "$1")
	// Remove any leftover hyperlink initiators if present.
	reHyperIncomplete := regexp.MustCompile(`\x1b]8;;.*?\a`)
	str = reHyperIncomplete.ReplaceAllString(str, "")
	return str
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

	maxCols := len(nestedList[0])
	normalizedList := make([][]string, 0, len(nestedList))

	// Normalize rows
	for _, row := range nestedList {
		normalizedRow := make([]string, maxCols)
		copy(normalizedRow, row)
		normalizedList = append(normalizedList, normalizedRow)
	}

	// Add index column if needed
	if indexStatus == 1 {
		normalizedList[0] = append([]string{"INDEX"}, normalizedList[0]...)
		for i := 1; i < len(normalizedList); i++ {
			normalizedList[i] = append([]string{fmt.Sprintf("%d", i)}, normalizedList[i]...)
		}
	}

	// Get visible widths ignoring ANSI codes
	colWidths := make([]int, len(normalizedList[0]))
	for _, row := range normalizedList {
		for colIdx, cell := range row {
			// Handle multiline content
			for _, line := range strings.Split(cell, "\n") {
				visibleLen := len([]rune(StripAnsiCodes(line))) // Use rune length for UTF-8
				if visibleLen > colWidths[colIdx] {
					colWidths[colIdx] = visibleLen
				}
			}
		}
	}

	// Helper functions for alignment that preserve ANSI codes
	rightAlign := func(s string, width int) string {
		stripped := StripAnsiCodes(s)
		padding := width - len([]rune(stripped))
		if padding <= 0 {
			return s
		}
		return strings.Repeat(" ", padding) + s
	}

	leftAlign := func(s string, width int) string {
		stripped := StripAnsiCodes(s)
		padding := width - len([]rune(stripped))
		if padding <= 0 {
			return s
		}
		return s + strings.Repeat(" ", padding)
	}

	// Print header
	fmt.Print("   ") // Leading spaces
	for colIdx, headerCell := range normalizedList[0] {
		aligned := leftAlign(headerCell, colWidths[colIdx])
		fmt.Print(" ", aligned, " ")
		if colIdx < len(normalizedList[0])-1 {
			fmt.Print("│")
		}
	}
	fmt.Println()

	// Print separator
	fmt.Print("    ") // Leading spaces
	for colIdx, width := range colWidths {
		fmt.Print(strings.Repeat("─", width))
		if colIdx < len(colWidths)-1 {
			fmt.Print("─┼─")
		}
	}
	fmt.Println()

	// Print data rows
	for _, row := range normalizedList[1:] {
		// Handle multiline cells
		rowLines := make([][]string, 0)
		maxLines := 1
		for _, cell := range row {
			lines := strings.Split(cell, "\n")
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
			rowLines = append(rowLines, lines)
		}

		// Print each line of the row
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			fmt.Print("   ") // Leading spaces
			for colIdx, cellLines := range rowLines {
				line := ""
				if lineIdx < len(cellLines) {
					line = cellLines[lineIdx]
				}

				var aligned string
				if colIdx == 0 && indexStatus == 1 {
					aligned = rightAlign(line, colWidths[colIdx])
				} else {
					aligned = leftAlign(line, colWidths[colIdx])
				}

				fmt.Print(" ", aligned, " ")
				if colIdx < len(row)-1 {
					fmt.Print("│")
				}
			}
			fmt.Println()
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
