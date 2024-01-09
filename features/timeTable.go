package features

import (
	//"VTOP-CLI/types"
	//"bytes"
	"fmt"
	//"io"
	"log"
	//"math"
	//"net/http"
	//"strconv"
	"strings"
	//"time"
	types "vtop-cli/types"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	//"golang.org/x/net/html"
)



func GetTimeTable(regNo string, cookies types.Cookies, semId string) {

	url := "https://vtop.vit.ac.in/vtop/processViewTimeTable"

	sel_id := Timetable(regNo, cookies, 0)
	//fmt.Println(sel_id)
	bodyText, err := fetchReq2(regNo, cookies, url, sel_id)
	if err != nil {
		log.Fatal(err)
	}

	// Use goquery to parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		log.Fatal(err)
	}
	
	findAndSaveTimeTable(doc)
	
}

func Timetable(regNo string, cookies types.Cookies, sem_choice int) string {

	selectedSemId := ""
	selectedSemName := ""

	var choice int
	fmt.Print("\nEnter the index of the semester to view time table: ")
	fmt.Scanln(&choice)
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

	output, err := renderer.Render(formattedSelection)
	if err != nil {
		log.Fatal("Error rendering formatted string:", err)
	}

	fmt.Print(output)

	fmt.Println()

	return selectedSemId

}

//  func findAndSaveTimeTable(doc *goquery.Document) {
//  	var markdownTable strings.Builder
//  	targetID := "timeTableStyle"
//  	table := doc.Find("table#" + targetID)
//  	if table.Length() > 0 {
//  		//printTableTimeTable("Header", nil, &markdownTable)
//  		table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
//  			row := []string{} // Initialize a new slice for each row
//  			rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
//  				text := strings.TrimSpace(cell.Text())
//  				row = append(row, text)
//  			})
//  			//fmt.Println("hi table")
//  			//fmt.Println(row)
//  			printFormattedRowTimeTable(row, &markdownTable)
//  		})
//  	} else {
//  		fmt.Println("Table with ID 'timeTableStyle' not found")
//  	}
//  	fmt.Println(markdownTable.String())
//  }

