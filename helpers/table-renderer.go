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

// FuzzySearchFunc is a function type for custom fuzzy search implementations
type FuzzySearchFunc func([][]string, string) []int

// SelectionResult represents the result of a table selection operation
type SelectionResult struct {
	Index       int
	Selected    bool
	ExitRequest bool
}

// TableSelector handles selection from a table with support for direct selection only
func TableSelector(subject string, nestedList [][]string, initialQuery string) SelectionResult {
	// If initial query is a number, try to select it directly
	if isNumeric(initialQuery) {
		choice, _ := strconv.Atoi(initialQuery)
		if choice >= 1 && choice <= len(nestedList)-1 {
			fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[choice][0])
			return SelectionResult{Index: choice, Selected: true}
		}
	} else if initialQuery == "exit" {
		return SelectionResult{ExitRequest: true}
	}

	reader := bufio.NewReader(os.Stdin)

	// Initial display of the table
	fmt.Println("")
	PrintTable(nestedList, 1)
	fmt.Println("")

	for {
		fmt.Printf("Choose a %s (enter a number): ", subject)
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		if choice == "exit" {
			return SelectionResult{ExitRequest: true}
		}

		// Check if input is a number
		choiceNum, err := strconv.Atoi(choice)
		if err != nil {
			fmt.Printf("Invalid input. Please enter a number between 1 and %d.\n", len(nestedList)-1)
			// Reprint the table after invalid input
			fmt.Println("")
			PrintTable(nestedList, 1)
			fmt.Println("")
			continue
		}

		// Validate the choice
		if choiceNum >= 1 && choiceNum <= len(nestedList)-1 {
			fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[choiceNum][0])
			return SelectionResult{Index: choiceNum, Selected: true}
		} else {
			fmt.Printf("Invalid choice. Please enter a number between 1 and %d.\n", len(nestedList)-1)
			// Reprint the table after invalid input
			fmt.Println("")
			PrintTable(nestedList, 1)
			fmt.Println("")
		}
	}
}

