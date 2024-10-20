package helpers

import (
    "bufio"
    "bytes"
    "cli-top/debug"
    "cli-top/types"
    "fmt"
    "os"
    "regexp"
    "strconv"
    "strings"

    "github.com/charmbracelet/glamour"
    "github.com/olekukonko/tablewriter"
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

func addLeftPadding(output string, spaces int) string {
    pad := strings.Repeat(" ", spaces)
    lines := strings.Split(output, "\n")
    for i, line := range lines {
        if strings.TrimSpace(line) != "" {
            lines[i] = pad + line
        }
    }
    return strings.Join(lines, "\n")
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
        slot = strings.ReplaceAll(slot, "┼", "+")
        name := RedactERPID(faculty.Name)

        if query != "" {
            name = HighlightMatches(name, query)
        }

        table.Append([]string{index, slot, name})
    }

    table.Render()
    output := strings.ReplaceAll(buf.String(), "-", "─")
    output = strings.ReplaceAll(output, "┼", "+")
    output = strings.ReplaceAll(output, "|", "│")

    output = addLeftPadding(output, 2)

    fmt.Print(output)
    fmt.Println()
}

func GenerateCourseDetailsTable(courses []types.Course) {
    var buf bytes.Buffer
    table := tablewriter.NewWriter(&buf)

    table.SetHeader([]string{"INDEX", "COURSE CODE", "COURSE NAME"})

    table.SetBorder(false)
    table.SetHeaderLine(true)
    table.SetRowLine(false)
    table.SetAutoWrapText(false)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetColumnSeparator("│")

    for i, course := range courses {
        index := fmt.Sprintf("%5d", i+1)
        courseCode, courseName := SplitCourseName(course.Name)
        table.Append([]string{index, courseCode, courseName})
    }

    table.Render()
    output := strings.ReplaceAll(buf.String(), "-", "─")
    output = strings.ReplaceAll(output, "┼", "+")
    output = strings.ReplaceAll(output, "|", "│")

    output = addLeftPadding(output, 2)

    fmt.Print(output)
    fmt.Println()
}

func SelectFaculty(faculties []types.Faculty, facultyFlag int) (types.Faculty, error) {
    if len(faculties) == 0 {
        return types.Faculty{}, fmt.Errorf("no faculties available for selection")
    }

    if facultyFlag > 0 && facultyFlag <= len(faculties) {
        selectedFaculty := faculties[facultyFlag-1]
        redactedName := RedactERPID(selectedFaculty.Name)
        successMessage := fmt.Sprintf("# You selected Faculty: %s", redactedName)
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

    // Existing selection logic remains the same
    // You can implement the interactive selection if needed

    return types.Faculty{}, fmt.Errorf("faculty selection not implemented")
}

func RemoveDuplicateFaculties(faculties []types.Faculty) []types.Faculty {
    uniqueFaculties := make([]types.Faculty, 0)
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

func GenerateCourseMaterialsTable(materials []types.CourseMaterial) {
    var buf bytes.Buffer
    table := tablewriter.NewWriter(&buf)

    table.SetHeader([]string{"INDEX", "DATE", "DAY ORDER/SLOT", "TOPIC", "REF MATERIALS"})

    table.SetBorder(false)
    table.SetHeaderLine(true)
    table.SetRowLine(false)
    table.SetAutoWrapText(false)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetColumnSeparator("│")

    for _, material := range materials {
        index := fmt.Sprintf("%5d", material.Index)
        date := material.Date
        dayOrderSlot := material.DayOrderSlot
        dayOrderSlot = strings.ReplaceAll(dayOrderSlot, "┼", "+")
        topic := TruncateString(material.Topic, 40)
        refMaterialsCount := fmt.Sprintf("%d", len(material.ReferenceMaterials))
        table.Append([]string{index, date, dayOrderSlot, topic, refMaterialsCount})
    }

    table.Render()
    output := strings.ReplaceAll(buf.String(), "-", "─")
    output = strings.ReplaceAll(output, "┼", "+")
    output = strings.ReplaceAll(output, "|", "│")

    output = addLeftPadding(output, 2)

    fmt.Print(output)
    fmt.Println()
}

func SelectCourseMaterials(materials []types.CourseMaterial) ([]types.CourseMaterial, error) {
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("Enter the index numbers of the topics to download (e.g., 1,2,3), or 0 for bulk download: ")
    input, err := reader.ReadString('\n')
    if err != nil {
        if debug.Debug {
            fmt.Println("Error reading input:", err)
        }
        return nil, err
    }
    input = strings.TrimSpace(input)
    if input == "" {
        fmt.Println("No input provided.")
        return nil, fmt.Errorf("no input provided")
    }
    if input == "0" {
        return nil, nil
    }
    indicesStr := strings.Split(input, ",")
    var selectedMaterials []types.CourseMaterial
    for _, idxStr := range indicesStr {
        idxStr = strings.TrimSpace(idxStr)
        idx, err := strconv.Atoi(idxStr)
        if err != nil {
            fmt.Println("Invalid index:", idxStr)
            continue
        }
        if idx < 1 || idx > len(materials) {
            fmt.Println("Index out of range:", idx)
            continue
        }
        selectedMaterials = append(selectedMaterials, materials[idx-1])
    }
    if len(selectedMaterials) == 0 {
        fmt.Println("No valid indices selected.")
        return nil, fmt.Errorf("no valid indices selected")
    }
    return selectedMaterials, nil
}
