package features

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	types "vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

func GetTimeTable(regNo string, cookies types.Cookies, semId string, sem_choice int) {
	url := "https://vtop.vit.ac.in/vtop/processViewTimeTable"

	schedule := "{(1, '08:00'): ['A1', 'L1'], (1, '09:00'): ['F1'], (1, '10:00'): ['D1'], (1, '11:00'): ['TB1'], (1, '12:00'): ['TG1'], (2, '08:00'): ['B1', 'L7'], (2, '09:00'): ['G1'], (2, '10:00'): ['E1'], (2, '11:00'): ['TC1'], (2, '12:00'): ['TAA1'], (3, '08:00'): ['C1', 'L13'], (3, '09:00'): ['A1'], (3, '10:00'): ['F1'], (3, '11:00'): ['TD1'], (3, '12:00'): ['TBB1'], (4, '08:00'): ['D1', 'L19'], (4, '09:00'): ['B1'], (4, '10:00'): ['G1'], (4, '11:00'): ['TE1'], (4, '12:00'): ['TCC1'], (5, '08:00'): ['E1', 'L25'], (5, '09:00'): ['C1'], (5, '10:00'): ['TA1'], (5, '11:00'): ['TF1'], (5, '12:00'): ['TD1'], (1, '14:00'): ['A2', 'L31'], (1, '15:00'): ['F2'], (1, '16:00'): ['D2'], (1, '17:00'): ['TB2'], (1, '18:00'): ['TG2'], (2, '14:00'): ['B2', 'L37'], (2, '15:00'): ['G2'], (2, '16:00'): ['E2'], (2, '17:00'): ['TC2'], (2, '18:00'): ['TAA2'], (3, '14:00'): ['C2', 'L43'], (3, '15:00'): ['A2'], (3, '16:00'): ['F2'], (3, '17:00'): ['TD2'], (3, '18:00'): ['TBB2'], (4, '14:00'): ['D2', 'L49'], (4, '15:00'): ['B2'], (4, '16:00'): ['G2'], (4, '17:00'): ['TE2'], (4, '18:00'): ['TCC2'], (5, '14:00'): ['E2', 'L55'], (5, '15:00'): ['C2'], (5, '16:00'): ['TA2'], (5, '17:00'): ['TF2'], (5, '18:00'): ['TD2'], (1, '09:50'): ['L3'], (1, '11:40'): ['L5'], (2, '09:50'): ['L9'], (2, '11:40'): ['L11'], (3, '09:50'): ['L15'], (3, '11:40'): ['L17'], (4, '09:50'): ['L21'], (4, '11:40'): ['L23'], (5, '09:50'): ['L27'], (5, '11:40'): ['L29'], (1, '15:50'): ['L33'], (1, '17:40'): ['L35'], (2, '15:50'): ['L39'], (2, '17:40'): ['L41'], (3, '15:50'): ['L45'], (3, '17:40'): ['L47'], (4, '15:50'): ['L51'], (4, '17:40'): ['L53'], (5, '15:50'): ['L57'], (5, '17:40'): ['L59']}"

	sel_id := Timetable(regNo, cookies, sem_choice)
	bodyText, err := fetchReq2(regNo, cookies, url, sel_id)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	findAndSaveTimeTable(doc, schedule)
}

func Timetable(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	if sem_choice == 0 {
		PrintSemDetails(regNo, cookies)
		fmt.Print("\nEnter the index of the semester to view time table: ")
		fmt.Scanln(&choice)
	} else {
		choice = sem_choice
	}
	semDet := GetSemDetailsAtten(cookies, regNo)

	if choice < 1 || choice > len(semDet.SemIds) {
		fmt.Println("Invalid choice.")
	} else {
		for i, id := range semDet.SemIds {
			if i+1 == choice {
				selectedSemId = id
				selectedSemName = semDet.SemNames[i]
			}
		}
	}

	// Format the string with glamour
	formattedSelection := fmt.Sprintf("\n# You selected SemId: %s, SemName: %s\n", selectedSemId, selectedSemName)

	// Render and print the formatted string
	renderer, err := glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(150))
	if err != nil {
		log.Fatal("Error creating glamour renderer:", err)
	}

	outputL, err := renderer.Render(formattedSelection)
	if err != nil {
		log.Fatal("Error rendering formatted string:", err)
	}

	fmt.Print(outputL)

	//fmt.Println()

	return selectedSemId

}

