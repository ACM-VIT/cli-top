package helpers

import (
    "bufio"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "regexp"
    "strconv"
    "strings"

    "github.com/charmbracelet/glamour"
    "github.com/olekukonko/tablewriter"

    "cli-top/debug"
    "cli-top/types"
)

// RemoveCourseCode removes the course code from the course name using regex.
func RemoveCourseCode(courseName string) string {
    re := regexp.MustCompile(`^[A-Z]{4}\d{3}[A-Z]?\s*-\s*`)
    return re.ReplaceAllString(courseName, "")
}

// TruncateString truncates a string to the specified maxLength and adds an ellipsis if truncated.
func TruncateString(str string, maxLength int) string {
    if len(str) <= maxLength {
        return str
    }
    if maxLength <= 3 {
        return str[:maxLength]
    }
    return str[:maxLength-3] + "..."
}

// HighlightMatches highlights the matching parts of the text using ANSI bold codes.
func HighlightMatches(text, query string) string {
    re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
    return re.ReplaceAllStringFunc(text, func(match string) string {
        return "\033[1m" + match + "\033[0m" // ANSI bold
    })
}

// FuzzyMatch checks if the query is a substring of the text, case-insensitive.
func FuzzyMatch(query, text string) bool {
    query = strings.ToLower(query)
    text = strings.ToLower(text)
    return strings.Contains(text, query)
}

// FormatBodyData formats the payload map into URL-encoded form data.
func FormatBodyData(payload map[string]string) string {
    var data []string
    for key, value := range payload {
        data = append(data, fmt.Sprintf("%s=%s", key, value))
    }
    return strings.Join(data, "&")
}

// FetchReq makes HTTP requests and returns the response body.
func FetchReq(regNo string, cookies types.Cookies, url, referer, body, method, contentType string) ([]byte, error) {
    client := &http.Client{}
    req, err := http.NewRequest(method, url, strings.NewReader(body))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", contentType)
    if referer != "" {
        req.Header.Set("Referer", referer)
    }
    // Set cookies if needed
    for key, value := range cookies {
        req.AddCookie(&http.Cookie{Name: key, Value: value})
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return ioutil.ReadAll(resp.Body)
}

// GenerateFacultyDetailsTable generates and prints the faculty details table using tablewriter.
func GenerateFacultyDetailsTable(faculties []types.Faculty, query string) {
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"INDEX", "SLOT", "NAME"})
    table.SetBorder(true)
    table.SetAutoWrapText(false)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetColWidth(60) // Adjust based on your terminal size and data

    for i, faculty := range faculties {
        index := fmt.Sprintf("%d", i+1)
        name := faculty.Name
        slot := faculty.Slot

        if query != "" {
            name = HighlightMatches(name, query)
        }

        table.Append([]string{index, slot, name})
    }

    table.Render()
}

// SelectFaculty allows the user to select a faculty, with optional search functionality.
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

// GenerateCourseDetailsTable generates and prints the course details table using tablewriter.
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
