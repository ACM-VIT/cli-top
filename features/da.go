package features

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	DATableRowSelector    = "tr.tableContent"
	DACustomTableSelector = "table.customTable"
	DACellSelector        = "td"
)

func PrintAllDAs(regNo string, cookies types.Cookies, courseName string) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	allSems, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error retrieving semester details:", err)
		}
		allSems, err = helpers.GetSemDetailsBackup(cookies, regNo)
		if err != nil {
			fmt.Println("Error retrieving semester details:", err)
			return
		}
	}
	if len(allSems) == 0 {
		fmt.Println("No semesters found.")
		return
	}

	var semID string
	var listOfSubjects []types.DAsubject

	for i := len(allSems) - 1; i >= 0; i-- {
		semID = allSems[i].SemID
		listOfSubjects = getAllSubs(regNo, cookies, semID)
		if len(listOfSubjects) > 0 {
			if debug.Debug {
				fmt.Printf("Selected Semester: %s (%s)\n", allSems[i].SemName, semID)
			}
			break
		} else {
			if debug.Debug {
				fmt.Printf("No subjects found for Semester: %s (%s). Trying previous semester.\n", allSems[i].SemName, semID)
			}
		}
	}

	if len(listOfSubjects) == 0 {
		fmt.Println("No subjects available in any semester.")
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

		var nextDueDate string = "N/A"
		var earliestDue time.Time

		for _, da := range singleSubAllDa.DAs {
			if (da.Last_upload == "N/A" || strings.EqualFold(da.Last_upload, "File Not Uploaded")) &&
				(da.DueDate.Equal(today) || da.DueDate.After(today)) {
				if earliestDue.IsZero() || da.DueDate.Before(earliestDue) {
					earliestDue = da.DueDate
					days := int(da.DueDate.Sub(today).Hours() / 24)
					if days == 0 {
						nextDueDate = "\033[31mTODAY\033[0m"
					} else if days < 3 {
						nextDueDate = "\033[31m" + da.DueDate.Format("02-Jan-2006") + "\033[0m"
					} else if days < 7 {
						nextDueDate = "\033[33m" + da.DueDate.Format("02-Jan-2006") + "\033[0m"
					} else {
						nextDueDate = da.DueDate.Format("02-Jan-2006")
					}
				}
			}
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
			nextDueDate,
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

		// Create the Other Downloads/ICS File directory for DA deadlines
		icsDir, err := helpers.GetOrCreateDownloadDir(filepath.Join("Other Downloads", "ICS File"))
		if err != nil {
			fmt.Println("Error creating ICS file directory:", err)
		} else {
			icsFileName := "All_DA_Deadlines.ics"
			icsFilePath = filepath.Join(icsDir, icsFileName)

			err := helpers.GenerateICSFileDateOnly(icsEvents, icsFilePath, "CLI-TOP DA")
			if err != nil {
				fmt.Println("Error generating ICS file:", err)
			} else {
				uploadedFileURL, err = helpers.UploadICSFile(icsFilePath, helpers.CalendarServerURL)
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
		}
	} else {
		if debug.Debug {
			fmt.Println("No upcoming DAs found. Skipping ICS generation.")
		}
	}

	if len(subjectsTable) > 0 {
		header := []string{"Subjects", "STATUS", "Next Pending DA"}
		subjectsTable = append([][]string{header}, subjectsTable...)

		if icsGenerated {
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "DAs")
		}
		fmt.Println("\nPlease select a subject by entering the corresponding number:")
		subjectChoice := helpers.TableSelector("subject", subjectsTable, "0")
		if subjectChoice.ExitRequest || !subjectChoice.Selected {
			fmt.Println("Selection canceled")
			return
		}

		selectedSubjectID := subjectIDs[subjectChoice.Index-1]

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
							status = "\033[31mNot Submitted\033[0m" // Red
							daysLeft = "N/A"
						} else {
							daysLeft = strconv.Itoa(singleDA.DaysLeft)
							if singleDA.DaysLeft < 3 {
								status = "\033[31mPending\033[0m" // Red
							} else if singleDA.DaysLeft < 7 {
								status = "\033[33mPending\033[0m" // Yellow
							} else {
								status = "\033[34mPending\033[0m" // Blue
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
			downloadChoice := helpers.TableSelector("DA", singleSubDownload, "0")
			if downloadChoice.ExitRequest || !downloadChoice.Selected {
				fmt.Println("Selection canceled")
				return
			}

			selectedDA := singleSubDownload[downloadChoice.Index]
			if selectedDA[4] == "No" {
				fmt.Println("No question papers available for this DA.")
				return
			}

			var selectedCode, selectedClassID string
			for _, everyDA := range subjDAs {
				if everyDA.Subject.ID == selectedSubjectID {
					for _, singleDA := range everyDA.DAs {
						truncatedTitle := helpers.TruncateWithEllipses(singleDA.Title, maxTitleWidth)
						if truncatedTitle == selectedDA[0] {
							if strings.Contains(singleDA.DownloadLink, "examinations/doDownloadQuestion/") {
								u := strings.TrimPrefix(singleDA.DownloadLink, "examinations/doDownloadQuestion/?")
								params := strings.Split(u, "&")
								for _, param := range params {
									kv := strings.SplitN(param, "=", 2)
									if len(kv) == 2 {
										if kv[0] == "code" {
											selectedCode = kv[1]
										} else if kv[0] == "classIdNumber" {
											selectedClassID = kv[1]
										}
									}
								}
							}
							break
						}
					}
				}
				if selectedCode != "" && selectedClassID != "" {
					break
				}
			}

			if selectedCode == "" || selectedClassID == "" {
				fmt.Println("Download link not found for the selected DA.")
				return
			}

			baseURL := "https://vtop.vit.ac.in/vtop/examinations/doDownloadQuestion/"
			cleanCSRF := strings.Trim(os.Getenv("CSRF"), "\"")
			payloadMap := map[string]string{
				"_csrf":         cleanCSRF,
				"authorizedID":  regNo,
				"code":          selectedCode,
				"classIdNumber": selectedClassID,
				"x":             fmt.Sprintf("%d", time.Now().Unix()),
			}
			formData := helpers.FormatBodyDataClient(payloadMap)

			client := &http.Client{Timeout: 30 * time.Second}
			body, headers, err := helpers.FetchReqClient(client, regNo, cookies, baseURL, "", formData, "POST", "application/x-www-form-urlencoded")
			if err != nil {
				if debug.Debug {
					fmt.Println("Error fetching DA download:", err)
				}
				fmt.Println("Failed to download the selected DA.")
				return
			}

			defaultName := "downloadedFile.pdf"
			ext := helpers.GetFileExtension(defaultName, body, headers)
			if debug.Debug {
				fmt.Printf("Determined file extension: %s\n", ext)
			}

			var selectedSubjectName string
			var selectedSubjectCode string
			for _, detail := range listOfSubjects {
				if detail.ID == selectedSubjectID {
					selectedSubjectName = detail.Name
					selectedSubjectCode = detail.Code
					break
				}
			}
			selectedSubjectName = helpers.SanitizeFilename(selectedSubjectName)

			courseCode := selectedSubjectCode
			courseName := selectedSubjectName

			fileName := fmt.Sprintf("%s%s", selectedDA[0], ext)
			daDir, err := helpers.GetOrCreateDownloadDir("DA")
			if err != nil {
				fmt.Println("Error creating DA download directory:", err)
				return
			}

			courseFolderName := fmt.Sprintf("%s_%s", courseCode, courseName)
			courseFolderName = helpers.SanitizeFilename(courseFolderName)
			courseDir := filepath.Join(daDir, courseFolderName)

			// Create the course directory
			if err := os.MkdirAll(courseDir, os.ModePerm); err != nil {
				fmt.Println("Error creating course directory:", err)
				return
			}

			filePath := filepath.Join(courseDir, fileName)

			err = helpers.SaveFile(body, filePath)
			if err != nil {
				fmt.Println("Error saving file:", err)
				return
			}
			fmt.Printf("File saved to: %s\n", filePath)

			fmt.Println()
			fmt.Printf("\033]8;;file://%s\a\033[34mClick Here\033[0m\033]8;;\a\n", filePath)
			fmt.Println()

			openFile(filePath)
		}
	}
}