func findAndSaveTimeTable(doc *goquery.Document) {
    targetID := "timeTableStyle"
    table := doc.Find("table#" + targetID)
    if table.Length() > 0 {
        rows := [][]string{}
        timeL := [][]string{} ; timeTh := [][]string{}
        var indices []int  // Move the indices declaration outside the inner scope
        table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
            row := []string{} // Initialize a new slice for each row
            timeLab := []string{}
			timeTheory := []string{}
            rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
                bgcolor, exists := cell.Attr("bgcolor")
                if exists && bgcolor == "#CCFF33" {
                    text := strings.TrimSpace(cell.Text())
                    row = append(row, text)
					fmt.Println(j)
                    indices = append(indices, j)
                } else if exists && bgcolor == "#99CCFF" {
                    text := strings.TrimSpace(cell.Text())
                    timeLab = append(timeLab, text)
                }else if exists && bgcolor == "##CCCCFF" {
                    text := strings.TrimSpace(cell.Text())
                    timeTheory = append(timeTheory, text)
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
        for i, subRow := range rows {
            fmt.Printf("Sub-Row %d:\n", i+1)
            fmt.Println(subRow)
            fmt.Println()
        }
        
        fmt.Println(timeL)

        

		for _, timings := range timeL {
			// Filter out "Lunch" and "-"
			filteredL := []string{}
			for _, time := range timings {
				if time != "Lunch" && time != "-" {
					filteredL = append(filteredL, time)
				}
			}
			for i := 0; i < len(timeL[0])-1; i++ {
				if timeL[0][i] != "Lunch" && timeL[0][i] != "-" && timeL[1][i] != "Lunch" && timeL[1][i] != "-" {
					output_.WriteString(fmt.Sprintf("%s to %s\n", timeL[0][i], timeL[1][i]))
				}
			}
			break
			fmt.Println()
			
		}
		// fmt.Println("theory")
		// for _, timingsTh := range timeTh {
		// 	// Filter out "Lunch" and "-"
		// 	filteredTh := []string{}
		// 	for _, timeTh := range timingsTh {
		// 		if timeTh != "Lunch" && timeTh != "-" {
		// 			filteredTh = append(filteredTh, timeTh)
		// 		}
		// 	}
		// 	for _, indexTh := range indices {
		// 		//fmt.Println(index,len(filtered))
		// 		if indexTh < len(filteredTh) {
		// 			//fmt.Println(index)
		// 			fmt.Print(filteredTh[indexTh-2]+" to ")
		// 		} 
		// 		fmt.Println(" ")
		// 	}
			
		// }


    } else {
        fmt.Println("Table with ID 'timeTableStyle' not found")
    }
}

// func removeDuplicates(nums []int) []int {
//     encountered := map[int]bool{}
//     result := []int{}

//     for v := range nums {
//         if encountered[nums[v]] == false {
//             encountered[nums[v]] = true
//             result = append(result, nums[v])
//         }
//     }

//     return result
// }



// func findAndSaveTimeTable(doc *goquery.Document) {
//     var markdownTable strings.Builder
//     targetID := "timeTableStyle"
//     table := doc.Find("table#" + targetID)
//     if table.Length() > 0 {
//         // Transpose the table data
//         var columnData [][]string
//         table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
//             rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
//                 bgcolor, exists := cell.Attr("bgcolor")
//                 if exists && bgcolor == "#CCFF33" {
//                     text := strings.TrimSpace(cell.Text())
//                     if len(columnData) <= j {
//                         columnData = append(columnData, []string{text})
//                     } else {
//                         columnData[j] = append(columnData[j], text)
//                     }
//                 }
//             })
//         })

//         // Build the string in markdown table format
//         for _, column := range columnData {
//             for _, data := range column {
//                 markdownTable.WriteString("| " + data + " ")
//             }
//             markdownTable.WriteString("|\n")
//         }
//     } else {
//         fmt.Println("Table with ID 'timeTableStyle' not found")
//     }

//     fmt.Println(markdownTable.String())
// }

// func findAndSaveTimeTable(doc *goquery.Document) {
//     var markdownTable strings.Builder
//     targetID := "timeTableStyle"
//     table := doc.Find("table#" + targetID)
//     if table.Length() > 0 {
//         // Initialize columnData to hold the transposed data
//         columnData := [][]string{}

//         // Loop through each row
//         table.Find("tbody tr").Each(func(i int, rowSelection *goquery.Selection) {
//             // Loop through each cell (td) in the row
//             rowSelection.Find("td").Each(func(j int, cell *goquery.Selection) {
//                 bgcolor, exists := cell.Attr("bgcolor")
//                 text := strings.TrimSpace(cell.Text())

//                 if exists && bgcolor == "#CCFF33" {
//                     // If the column index exceeds the number of columns in columnData, add a new column
//                     for len(columnData) <= j {
//                         columnData = append(columnData, []string{})
//                     }

//                     // Store data in the appropriate column
//                     columnData[j] = append(columnData[j], text)
//                 }
//             })
//         })

//         // Print the transposed data (column-wise)
//         for _, column := range columnData {
//             for _, data := range column {
//                 markdownTable.WriteString("| " + data + " ")
//             }
//             markdownTable.WriteString("|\n")
//         }
//     } else {
//         fmt.Println("Table with ID 'timeTableStyle' not found")
//     }

//     fmt.Println(markdownTable.String())
// }



func printTableTimeTable(title string, data [][]string, builder *strings.Builder) {

	builder.WriteString(fmt.Sprintf("| %-5s | %-10s | %-50s | %-25s | %-21s | %-5s | %-6s |\n",
 		"S.No.", "Course Code", "Course Title", "Course Type", "Credits"," Total","Grade"))
 	builder.WriteString("|       |             |                                                    |                           |-----------------------|        |        |\n")
 	//builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")

 	builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-50s | %-25s | %-3s | %-3s | %-3s | %-3s | %-7s| %-6s |\n",
 		"", "", "", "", "L", "P","J","C", "",""))
 	builder.WriteString("|-------|-------------|----------------------------------------------------|---------------------------|-----|-----|-----|-----|--------|--------|\n")
}

func printFormattedRowTimeTable(row []string, builder *strings.Builder) {
	
		//  builder.WriteString(fmt.Sprintf("| %-5s | %-11s | %-10s | %-15s | %-3s | %-3s | %-3s | %-3s | %-6s | %-6s | %-6s | %-6s | %-6s |\n",
	 	//  row[0], row[1], row[2], row[3], row[4],row[5], row[6], row[7],row[9],row[10],row[11],row[12],row[13]))
		//  builder.WriteString("|-------------|------------|-----------|------------|------------|------------|------------|------------|------------|------------|------------|------------|------------|\n")
		// for i := 0; i < 3; i++ {
		// 	if i < len(row[0]) {
		// 		builder.WriteString(fmt.Sprintf("| %-5s |\n", row[0][i]))
		// 	} else {
		// 		// Handle the case if the index is out of range
		// 		builder.WriteString("|      |\n") // Print an empty cell or handle it as needed
		// 	}
		// }
		
		builder.WriteString(fmt.Sprintf("| %-5s |\n",row[0]))
		//fmt.Println("hi")
}

func printDay(row []string, builder *strings.Builder){


}


