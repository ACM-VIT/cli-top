package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func PrintAllDAs(regNo string, cookies types.Cookies, courseName string) {
	allSems, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error retrieving semester details:", err)
		}
		fmt.Println("Failed to retrieve semester details.")
		return
	}
	if len(allSems) == 0 {
		fmt.Println("No semesters found.")
		return
	}

	semID := allSems[len(allSems)-1].SemID

	listOfSubjects := getAllSubs(regNo, cookies, semID)
	if len(listOfSubjects) == 0 {
		fmt.Println("No subjects found.")
		return
	}

	var (
		subjDAs        []types.SubjectDAs
		allUpcomingDAs []types.DAEvent
		subjectsTable  [][]string
		subjectIDs     []string
	)

	today := time.Now().Truncate(24 * time.Hour)

	for _, detail := range listOfSubjects {
		doc := getOneSub(regNo, cookies, detail.ID)
		if doc == nil {
			if debug.Debug {
				fmt.Printf("Document for subject ID %s is nil. Skipping.\n", detail.ID)
			}
			continue
		}
		_, singleSubAllDa := pendingDAs(doc, detail)
		subjDAs = append(subjDAs, singleSubAllDa)
		subjectIDs = append(subjectIDs, detail.ID)

		for _, da := range singleSubAllDa.DAs {
			if da.DueDate.After(today) || da.DueDate.Equal(today) {
				qpNormalized := strings.TrimSpace(strings.ToLower(da.QP))
				lastUploadNormalized := strings.TrimSpace(strings.ToLower(da.Last_upload))

				if (qpNormalized == "yes" && (lastUploadNormalized == "n/a" || lastUploadNormalized == "file not uploaded")) ||
					(qpNormalized == "no" && lastUploadNormalized == "n/a") {
					allUpcomingDAs = append(allUpcomingDAs, da)
					if debug.Debug {
						fmt.Printf("Identified Upcoming DA: Title='%s', QP='%s', DueDate='%s', Last_upload='%s'\n",
							da.Title, da.QP, da.DueDate.Format(time.RFC3339), da.Last_upload)
					}
				} else {
					if debug.Debug {
						fmt.Printf("DA '%s' does not meet upcoming criteria.\n", da.Title)
					}
				}
			} else {
				if debug.Debug {
					fmt.Printf("DA '%s' is not upcoming. DueDate: '%s'\n", da.Title, da.DueDate.Format(time.RFC3339))
				}
			}
		}

		completedDAs := 0
		totalDAs := len(singleSubAllDa.DAs)
		for _, da := range singleSubAllDa.DAs {
			if da.QP == "No" && da.Last_upload != "N/A" && da.Last_upload != "File Not Uploaded" {
				completedDAs++
			} else if da.QP == "Yes" && da.Last_upload != "N/A" && da.Last_upload != "File Not Uploaded" {
				completedDAs++
			}
		}

		subjectsTable = append(subjectsTable, []string{
			detail.Name,
			fmt.Sprintf("%d/%d", completedDAs, totalDAs),
		})
	}

	var (
		icsFilePath     string
		uploadedFileURL string
		icsGenerated    bool
	)
	if len(allUpcomingDAs) > 0 {
		var icsEvents []types.ICSEvent
		for _, singleDA := range allUpcomingDAs {
			event := types.ICSEvent{
				UID:         helpers.GenerateUID("DA"),
				DtStamp:     time.Now().UTC().Format("20060102T150405Z"),
				DtStart:     singleDA.DueDate.Format("20060102"),
				DtEnd:       singleDA.DueDate.AddDate(0, 0, 1).Format("20060102"),
				Summary:     fmt.Sprintf("%s - %s", singleDA.Description, singleDA.Title),
				Description: fmt.Sprintf("DA due for %s: %s", singleDA.Description, singleDA.Title),
			}
			icsEvents = append(icsEvents, event)
		}

		downloadsDir := helpers.GetDownloadsDir()
		icsFileName := "All_DA_Deadlines.ics"
		icsFilePath = filepath.Join(downloadsDir, icsFileName)

		err := helpers.GenerateICSFileDateOnly(icsEvents, icsFilePath, "CLI-TOP DA")
		if err != nil {
			fmt.Println("Error generating ICS file:", err)
		} else {
			serverURL := "https://cli-calendar.acmvit.in" 
			uploadedFileURL, err = helpers.UploadICSFile(icsFilePath, serverURL)
			if err != nil {
				fmt.Println("Error uploading ICS file:", err)
				fmt.Println("Please import the 'All_DA_Deadlines.ics' file manually from your Downloads folder.")
			} else {
				icsGenerated = true
				if debug.Debug {
					fmt.Println("ICS file uploaded successfully. URL:", uploadedFileURL)
				}
			}
		}
	} else {
		if debug.Debug {
			fmt.Println("No upcoming DAs found. Skipping ICS generation.")
		}
	}

	if len(subjectsTable) > 0 {
		header := []string{"Subjects", "STATUS"}
		subjectsTable = append([][]string{header}, subjectsTable...)


		if icsGenerated {
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "DAs")
		}

		fmt.Println("\nPlease select a subject by entering the corresponding number:")
		subjectChoice := helpers.TableSelector("subject", subjectsTable, 0)
		if subjectChoice < 1 || subjectChoice > len(subjectIDs) {
			fmt.Println("Invalid subject selection.")
			return
		}

		selectedSubjectID := subjectIDs[subjectChoice-1]

		var singleSubDownload [][]string
		singleSubDownload = append(singleSubDownload, []string{"Title", "Due Date", "Days Left", "Status", "QP", "Last Upload"})

		maxTitleWidth := 30

		for _, everyDA := range subjDAs {
			if everyDA.Subject.ID == selectedSubjectID {
				for _, singleDA := range everyDA.DAs {
					var status string
					var daysLeft string

					if singleDA.Last_upload != "N/A" && singleDA.Last_upload != "File Not Uploaded" {
						status = "\033[32mCompleted\033[0m" // Green
						daysLeft = "N/A"
					} else {
						if singleDA.DueDate.Before(today) {
							status = "\033[31mOverdue\033[0m" // Red
							daysLeft = "N/A"
						} else {
							daysLeft = strconv.Itoa(singleDA.DaysLeft)
							// Color coding based on days left
							if singleDA.DaysLeft < 3 {
								status = "\033[31mPending  \033[0m" // Red
							} else if singleDA.DaysLeft < 7 {
								status = "\033[33mPending  \033[0m" // Yellow
							} else {
								status = "\033[34mPending  \033[0m" // Blue
							}
						}
					}

					qp := "No"
					downloadLink := ""
					if singleDA.QP == "Yes" && singleDA.DownloadLink != "" {
						qp = "Yes"
						downloadLink = singleDA.DownloadLink
					}

					var formattedLastUpload string
					if singleDA.Last_upload != "N/A" && singleDA.Last_upload != "File Not Uploaded" {
						formattedLastUpload = helpers.FormatDateTime(singleDA.Last_upload)
					} else {
						formattedLastUpload = singleDA.Last_upload
					}

					singleDownloadDA := []string{
						helpers.TruncateWithEllipses(singleDA.Title, maxTitleWidth),
						helpers.FormatDate(singleDA.DueDate.Format("02-Jan-2006")),
						daysLeft,
						status,
						qp,
						formattedLastUpload,
					}
					singleSubDownload = append(singleSubDownload, singleDownloadDA)

					if downloadLink != "" {
						singleDA.DownloadLink = downloadLink
					}
				}
			}
		}

		if len(singleSubDownload) > 1 {
			downloadChoice := helpers.TableSelector("DA index", singleSubDownload, 0)
			if downloadChoice < 1 || downloadChoice > len(singleSubDownload)-1 {
				fmt.Println("Invalid DA selection.")
				return
			}

			selectedDA := singleSubDownload[downloadChoice]
			if selectedDA[4] == "No" {
				fmt.Println("No question papers available for this DA.")
				return
			}

			var downloadLink string
			for _, everyDA := range subjDAs {
				if everyDA.Subject.ID == selectedSubjectID {
					for _, singleDA := range everyDA.DAs {
						truncatedTitle := helpers.TruncateWithEllipses(singleDA.Title, maxTitleWidth)
						if truncatedTitle == selectedDA[0] {
							downloadLink = singleDA.DownloadLink
							break
						}
					}
				}
				if downloadLink != "" {
					break
				}
			}

			if downloadLink == "" {
				fmt.Println("Download link not found for the selected DA.")
				return
			}

			currentTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
			currentTime = strings.ReplaceAll(currentTime, "UTC", "GMT")
			cur := strings.ReplaceAll(currentTime, " ", "%20")
			url := "https://vtop.vit.ac.in/vtop/" + downloadLink + "?authorizedID=" + regNo + "&_csrf=" + cookies.CSRF + "&x=" + cur

			body, err := helpers.FetchReq(regNo, cookies, url, "", "", "GET", "")
			if err != nil {
				if debug.Debug {
					fmt.Println("Error fetching DA download:", err)
				}
				fmt.Println("Failed to download the selected DA.")
				return
			}

			downloadsDir := helpers.GetDownloadsDir()
			fileName := selectedDA[0] + ".pdf"
			filePath := filepath.Join(downloadsDir, fileName)

			file, err := os.Create(filePath)
			if err != nil {
				fmt.Println("Error creating file:", err)
				return
			}
			defer file.Close()

			_, err = file.Write(body)
			if err != nil {
				fmt.Println("Error writing to file:", err)
				return
			}

			fmt.Println()
			fmt.Printf("\033]8;;file://%s\a\033[34mClick Here\033[0m\033]8;;\a\n", filePath)
			fmt.Println()
		}
	}
}