func openFile(filePath string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default:
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			if _, err := exec.LookPath("wslview"); err == nil {
				cmd = exec.Command("wslview", filePath)
			}
		}
		if cmd == nil {
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", filePath)
			} else if _, err := exec.LookPath("gio"); err == nil {
				cmd = exec.Command("gio", "open", filePath)
			} else {
				fmt.Println("No supported command found to open the file automatically. Please open it manually:", filePath)
				return
			}
		}
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error opening file: %v\n", err)
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
	doc.Find(DATableRowSelector).Each(func(i int, s *goquery.Selection) {
		td := s.Find(DACellSelector)
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

	doc.Find(DACustomTableSelector).Each(func(i int, s *goquery.Selection) {
		headers := []string{}
		s.Find("tr.tableHeader td").Each(func(j int, th *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(th.Text()))
		})
		if len(headers) < 6 {
			return
		}
		if headers[0] == "Sl.No." && headers[1] == "Title" && headers[4] == "Due Date" && headers[5] == "QP" {
			s.Find("tr.fixedContent.tableContent").Each(func(k int, tr *goquery.Selection) {
				td := tr.Find(DACellSelector)
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
				} else {
					btn := td.Eq(5).Find("button")
					if btn.Length() > 0 {
						qp = "Yes"
						codeAttr, existsCode := btn.Attr("data-code")
						classAttr, existsClass := btn.Attr("data-classid")
						if existsCode && existsClass {
							downloadLinkQP = fmt.Sprintf("examinations/doDownloadQuestion/?code=%s&classIdNumber=%s", codeAttr, classAttr)
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
					tempDA.DaysLeft = int(diff.Hours() / 24)
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
