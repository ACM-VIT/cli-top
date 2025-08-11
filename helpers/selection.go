package helpers

import (
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	// "github.com/olekukonko/tablewriter"
)

func RemoveCourseCode(courseName string) string {
	re := regexp.MustCompile(`^[A-Z]{4}\d{3}[A-Z]?\s*[─-]\s*`)
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

func RedactERPID(facultyName string) string {
	re := regexp.MustCompile(`^\d+\s*[─–—-]\s*`)
	return re.ReplaceAllString(facultyName, "")
}

func SplitCourseName(courseName string) (string, string) {
	re := regexp.MustCompile(`\s*[─–—-]\s*`)
	idx := re.FindStringIndex(courseName)
	if idx != nil {
		courseCode := strings.TrimSpace(courseName[:idx[0]])
		courseNamePart := strings.TrimSpace(courseName[idx[1]:])
		return courseCode, courseNamePart
	}
	return courseName, ""
}

func SplitCourseNameFull(courseName string) []string {
	re := regexp.MustCompile(`\s*[─–—-]\s*`)
	parts := re.Split(courseName, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func SplitFacultyNameFull(facultyName string) []string {
	re := regexp.MustCompile(`\s*[─–—-]\s*`)
	parts := re.Split(facultyName, -1)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func ReplaceCrossWithPlus(input string) string {
	return strings.ReplaceAll(input, "┼", "+")
}

// func GenerateFacultyDetailsTable(faculties []types.Faculty, query string) {
// 	var buf bytes.Buffer
// 	table := tablewriter.NewWriter(&buf)

// 	table.SetHeader([]string{"INDEX", "SLOT", "NAME"})

// 	table.SetBorder(false)
// 	table.SetHeaderLine(true)
// 	table.SetRowLine(false)
// 	table.SetAutoWrapText(false)
// 	table.SetAlignment(tablewriter.ALIGN_LEFT)
// 	table.SetColumnSeparator("│")

// 	for i, faculty := range faculties {
// 		index := fmt.Sprintf("%5d", i+1)
// 		slot := ReplaceCrossWithPlus(faculty.Slot)

// 		name := RedactERPID(faculty.Name)

// 		if query != "" {
// 			name = HighlightMatches(name, query)
// 		}

// 		table.Append([]string{index, slot, name})
// 	}

// 	table.Render()
// 	output := buf.String()
// 	output = strings.ReplaceAll(output, "-", "─")
// 	output = strings.ReplaceAll(output, "|", "│")

// 	output = AddLeftPadding(output, 2)

// 	fmt.Print(output)
// 	fmt.Println()
// }

// func GenerateCourseDetailsTable(courses []types.Course) {
// 	var buf bytes.Buffer
// 	table := tablewriter.NewWriter(&buf)

// 	table.SetHeader([]string{"INDEX", "COURSE CODE", "COURSE NAME"})

// 	table.SetBorder(false)
// 	table.SetHeaderLine(true)
// 	table.SetRowLine(false)
// 	table.SetAutoWrapText(false)
// 	table.SetAlignment(tablewriter.ALIGN_LEFT)
// 	table.SetColumnSeparator("│")

// 	for i, course := range courses {
// 		index := fmt.Sprintf("%5d", i+1)
// 		courseCode, courseName := SplitCourseName(course.Name)

// 		courseCode = ReplaceCrossWithPlus(courseCode)

// 		table.Append([]string{index, courseCode, courseName})
// 	}

// 	table.Render()
// 	output := buf.String()
// 	output = strings.ReplaceAll(output, "-", "─")
// 	output = strings.ReplaceAll(output, "|", "│")

// 	output = AddLeftPadding(output, 2)

// 	fmt.Print(output)
// 	fmt.Println()
// }

// func SelectFaculty(faculties []types.Faculty, facultyFlag int) (types.Faculty, error) {
// 	if len(faculties) == 0 {
// 		return types.Faculty{}, fmt.Errorf("no faculties available for selection")
// 	}

// 	if facultyFlag > 0 && facultyFlag <= len(faculties) {
// 		return faculties[facultyFlag-1], nil
// 	}

// 	if len(faculties) <= 15 {
// 		GenerateFacultyDetailsTable(faculties, "")
// 		fmt.Println()
// 		fmt.Print("Select a Faculty by entering the number: ")
// 		var index int
// 		_, err := fmt.Scanln(&index)
// 		if err != nil {
// 			if debug.Debug {
// 				fmt.Println("Invalid input for faculty selection:", err)
// 			}
// 			return types.Faculty{}, fmt.Errorf("invalid input for faculty selection")
// 		}
// 		if index < 1 || index > len(faculties) {
// 			fmt.Println("Invalid selection. Please enter a valid number.")
// 			return types.Faculty{}, fmt.Errorf("invalid faculty selection")
// 		}
// 		return faculties[index-1], nil
// 	}

// 	for {
// 		fmt.Print("\nEnter search query (or press Enter to list all, type 'exit' to cancel): ")
// 		var query string
// 		_, err := fmt.Scanln(&query)
// 		if err != nil {
// 			if debug.Debug {
// 				fmt.Println("Error reading input:", err)
// 			}
// 			return types.Faculty{}, fmt.Errorf("error reading input")
// 		}

// 		query = strings.TrimSpace(query)

// 		if strings.ToLower(query) == "exit" {
// 			fmt.Println("Operation cancelled by user.")
// 			return types.Faculty{}, fmt.Errorf("selection cancelled")
// 		}

// 		var displayFaculties []types.Faculty
// 		if query != "" {
// 			for _, faculty := range faculties {
// 				if FuzzyMatch(query, faculty.Name) {
// 					displayFaculties = append(displayFaculties, faculty)
// 				}
// 			}
// 			if len(displayFaculties) == 0 {
// 				fmt.Println("No faculties matched your search. Try again.")
// 				continue
// 			}
// 		} else {
// 			displayFaculties = faculties
// 		}

// 		GenerateFacultyDetailsTable(displayFaculties, query)
// 		fmt.Println()
// 		fmt.Print("Enter the number of the faculty to select (or type 's' to search again, 'exit' to cancel): ")
// 		var selection string
// 		_, err = fmt.Scanln(&selection)
// 		if err != nil {
// 			if debug.Debug {
// 				fmt.Println("Error reading selection:", err)
// 			}
// 			return types.Faculty{}, fmt.Errorf("error reading selection")
// 		}

// 		selection = strings.TrimSpace(selection)

// 		if strings.ToLower(selection) == "s" {
// 			continue
// 		}
// 		if strings.ToLower(selection) == "exit" {
// 			fmt.Println("Operation cancelled by user.")
// 			return types.Faculty{}, fmt.Errorf("selection cancelled")
// 		}

// 		index, err := strconv.Atoi(selection)
// 		if err != nil || index < 1 || index > len(displayFaculties) {
// 			fmt.Println("Invalid selection. Please enter a valid number.")
// 			continue
// 		}

// 		return displayFaculties[index-1], nil
// 	}
// }

func RemoveDuplicateFaculties(faculties []types.FacultyOld) []types.FacultyOld {
	uniqueFaculties := make([]types.FacultyOld, 0, len(faculties))
	keys := make(map[string]bool)
	for _, faculty := range faculties {
		key := faculty.ID + "_" + faculty.Name + "_" + faculty.Slot
		if _, exists := keys[key]; !exists {
			keys[key] = true
			uniqueFaculties = append(uniqueFaculties, faculty)
		}
	}
	return uniqueFaculties
}

func SortFacultiesAlphabetically(faculties []types.FacultyOld) {
	sort.Slice(faculties, func(i, j int) bool {
		return strings.ToLower(faculties[i].Name) < strings.ToLower(faculties[j].Name)
	})
}

// func GenerateCourseMaterialsTable(materials []types.CourseMaterial) {
// 	var buf bytes.Buffer
// 	table := tablewriter.NewWriter(&buf)

// 	table.SetHeader([]string{"INDEX", "DATE", "DAY ORDER/SLOT", "TOPIC", "REF MATERIALS"})

// 	table.SetBorder(false)
// 	table.SetHeaderLine(true)
// 	table.SetRowLine(false)
// 	table.SetAutoWrapText(false)
// 	table.SetAlignment(tablewriter.ALIGN_LEFT)
// 	table.SetColumnSeparator("│")

// 	for _, material := range materials {
// 		index := fmt.Sprintf("%5d", material.Index)
// 		date := material.Date
// 		dayOrderSlot := ReplaceCrossWithPlus(material.DayOrderSlot)
// 		topic := TruncateString(material.Topic, 40)
// 		refMaterialsCount := fmt.Sprintf("%d", len(material.ReferenceMaterials))
// 		table.Append([]string{index, date, dayOrderSlot, topic, refMaterialsCount})
// 	}

// 	table.Render()
// 	output := buf.String()
// 	output = strings.ReplaceAll(output, "-", "─")
// 	output = strings.ReplaceAll(output, "|", "│")

// 	output = AddLeftPadding(output, 2)

// 	fmt.Print(output)
// 	fmt.Println()
// }

func SelectCourseMaterials(materials []types.CourseMaterial) ([]types.CourseMaterial, error) {
	for {
		fmt.Print("Enter the index numbers of the topics to download (e.g., 1,2,3), or 0 for bulk download: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			if debug.Debug {
				fmt.Println("Error reading input:", err)
			}
			return nil, err
		}

		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("No input provided.")
			continue
		}

		if input == "0" {
			return materials, nil
		}

		indicesStr := strings.Split(input, ",")
		indexSet := make(map[int]struct{})
		var invalidIndices []string
		for _, idxStr := range indicesStr {
			idxStr = strings.TrimSpace(idxStr)
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				invalidIndices = append(invalidIndices, idxStr)
				continue
			}
			if idx < 1 || idx > len(materials) {
				invalidIndices = append(invalidIndices, idxStr)
				continue
			}
			indexSet[idx] = struct{}{}
		}

		if len(invalidIndices) > 0 {
			fmt.Println("Invalid indices:", strings.Join(invalidIndices, ", "))
		}

		if len(indexSet) == 0 {
			fmt.Println("No valid indices selected.")
			continue
		}

		var selectedMaterials []types.CourseMaterial
		for idx := range indexSet {
			selectedMaterials = append(selectedMaterials, materials[idx-1])
		}

		return selectedMaterials, nil
	}
}