// TableSelectorFuzzy handles selection with support for fuzzy search
func TableSelectorFuzzy(subject string, nestedList [][]string, initialQuery string, fuzzySearchFunc FuzzySearchFunc) SelectionResult {
	// If initial query is numeric, use direct selection
	if isNumeric(initialQuery) {
		choice, _ := strconv.Atoi(initialQuery)
		if choice >= 1 && choice <= len(nestedList)-1 {
			fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[choice][0])
			return SelectionResult{Index: choice, Selected: true}
		} else {
			// Invalid numeric initial query, will display table and prompt for new input
			fmt.Printf("Invalid number. Please enter a number between 1 and %d.\n", len(nestedList)-1)
			initialQuery = ""
		}
	} else if initialQuery == "exit" {
		return SelectionResult{ExitRequest: true}
	}

	reader := bufio.NewReader(os.Stdin)
	searchQuery := initialQuery

	// If no initial query provided or if initial query was an invalid number, prompt for input
	if searchQuery == "" {
		if subject == "Course" {
			fmt.Printf("Enter the course name to download the syllabus (or 'exit' to quit): ")
		} else {
			fmt.Println("")
			PrintTable(nestedList, 1)
			fmt.Println("")
			fmt.Printf("Enter a search term or number for %s (or 'exit' to quit): ", subject)
		}
		input, _ := reader.ReadString('\n')
		searchQuery = strings.TrimSpace(input)

		if searchQuery == "exit" {
			return SelectionResult{ExitRequest: true}
		}
	}

	for {
		// Check if query is a direct number selection
		if num, err := strconv.Atoi(searchQuery); err == nil {
			if num >= 1 && num <= len(nestedList)-1 {
				fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[num][0])
				return SelectionResult{Index: num, Selected: true}
			} else {
				fmt.Printf("Invalid number. Please enter a number between 1 and %d.\n", len(nestedList)-1)
				// Reprint the table after invalid input
				fmt.Println("")
				PrintTable(nestedList, 1)
				fmt.Println("")
				fmt.Printf("Enter a search term or number for %s (or 'exit' to quit): ", subject)
				input, _ := reader.ReadString('\n')
				searchQuery = strings.TrimSpace(input)
				if searchQuery == "exit" {
					return SelectionResult{ExitRequest: true}
				}
				continue // Skip the fuzzy search for invalid numeric input
			}
		}

		// Only perform fuzzy search for non-numeric input
		// Use provided fuzzy search function or default to NewFuzzySearch
		var selectedIndices []int
		if fuzzySearchFunc != nil {
			selectedIndices = fuzzySearchFunc(nestedList, searchQuery)
		} else {
			selectedIndices = NewFuzzySearch(nestedList, searchQuery)
		}

		if len(selectedIndices) == 0 {
			fmt.Println("No matching results found for your query.")
			// Reprint the table after no results found
			fmt.Println("")
			PrintTable(nestedList, 1)
			fmt.Println("")
			fmt.Printf("Enter a new search term or number for %s (or 'exit' to quit): ", subject)
			input, _ := reader.ReadString('\n')
			searchQuery = strings.TrimSpace(input)
			if searchQuery == "exit" {
				return SelectionResult{ExitRequest: true}
			}
			continue
		} else if len(selectedIndices) == 1 {
			index := selectedIndices[0]
			if index < 1 || index > len(nestedList)-1 {
				fmt.Println("Selected index is out of range.")
				// Reprint the table after invalid selection
				fmt.Println("")
				PrintTable(nestedList, 1)
				fmt.Println("")
				fmt.Printf("Enter a new search term or number for %s (or 'exit' to quit): ", subject)
				input, _ := reader.ReadString('\n')
				searchQuery = strings.TrimSpace(input)
				if searchQuery == "exit" {
					return SelectionResult{ExitRequest: true}
				}
				continue
			}
			fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[index][0])
			return SelectionResult{Index: index, Selected: true}
		} else {
			// Multiple matches found, create a filtered list
			filteredList := [][]string{nestedList[0]} // Keep the header
			filteredIndices := []int{}                // Track which original indices correspond to each filtered row
			for _, index := range selectedIndices {
				if index >= 1 && index <= len(nestedList)-1 {
					filteredList = append(filteredList, nestedList[index])
					filteredIndices = append(filteredIndices, index)
				}
			}

			if len(filteredList) <= 1 {
				fmt.Println("No valid results found in the matched items.")
				// Reprint the table after no valid results
				fmt.Println("")
				PrintTable(nestedList, 1)
				fmt.Println("")
				fmt.Printf("Enter a new search term or number for %s (or 'exit' to quit): ", subject)
				input, _ := reader.ReadString('\n')
				searchQuery = strings.TrimSpace(input)
				if searchQuery == "exit" {
					return SelectionResult{ExitRequest: true}
				}
				continue
			}

			fmt.Println("\nMultiple matches found. Please select from the results below:")
			fmt.Println("")
			PrintTable(filteredList, 1)
			fmt.Println("")

			// Loop until valid selection from filtered results
			for {
				fmt.Printf("Choose a %s (number) or type 'search' for a new search: ", subject)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if input == "exit" {
					return SelectionResult{ExitRequest: true}
				}

				if input == "search" {
					fmt.Println()
					PrintTable(nestedList, 1)
					fmt.Println()
					fmt.Printf("Enter a new search term or number for %s (or 'exit' to quit): ", subject)
					input, _ := reader.ReadString('\n')
					searchQuery = strings.TrimSpace(input)
					if searchQuery == "exit" {
						return SelectionResult{ExitRequest: true}
					}
					break // Break inner loop to return to outer loop with new search
				}

				choice, err := strconv.Atoi(input)
				if err != nil {
					fmt.Println("Invalid input. Please enter a valid number.")
					// Reprint the filtered table after invalid input
					fmt.Println("")
					PrintTable(filteredList, 1)
					fmt.Println("")
					continue
				}

				if choice < 1 || choice > len(filteredList)-1 {
					fmt.Printf("Invalid choice. Please enter a number between 1 and %d.\n", len(filteredList)-1)
					// Reprint the filtered table after invalid choice
					fmt.Println("")
					PrintTable(filteredList, 1)
					fmt.Println("")
					continue
				}

				// Map the selection back to the original index directly from our mapping
				originalIndex := filteredIndices[choice-1]
				fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n", subject, nestedList[originalIndex][0])
				return SelectionResult{Index: originalIndex, Selected: true}
			}
		}
	}
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

// isNumeric checks if a string can be parsed as an integer
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