func getAllSubs(regNo string, cookies types.Cookies, semID string) []types.DAsubject {
	url := "https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment"
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error fetching subjects: %v\n", err)
		}
		fmt.Println("Failed to fetch subjects.")
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyText)))
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error parsing subjects document: %v\n", err)
		}
		fmt.Println("Failed to parse subjects data.")
		return nil
	}
	return allSubDetails(doc)
}

func allSubDetails(doc *goquery.Document) []types.DAsubject {
	var allsubs []types.DAsubject
	subjectMap := make(map[string]bool)
	doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
		td := s.Find("td")
		if td.Length() < 5 {
			return
		}
		id := strings.TrimSpace(td.Eq(1).Text())
		if id == "" || subjectMap[id] {
			return
		}
		subjectMap[id] = true
		code := strings.TrimSpace(td.Eq(2).Text())
		name := strings.TrimSpace(td.Eq(3).Text())
		tempsub := types.DAsubject{Name: name, Code: code, ID: id}
		allsubs = append(allsubs, tempsub)
	})
	if debug.Debug {
		fmt.Printf("Found %d unique subjects.\n", len(allsubs))
	}
	return allsubs
}

func getOneSub(regNo string, cookies types.Cookies, code string) *goquery.Document {
	url := "https://vtop.vit.ac.in/vtop/examinations/processDigitalAssignment"
	payloadMap := map[string]string{
		"_csrf":        cookies.CSRF,
		"classId":      code,
		"authorizedID": regNo,
		"x":            fmt.Sprintf("%d", time.Now().Unix()),
	}
	formData := helpers.FormatBodyData(payloadMap)
	subBody, err := helpers.FetchReq(regNo, cookies, url, "", formData, "POST", "")
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error fetching subject details for code %s: %v\n", code, err)
		}
		fmt.Printf("Failed to fetch details for subject code: %s\n", code)
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(subBody)))
	if err != nil {
		if debug.Debug {
			fmt.Printf("Error parsing subject details document for code %s: %v\n", code, err)
		}
		fmt.Printf("Failed to parse details for subject code: %s\n", code)
		return nil
	}
	return doc
}

