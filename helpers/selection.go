package helpers

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/olekukonko/tablewriter"

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

func GenerateFacultyDetailsTable(faculties []types.Faculty, query string) {
    var buf bytes.Buffer
    table := tablewriter.NewWriter(&buf)

    table.SetHeader([]string{"INDEX", "SLOT", "NAME"})

    table.SetBorder(false)
    table.SetHeaderLine(true)
    table.SetRowLine(false)
    table.SetAutoWrapText(false)
    table.SetAlignment(tablewriter.ALIGN_LEFT) 
    table.SetColumnSeparator("│")

    for i, faculty := range faculties {
        index := fmt.Sprintf("%5d", i+1) 
        slot := faculty.Slot              
        name := faculty.Name              

        if query != "" {
            name = HighlightMatches(name, query) 
        }

        table.Append([]string{index, slot, name})
    }

    table.Render()
    output := strings.ReplaceAll(buf.String(), "-", "─")
    output = strings.ReplaceAll(output, "+", "┼")
    output = strings.ReplaceAll(output, "|", "│")

    fmt.Print(output)
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
		successMessage := fmt.Sprintf("# You selected Faculty: %s", selectedFaculty.Name)
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
				if FuzzyMatch(query, faculty.Name) {
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
		successMessage := fmt.Sprintf("# You selected Faculty: %s", selectedFaculty.Name)
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

func GenerateCourseDetailsTable(courses []types.Course) {
    var buf bytes.Buffer
    table := tablewriter.NewWriter(&buf)

    table.SetHeader([]string{"INDEX", "COURSE NAME"})

    table.SetBorder(false)
    table.SetHeaderLine(true)
    table.SetRowLine(false)
    table.SetAutoWrapText(false)
    table.SetAlignment(tablewriter.ALIGN_LEFT) 
    table.SetColumnSeparator("│")

    for i, course := range courses {
        index := fmt.Sprintf("%5d", i+1) 
        name := course.Name
        table.Append([]string{index, name})
    }

    table.Render()
    output := strings.ReplaceAll(buf.String(), "-", "─")
    output = strings.ReplaceAll(output, "+", "┼")
    output = strings.ReplaceAll(output, "|", "│")

    fmt.Print(output)
}


func RemoveDuplicateFaculties(faculties []types.Faculty) []types.Faculty {
	uniqueFaculties := make([]types.Faculty, 0)
	keys := make(map[string]bool)
	for _, faculty := range faculties {
		key := faculty.ID + "_" + faculty.Name + "_" + faculty.Slot
		if _, value := keys[key]; !value {
			keys[key] = true
			uniqueFaculties = append(uniqueFaculties, faculty)
		}
	}
	return uniqueFaculties
}
