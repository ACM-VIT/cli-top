package helpers

import (
    "fmt"
    "strconv"
    "strings"
)

func TableSelector(subject string ,nestedList [][]string, choice int) int {
    if choice != 0 {
		return choice
	}
    fmt.Println("\n")
    PrintTable(nestedList)
    fmt.Println("\n")
    fmt.Print("Choose a ",subject,": ")
    _, err := fmt.Scan(&choice)
	if err != nil {
		fmt.Println("Invalid input. Please enter a valid number.")
		return -1
	}
	if choice < 1 || choice > len(nestedList) {
		fmt.Println("Invalid choice.")
		return -1
	}
    fmt.Printf("\n    \033[1;44m Your selected %s: %s \033[0m\n\n",subject, nestedList[choice][1])
    return choice
} 

func PrintTable(nestedList [][]string) int {
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

    // Normalize the number of columns in each row
    for i, row := range nestedList {
        if len(row) < maxCols {
            // Add empty strings to rows with fewer columns
            for len(row) < maxCols {
                row = append(row, "")
            }
            nestedList[i] = row
        } else if len(row) > maxCols {
            // Truncate rows with more columns
            nestedList[i] = row[:maxCols]
        }
    }

    // Prepend 'INDEX' to the header row
    nestedList[0] = append([]string{"INDEX"}, nestedList[0]...)

    // Add index numbers to the remaining rows
    for i := 1; i < len(nestedList); i++ {
        index := fmt.Sprintf("%d", i)
        nestedList[i] = append([]string{index}, nestedList[i]...)
    }

    // Compute the maximum width for each column
    maxwidth := make([]int, len(nestedList[0]))
    for i, v := range nestedList[0] {
        maxwidth[i] = len(v)
    }
    for _, row := range nestedList {
        for j, v := range row {
            if len(v) > maxwidth[j] {
                maxwidth[j] = len(v)
            }
        }
    }

    // Define the right-align function for int values
    rightAlign := func(s string, width int) string {
        return fmt.Sprintf("%*s", width, s)
    }

    // Define the left-align function
    leftAlign := func(s string, width int) string {
        return fmt.Sprintf("%-*s", width, s)
    }

    // Build the format string for rows, including 3 leading spaces
    formatRow := "   " // Add 3 leading spaces
    for _, width := range maxwidth {
        formatRow += fmt.Sprintf(" %%-%ds │", width)
    }
    formatRow = strings.TrimSuffix(formatRow, " │") + "\n"

    // Print the header row
    headerRow := make([]interface{}, len(nestedList[0]))
    for i, v := range nestedList[0] {
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
    for _, row := range nestedList[1:] {
        rowToPrint := make([]interface{}, len(row))
        for i, v := range row {
            width := maxwidth[i]

            // Handle missing values by replacing them with empty strings
            if v == "" {
                v = strings.Repeat(" ", width)
            }

            // Check if the value can be converted to an integer
            if _, err := strconv.Atoi(v); err == nil {
                // Right-align if it can be converted to an int
                rowToPrint[i] = rightAlign(v, width)
            } else {
                // Left-align if it cannot be converted to an int
                rowToPrint[i] = leftAlign(v, width)
            }
        }
        fmt.Printf(formatRow, rowToPrint...)
    }
    return 0
}
