package helpers

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"regexp"

	"github.com/olekukonko/tablewriter"
	"github.com/charmbracelet/glamour"

	"cli-top/debug"
	"cli-top/types"
)

func RemoveCourseCode(courseName string) string {
	re := regexp.MustCompile(`^[A-Z]{4}\d{3}[A-Z]?\s*-\s*`)
	return re.ReplaceAllString(courseName, "")
}

func TruncateString(str string, maxLength int) string {
	if len(str) <= maxLength {
		return str
	}
	if maxLength <= 3 {
		return str[:maxLength]
	}
	return str[:maxLength-3] + "..."
}

func HighlightMatches(text, query string) string {
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	return re.ReplaceAllStringFunc(text, func(match string) string {
		return "\033[1m" + match + "\033[0m"
	})
}

func GenerateCourseDetailsTable(courses []types.Course) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"INDEX", "COURSE NAME"})
	table.SetBorder(true)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColWidth(60)

	for i, course := range courses {
		index := fmt.Sprintf("%d", i+1)
		courseName := RemoveCourseCode(course.Name)
		courseName = strings.ReplaceAll(courseName, "\n", " ")
		courseName = strings.TrimSpace(courseName)
		courseName = TruncateString(courseName, 60)
		table.Append([]string{index, courseName})
	}

	table.Render()
}

func RemoveEmployeeID(text string) string {
		re := regexp.MustCompile(`\b\d+\b\s*-\s*`)
	return re.ReplaceAllString(text, "")
}

func GenerateFacultyDetailsTable(faculties []types.Faculty, query string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"INDEX", "NAME"})
	table.SetBorder(true)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColWidth(27)

	for i, faculty := range faculties {
		index := fmt.Sprintf("%d", i+1)

		name := RemoveEmployeeID(faculty.Name)

		if query != "" {
			name = HighlightMatches(name, query)
		}

		table.Append([]string{index, name})
	}

	table.Render()
}

func SelectFaculty(faculties []types.Faculty) (types.Faculty, error) {
	if len(faculties) == 0 {
		return types.Faculty{}, fmt.Errorf("no faculties available for selection")
	}

	if len(faculties) <= 15 {
		GenerateFacultyDetailsTable(faculties, "")
		fmt.Print("Select a Faculty by entering the number: ")
		var index int
		_, err := fmt.Scanln(&index)
		if err != nil {
			if debug.Debug {
				fmt.Println("Invalid input for faculty selection:", err)
			}
			return types.Faculty{}, fmt.Errorf("invalid input for faculty selection")
		}
		if index < 1 || index > len(faculties) {
			fmt.Println("Invalid selection. Please enter a valid number.")
			return types.Faculty{}, fmt.Errorf("invalid faculty selection")
		}
		selectedFaculty := faculties[index-1]
		successMessage := fmt.Sprintf("# You selected Faculty: %s (ERP ID: %s)", selectedFaculty.Name, selectedFaculty.ErpID)
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(0),
		)
		if err != nil && debug.Debug {
			fmt.Println("Error creating glamour renderer:", err)
		}
		renderedMessage, err := renderer.Render(successMessage)
		if err != nil && debug.Debug {
			fmt.Println("Error rendering selected faculty message:", err)
		}
		fmt.Print(renderedMessage)
		return selectedFaculty, nil
	}

	reader := bufio.NewReader(os.Stdin)
	displayFaculties := faculties

	for {
		fmt.Print("\nEnter search query (or press Enter to list all, type 'exit' to cancel): ")
		query, err := reader.ReadString('\n')
		if err != nil {
			if debug.Debug {
				fmt.Println("Error reading input:", err)
			}
			return types.Faculty{}, fmt.Errorf("error reading input")
		}
		query = strings.TrimSpace(query)

		if strings.ToLower(query) == "exit" {
			fmt.Println("Operation cancelled by user.")
			return types.Faculty{}, fmt.Errorf("selection cancelled")
		}

		if query != "" {
			filtered := []types.Faculty{}
			for _, faculty := range faculties {
				if FuzzyMatch(query, faculty.Name) || FuzzyMatch(query, faculty.ErpID) {
					filtered = append(filtered, faculty)
				}
			}
			if len(filtered) == 0 {
				fmt.Println("No faculties matched your search. Try again.")
				continue
			}
			displayFaculties = filtered
		} else {
			displayFaculties = faculties
		}

		GenerateFacultyDetailsTable(displayFaculties, query)
		fmt.Print("Enter the number of the faculty to select (or type 's' to search again, 'exit' to cancel): ")
		selection, err := reader.ReadString('\n')
		if err != nil {
			if debug.Debug {
				fmt.Println("Error reading selection:", err)
			}
			return types.Faculty{}, fmt.Errorf("error reading selection")
		}
		selection = strings.TrimSpace(selection)

		if strings.ToLower(selection) == "s" {
			continue
		}
		if strings.ToLower(selection) == "exit" {
			fmt.Println("Operation cancelled by user.")
			return types.Faculty{}, fmt.Errorf("selection cancelled")
		}

		index, err := strconv.Atoi(selection)
		if err != nil || index < 1 || index > len(displayFaculties) {
			fmt.Println("Invalid selection. Please enter a valid number.")
			continue
		}

		selectedFaculty := displayFaculties[index-1]
		successMessage := fmt.Sprintf("# You selected Faculty: %s (ERP ID: %s)", selectedFaculty.Name, selectedFaculty.ErpID)
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(0),
		)
		if err != nil && debug.Debug {
			fmt.Println("Error creating glamour renderer:", err)
		}
		renderedMessage, err := renderer.Render(successMessage)
		if err != nil && debug.Debug {
			fmt.Println("Error rendering selected faculty message:", err)
		}
		fmt.Print(renderedMessage)

		return selectedFaculty, nil
	}
}