func pendingDAs(doc *goquery.Document, subject types.DAsubject) (types.LatestDA, types.SubjectDAs) {
	var events types.SubjectDAs
	events.Subject = subject
	var latestDA types.LatestDA
	latestDA.Subject = subject
	daMap := make(map[string]bool)

	doc.Find("table.customTable").Each(func(i int, s *goquery.Selection) {
		headers := []string{}
		s.Find("tr.tableHeader td").Each(func(j int, th *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(th.Text()))
		})
		if len(headers) < 6 {
			return
		}
		if headers[0] == "Sl.No." && headers[1] == "Title" && headers[4] == "Due Date" && headers[5] == "QP" {
			s.Find("tr.fixedContent.tableContent").Each(func(k int, tr *goquery.Selection) {
				td := tr.Find("td")
				if td.Length() < 9 {
					return
				}

				title := strings.TrimSpace(td.Eq(1).Text())
				if title == "" || daMap[title] || title == subject.Code {
					return
				}
				daMap[title] = true

				dueDateStr := strings.TrimSpace(td.Eq(4).Find("span").Text())
				var dueDate time.Time
				if dueDateStr == "-" || dueDateStr == "" {
					dueDate = time.Time{}
				} else {
					date, err := time.Parse("02-Jan-2006", dueDateStr)
					if err != nil {
						if debug.Debug {
							fmt.Printf("Error parsing date %s: %v\n", dueDateStr, err)
						}
						dueDate = time.Time{}
					} else {
						dueDate = date
					}
				}

				qp := "No"
				downloadLinkQP := ""
				aTagQP := td.Eq(5).Find("a")
				if aTagQP.Length() > 0 {
					qp = "Yes"
					href, exists := aTagQP.Attr("href")
					if exists {
						re := regexp.MustCompile(`vtopDownload\('([^']+)'\)`)
						matches := re.FindStringSubmatch(href)
						if len(matches) > 1 {
							downloadLinkQP = matches[1]
						}
					}
				}

				lastUpdated := strings.TrimSpace(td.Eq(6).Find("span").Text())
				if lastUpdated == "" {
					lastUpdated = "N/A"
				} else {
					if isValidDateTime(lastUpdated) {
						lastUpdated = helpers.FormatDateTime(lastUpdated)
					}
				}

				tempDA := types.DAEvent{
					Title:        title,
					Description:  subject.Name,
					QP:           qp,
					Last_upload:  lastUpdated,
					DownloadLink: downloadLinkQP,
					DueDate:      dueDate,
				}

				if !tempDA.DueDate.IsZero() {
					today := time.Now().UTC().Truncate(24 * time.Hour)
					dueDateMidnight := tempDA.DueDate.Truncate(24 * time.Hour)
					diff := dueDateMidnight.Sub(today)
					tempDA.DaysLeft = int(diff.Hours() / 24) - 1 
					if tempDA.DaysLeft < 0 {
						tempDA.DaysLeft = 0 
					}
				} else {
					tempDA.DaysLeft = 0
				}

				events.DAs = append(events.DAs, tempDA)
				if debug.Debug {
					fmt.Printf("Parsed DA: %+v\n", tempDA)
				}
			})
		}
	})

	return latestDA, events
}

func isValidDateTime(dateStr string) bool {
	formats := []string{
		"02 Jan 2006 03:04 PM",
		"02 Jan 2006 15:04",
		"02-Jan-2006 03:04 PM",
		"02-Jan-2006 15:04",
		"02/01/2006 03:04 PM",
		"02/01/2006 15:04",
	}
	for _, format := range formats {
		_, err := time.Parse(format, dateStr)
		if err == nil {
			return true
		}
	}
	return false
}
