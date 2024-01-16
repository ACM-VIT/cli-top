package helpers

import (
	"fmt"
	"strings"
)

func PrintTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-10s | %-50s | %-25s | %-21s | %-5s | %-6s |\n",
		"S.No.", "Course Code", "Course Title", "Course Type", "Credits", " Total", "Grade"))
	builder.WriteString("|       |             |                                                    |                           |-----------------------|        |        |\n")
	//builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")

	builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-7s| %-6s |\n",
		"", "", "", "", "L", "P", "J", "C", "", ""))
	builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")
}

func PrintFormattedRow(row []string, builder *strings.Builder, count int) {
	if strToInt(row[0]) == count-1 {
		builder.WriteString(fmt.Sprintf("| \x1b[32m%-5s\x1b[0m | \x1b[32m%-11s\x1b[0m | \x1b[32m%-50s\x1b[0m | \x1b[32m%-25s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-3s\x1b[0m | \x1b[32m%-6s\x1b[0m | \x1b[32m%-6s\x1b[0m |\n",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[9], row[10]))
	} else {
		builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-6s | %-6s |\n",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[9], row[10]))
	}
}