type KeyStruct struct {
	Group int
	Time  string
}

func parsePythonDict(schedule string) map[KeyStruct][]string {
	pythonDict := make(map[KeyStruct][]string)

	// Removing unnecessary characters and splitting the schedule string
	schedule = strings.ReplaceAll(schedule, "{", "")
	schedule = strings.ReplaceAll(schedule, "}", "")
	entries := strings.Split(schedule, "],")

	// Iterating through entries to populate the pythonDict
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, ": [")
		if len(parts) != 2 {
			continue
		}

		keyPart := strings.TrimSpace(parts[0])
		valuePart := strings.TrimSpace(parts[1])

		// Extracting group and time from keyPart
		var group int
		var time string
		fmt.Sscanf(keyPart, "(%d, '%s')", &group, &time)

		// Creating KeyStruct
		key := KeyStruct{Group: group, Time: strings.TrimSuffix(time, "')")}

		// Extracting values and populating pythonDict
		values := strings.Split(strings.Trim(valuePart, "[]"), ", ")

		// Removing single quotes and any trailing ')'
		for i := range values {
			values[i] = strings.TrimSuffix(strings.Trim(values[i], "'"), "')")
		}

		pythonDict[key] = values
	}

	return pythonDict
}

func checkTime(goMap map[int][][]string, pythonDict map[KeyStruct][]string) {

	var sortedKeys []int
	for key := range goMap {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Ints(sortedKeys)

	for _, key := range sortedKeys {
		// Access the value using the key
		value := goMap[key]

		// Print the day outside the inner loop
		fmt.Print("\033[1m")
		switch key {
		case 1:
			fmt.Println("Monday\n")
		case 2:
			fmt.Println("Tuesday\n")
		case 3:
			fmt.Println("Wednesday\n")
		case 4:
			fmt.Println("Thursday\n")
		case 5:
			fmt.Println("Friday\n")
		case 6:
			fmt.Println("Saturday\n")
		case 7:
			fmt.Println("Sunday\n")
		}
		fmt.Print("\033[0m")

		// Create a new slice to store the printInfo for the current day
		var printInfoDay []string

		for _, row := range value {
			for i := range row {
				if row[i] != "null" {
					slot := strings.Split(row[i], "-")[0]
					courseCode := strings.Split(row[i], "-")[1]
					venue := strings.Split(row[i], "-")[3]

					for keyPy, valuePy := range pythonDict {
						if key == keyPy.Group {
							for j := range valuePy {
								if valuePy[j] == slot {
									// Store information in the slice
									hour, _ := strconv.Atoi(strings.Split(keyPy.Time, ":")[0])
									min, _ := strconv.Atoi(strings.Split(keyPy.Time, ":")[1])
									if (slot[0]) == 'L' {
										// For 'L' slots, adjust the time
										min += 100
										for min >= 60 {
											min -= 60
											hour++
										}
									} else {
										// For other slots, adjust the time
										min += 50
										for min >= 60 {
											min -= 60
											hour++
										}
									}
									end := fmt.Sprintf("%d:%02d", hour, min)

									info := fmt.Sprintf("%s to %s\t| %s\t\t| %s\t| %s\t\t ", keyPy.Time, end, slot, courseCode, venue)
									printInfoDay = append(printInfoDay, info)
								}
							}
						}
					}
				}
			}
		}

		// Print the timetable for the current day
		for _, info := range printInfoDay {
			fmt.Println(info)
		}
		fmt.Println() // Add a newline to separate tables
	}

}

func findAndSaveTimeTable(doc *goquery.Document, schedule string) {

	//fmt.Println("printing inside func")
	//fmt.Println(schedule)
	pythonDict := parsePythonDict(schedule)
	//fmt.Println(pythonDict)

	targetID := "timeTableStyle"
	table := doc.Find("table#" + targetID)
	if table.Length() > 0 {
		rows := [][]string{}
		timeL := [][]string{}
		timeTh := [][]string{}
		var indices []int // Move the indices declaration outside the inner scope
		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {

			row := []string{} // Initialize a new slice for each row
			timeLab := []string{}
			timeTheory := []string{}
			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
				bgcolor, exists := cell.Attr("bgcolor")
				if exists && bgcolor == "#CCFF33" {
					text := strings.TrimSpace(cell.Text())
					row = append(row, text)

					indices = append(indices, j-2)
				} else if exists && bgcolor == "#99CCFF" {
					text := strings.TrimSpace(cell.Text())
					timeLab = append(timeLab, text)
				} else if exists && bgcolor == "##CCCCFF" {
					text := strings.TrimSpace(cell.Text())
					timeTheory = append(timeTheory, text)
				} else {
					row = append(row, "null")
				}
			})
			if len(row) > 0 {
				rows = append(rows, row)
			}
			if len(timeLab) > 0 {
				timeL = append(timeL, timeLab)
			}
			if len(timeTheory) > 0 {
				timeTh = append(timeTh, timeTheory)
			}
		})

		// Print sub-rows after all rows have been processed
		subject := [][]string{}
		//var count int
		count := 0
		for i, subRow := range rows {
			//count = 0
			//fmt.Printf("Sub-Row %d:\n", i+1)
			//fmt.Println(subRow)
			if i > 3 && i < 14 {
				subject = append(subject, subRow)
				count += 1
			}
			//fmt.Println()
		}

		var subjectDayWise map[int][][]string
		subjectDayWise = make(map[int][][]string)

		// Divide the subject array into groups of 2 subarrays
		groupSize := 2
		key := 1

		for i, subRow := range subject {
			subjectDayWise[key] = append(subjectDayWise[key], subRow)

			// Check if the current group is complete
			if (i+1)%groupSize == 0 {
				key++
			}
		}
		checkTime(subjectDayWise, pythonDict)

		var outputL strings.Builder

		var filteredL [][]string

		for _, subArray := range timeL {
			var filteredSubL []string
			for _, item := range subArray {
				if item != "Lunch" && item != "-" {
					filteredSubL = append(filteredSubL, item)
				}
			}
			filteredL = append(filteredL, filteredSubL)
		}

		for i := 0; i < len(filteredL[0])-1; i = i + 2 {

			start := fmt.Sprintf("%s", filteredL[0][i]) // Convert to string
			end := fmt.Sprintf("%s", filteredL[1][i+1]) // Convert to string
			//fmt.Println(start,end)
			outputL.WriteString(fmt.Sprintf("%s to %s\n", start, end))
		}

		//fmt.Println(filteredL)
		//fmt.Println(outputL.String())
		//fmt.Println()

	} else {
		fmt.Println("Table with ID 'timeTableStyle' not found")
	}
}

// Utility function to get the index of a slot (e.g., F1 -> 1)
func getSlotIndex(slotName string) int {
	if len(slotName) > 1 {
		return int(slotName[1] - '0')
	}
	return 0
}

func separateArray(arr []int) [][]int {
	var result [][]int
	start := 0
	for i := 1; i < len(arr); i++ {
		if arr[i] < arr[i-1] {
			result = append(result, arr[start:i])
			start = i
		}
	}
	// Add the remaining elements if any
	if start < len(arr) {
		result = append(result, arr[start:])
	}
	return result
}

func printTableTimeTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-10s | %-50s | %-25s | %-21s | %-5s | %-6s |\n",
		"S.No.", "Course Code", "Course Title", "Course Type", "Credits", " Total", "Grade"))
	builder.WriteString("|       |             |                                                    |                           |-----------------------|        |        |\n")
	//builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")

	builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-7s| %-6s |\n",
		"", "", "", "", "L", "P", "J", "C", "", ""))
	builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")
}

func printFormattedRowTimeTable(row []string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s |\n", row[0]))
	//fmt.Println("hi")
}
