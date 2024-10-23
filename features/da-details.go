package features

//import (
// 	"cli-top/helpers"
// 	"cli-top/types"
// 	"fmt"
// 	"time"

// 	"os"
// 	"path/filepath"
// 	"regexp"
// 	"strings"

// 	"github.com/PuerkitoBio/goquery"
// )

// func PrintAllDAs(regNo string, cookies types.Cookies, course_name string) {
//     listOfSubjects := getAllSubs(regNo, cookies)
// 	listOfSubjects = append([][]string{{"Name","Code"}},listOfSubjects...)
//     subject_choice :=  helpers.TableSelectorFuzzy("course name",listOfSubjects, course_name)
// 	//fmt.Println(subject_choice)
//     subject_name := findSubjectName(listOfSubjects, subject_choice)
// 	doc := getOneSub(regNo, cookies, subject_choice)
// 	var details [][]string
// 	details = append(details, []string{"title","Due Date","QP","Last Updated On"})
// 	doc.Find("tr.fixedContent.tableContent").Each(func(i int, s *goquery.Selection) {
// 		//var row []string
//         td := s.Find("td")
//         title := td.Eq(1).Text()
//         dueDateSpan := td.Eq(4).Find("span")
//         dueDate := dueDateSpan.Text()
//         color, exists := dueDateSpan.Attr("style")
//         if exists && strings.Contains(color, "color: green;") {
//             qp := "N"
//             functionName := ""
//             if td.Eq(5).Find("span").Length() > 0 {
//                 qp = "Y"
//                 href, _ := td.Eq(5).Find("a").Attr("href")
//                 re := regexp.MustCompile(`vtopDownload\('([^']+)'\)`)
//                 matches := re.FindStringSubmatch(href)
//                 if len(matches) > 1 {
//                     functionName = matches[1]
//                 }
//             }
//             lastUpdated := td.Eq(6).Find("span").Text()
//             if lastUpdated == "" {
//                 lastUpdated = "N/A"
//             }
//             if functionName != "" {
//                 details = append(details, []string{title, dueDate, qp, lastUpdated, functionName})
//             } else {
//                 details = append(details, []string{title, dueDate, qp, lastUpdated})
//             }
//             return
//         }
//     })
//     if len(details) == 1 {
//         fmt.Println("No assignments with upcoming due dates found.")
//         return
//     }
//     ans :=helpers.TableSelector("index",details,0)
//     currentTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
//     currentTime = strings.ReplaceAll(currentTime, "UTC", "GMT")
//     cur := strings.ReplaceAll(currentTime, " ", "%20")
//     url := "https://vtop.vit.ac.in/vtop/"+details[ans][4]+"?authorizedID="+regNo+"&_csrf="+cookies.CSRF+"&x="+cur
//     body, err := helpers.FetchReq(regNo, cookies, url, "", "", "GET", "")

//     downloadsDir := filepath.Join(os.Getenv("HOME"), "Downloads")
//     fileName := details[ans][0]+" _ "+ subject_name + ".pdf"
//     filePath := filepath.Join(downloadsDir, fileName)

//     // Create the file in the Downloads folder
//     file, err := os.Create(filePath)
//     if err != nil {
//         fmt.Println("Error creating file:", err)
//         return
//     }
//     defer file.Close()

//     // Write the response body to the file
//     _, err = file.Write(body)
//     if err != nil {
//         fmt.Println("Error writing to file:", err)
//         return
//     }

//     fmt.Printf("\033[34m\033[4m\033]8;;file://%s\033\\%s\033]8;;\033\\\033[0m to open the folder.\n", filePath, "Click Here")
//     fmt.Println()
// }

// func findSubjectName(listOfSubjects [][]string, subject_choice string) string {
//     for i := 1; i < len(listOfSubjects); i++ {
//         if listOfSubjects[i][2] == subject_choice {
//             return listOfSubjects[i][0]
//         }
//     }
//     return ""
// }