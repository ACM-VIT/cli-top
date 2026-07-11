package features

import (
	"bytes"
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

	"golang.org/x/net/html"
)

var daDownloadRegex = regexp.MustCompile(`vtopDownload\('([^']+)'\)`)

func PrintAllDAs(regNo string, cookies types.Cookies, courseName string, assignmentSelection string) {
	if !helpers.ValidateLogin(cookies) {
		return
	}

	isProxyMode := os.Getenv("CLI_TOP_PROXY_MODE") == "1"

	allSems, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			helpers.Println("Error retrieving semester details:", err)
		}
		helpers.Println("Error retrieving semester details:", err)
		return
	}
	if len(allSems) == 0 {
		helpers.Println("No semesters found.")
		return
	}

	var semID string
	var listOfSubjects []types.DAsubject

	for i := len(allSems) - 1; i >= 0; i-- {
		semID = allSems[i].SemID
		listOfSubjects = getAllSubs(regNo, cookies, semID)
		if len(listOfSubjects) > 0 {
			if debug.Debug {
				helpers.Printf("Selected Semester: %s (%s)\n", allSems[i].SemName, semID)
			}
			break
		} else {
			if debug.Debug {
				helpers.Printf("No subjects found for Semester: %s (%s). Trying previous semester.\n", allSems[i].SemName, semID)
			}
		}
	}

	if len(listOfSubjects) == 0 {
		helpers.Println("No subjects available in any semester.")
		return
	}

	type subjectFetchResult struct {
		data types.SubjectDAs
		ok   bool
	}
	results := make([]subjectFetchResult, len(listOfSubjects))
	parallel := helpers.NewParallelizer(helpers.DetermineParallelism(len(listOfSubjects)))

	for idx, detail := range listOfSubjects {
		idx := idx
		detail := detail
		parallel.Go(func() {
			body := getOneSub(regNo, cookies, detail.ID)
			if len(body) == 0 {
				if debug.Debug {
					helpers.Printf("Document for subject ID %s is nil. Skipping.\n", detail.ID)
				}
				return
			}
			_, singleSubAllDa := pendingDAs(body, detail)
			results[idx] = subjectFetchResult{data: singleSubAllDa, ok: true}
		})
	}
	parallel.Wait()

	var (
		subjDAs        []types.SubjectDAs
		allUpcomingDAs []types.DAEvent
		subjectsTable  [][]string
		subjectIDs     []string
	)

	today := time.Now().Truncate(24 * time.Hour)

	for idx, detail := range listOfSubjects {
		result := results[idx]
		if !result.ok {
			continue
		}
		singleSubAllDa := result.data
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
						helpers.Printf("Identified Upcoming DA: Title='%s', QP='%s', DueDate='%s', Last_upload='%s'\n",
							da.Title, da.QP, da.DueDate.Format(time.RFC3339), da.Last_upload)
					}
				} else if debug.Debug {
					helpers.Printf("DA '%s' does not meet upcoming criteria.\n", da.Title)
				}
			} else if debug.Debug {
				helpers.Printf("DA '%s' is not upcoming. DueDate: '%s'\n", da.Title, da.DueDate.Format(time.RFC3339))
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

	if len(subjDAs) == 0 {
		helpers.Println("No digital assignments available across subjects.")
		return
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
			helpers.Println("Error creating ICS file directory:", err)
		} else {
			icsFileName := "All_DA_Deadlines.ics"
			icsFilePath = filepath.Join(icsDir, icsFileName)

			err := helpers.GenerateICSFileDateOnly(icsEvents, icsFilePath, "CLI-TOP DA")
			if err != nil {
				helpers.Println("Error generating ICS file:", err)
			} else {
				uploadedFileURL, err = helpers.UploadICSFile(icsFilePath, helpers.CalendarServerURL)
				if err != nil {
					helpers.Println("Error uploading ICS file:", err)
					helpers.Println("Please import the 'All_DA_Deadlines.ics' file manually from your Downloads folder.")
				} else {
					icsGenerated = true
					if debug.Debug {
						helpers.Println("ICS file uploaded successfully. URL:", uploadedFileURL)
					}
				}
			}
		}
	} else {
		if debug.Debug {
			helpers.Println("No upcoming DAs found. Skipping ICS generation.")
		}
	}

	if len(subjectsTable) > 0 {
		header := []string{"Subjects", "STATUS", "Next Pending DA"}
		subjectsTable = append([][]string{header}, subjectsTable...)

		if icsGenerated {
			helpers.GenerateCalendarImportLinks(uploadedFileURL, "DAs")
		}

		if !isProxyMode {
			helpers.Println("\nPlease select a subject by entering the corresponding number:")
		}
		subjectChoice := helpers.TableSelectorFuzzy("subject", subjectsTable, courseName, helpers.NewFuzzySearch)
		if subjectChoice.ExitRequest || !subjectChoice.Selected {
			if !isProxyMode {
				helpers.Println("Selection canceled")
			}
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
			downloadChoice := helpers.TableSelectorFuzzy("DA", singleSubDownload, assignmentSelection, helpers.NewFuzzySearch)
			if downloadChoice.ExitRequest || !downloadChoice.Selected {
				if !isProxyMode {
					helpers.Println("Selection canceled")
				}
				return
			}

			selectedDA := singleSubDownload[downloadChoice.Index]
			if selectedDA[4] == "No" {
				helpers.Println("No question papers available for this DA.")
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
				helpers.Println("Download link not found for the selected DA.")
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
			formData := helpers.FormatBodyData(payloadMap)

			client := &http.Client{Timeout: 30 * time.Second}
			body, headers, err := helpers.FetchReqClient(client, cookies, baseURL, "", formData, "POST", "application/x-www-form-urlencoded")
			if err != nil {
				if debug.Debug {
					helpers.Println("Error fetching DA download:", err)
				}
				helpers.Println("Failed to download the selected DA.")
				return
			}

			defaultName := "downloadedFile.pdf"
			ext := helpers.GetFileExtension(defaultName, body, headers)
			if debug.Debug {
				helpers.Printf("Determined file extension: %s\n", ext)
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
				helpers.Println("Error creating DA download directory:", err)
				return
			}

			courseFolderName := fmt.Sprintf("%s_%s", courseCode, courseName)
			courseFolderName = helpers.SanitizeFilename(courseFolderName)
			courseDir := filepath.Join(daDir, courseFolderName)

			// Create the course directory
			if err := os.MkdirAll(courseDir, os.ModePerm); err != nil {
				helpers.Println("Error creating course directory:", err)
				return
			}

			filePath := filepath.Join(courseDir, fileName)

			err = helpers.SaveFile(body, filePath)
			if err != nil {
				helpers.Println("Error saving file:", err)
				return
			}
			helpers.Printf("File saved to: %s\n", filePath)

			helpers.Println()
			helpers.Printf("\033]8;;file://%s\a\033[34mClick Here\033[0m\033]8;;\a\n", filePath)
			helpers.Println()

			openFile(filePath)
		}
	}
}

func openFile(filePath string) {
	if helpers.ShouldMuteUI() {
		return
	}
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
				helpers.Println("No supported command found to open the file automatically. Please open it manually:", filePath)
				return
			}
		}
	}
	if err := cmd.Start(); err != nil {
		helpers.Printf("Error opening file: %v\n", err)
	}
}

func getAllSubs(regNo string, cookies types.Cookies, semID string) []types.DAsubject {
	url := "https://vtop.vit.ac.in/vtop/examinations/doDigitalAssignment"
	bodyText, err := helpers.FetchReq(regNo, cookies, url, semID, "UTC", "POST", "")
	if err != nil {
		if debug.Debug {
			helpers.Printf("Error fetching subjects: %v\n", err)
		}
		helpers.Println("Failed to fetch subjects.")
		return nil
	}
	return parseDASubjects(bodyText)
}

func parseDASubjects(body []byte) []types.DAsubject {
	var allsubs []types.DAsubject
	subjectMap := make(map[string]bool)
	z := html.NewTokenizer(bytes.NewReader(body))
	var (
		inRow    bool
		inTD     bool
		tdIndex  int
		cellText strings.Builder
		rowCells []string
	)
	for {
		switch z.Next() {
		case html.ErrorToken:
			if debug.Debug {
				helpers.Printf("Found %d unique subjects.\n", len(allsubs))
			}
			return allsubs
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			switch string(tagName) {
			case "tr":
				if hasAttr {
					class := ""
					for {
						key, val, more := z.TagAttr()
						if string(key) == "class" {
							class = string(val)
						}
						if !more {
							break
						}
					}
					if strings.Contains(class, "tableContent") {
						inRow = true
						tdIndex = -1
						rowCells = rowCells[:0]
					}
				}
			case "td":
				if inRow {
					inTD = true
					tdIndex++
					cellText.Reset()
				}
			}
		case html.TextToken:
			if inTD {
				cellText.Write(z.Text())
			}
		case html.EndTagToken:
			tagName, _ := z.TagName()
			switch string(tagName) {
			case "td":
				if inTD {
					inTD = false
					rowCells = append(rowCells, cleanCellText(cellText.String()))
				}
			case "tr":
				if inRow {
					inRow = false
					if len(rowCells) < 5 {
						break
					}
					id := strings.TrimSpace(rowCells[1])
					if id == "" || subjectMap[id] {
						break
					}
					subjectMap[id] = true
					code := strings.TrimSpace(rowCells[2])
					name := strings.TrimSpace(rowCells[3])
					allsubs = append(allsubs, types.DAsubject{Name: name, Code: code, ID: id})
				}
			}
		}
	}
}

func getOneSub(regNo string, cookies types.Cookies, code string) []byte {
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
			helpers.Printf("Error fetching subject details for code %s: %v\n", code, err)
		}
		helpers.Printf("Failed to fetch details for subject code: %s\n", code)
		return nil
	}
	return subBody
}

func pendingDAs(body []byte, subject types.DAsubject) (types.LatestDA, types.SubjectDAs) {
	var events types.SubjectDAs
	events.Subject = subject
	var latestDA types.LatestDA
	latestDA.Subject = subject
	daMap := make(map[string]bool)

	parseDAEvents(body, subject, &events, daMap)
	return latestDA, events
}

func parseDAEvents(body []byte, subject types.DAsubject, events *types.SubjectDAs, daMap map[string]bool) {
	z := html.NewTokenizer(bytes.NewReader(body))

	var (
		inTable       bool
		tableDepth    int
		tableIsTarget bool
		inHeaderRow   bool
		inDataRow     bool
		inTD          bool
		tdIndex       int
		cellText      strings.Builder
		headerCells   []string
		rowCells      []string

		rowDAcode    string
		rowEditCode  string
		rowQPLink    string
		rowQPCode    string
		rowQPClassID string
	)

	resetRow := func() {
		tdIndex = -1
		rowCells = rowCells[:0]
		rowDAcode = ""
		rowEditCode = ""
		rowQPLink = ""
		rowQPCode = ""
		rowQPClassID = ""
	}

	for {
		switch z.Next() {
		case html.ErrorToken:
			return
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, hasAttr := z.TagName()
			switch string(tagName) {
			case "table":
				if !inTable {
					classAttr := ""
					if hasAttr {
						for {
							key, val, more := z.TagAttr()
							if string(key) == "class" {
								classAttr = string(val)
							}
							if !more {
								break
							}
						}
					}
					if strings.Contains(classAttr, "customTable") {
						inTable = true
						tableDepth = 1
						tableIsTarget = false
						headerCells = headerCells[:0]
					}
				} else {
					tableDepth++
				}
			case "tr":
				if inTable && hasAttr {
					classAttr := ""
					for {
						key, val, more := z.TagAttr()
						if string(key) == "class" {
							classAttr = string(val)
						}
						if !more {
							break
						}
					}
					if strings.Contains(classAttr, "tableHeader") {
						inHeaderRow = true
						headerCells = headerCells[:0]
					} else if strings.Contains(classAttr, "tableContent") {
						inDataRow = true
						resetRow()
					}
				}
			case "td":
				if inTable && (inHeaderRow || inDataRow) {
					inTD = true
					tdIndex++
					cellText.Reset()
				}
			case "a":
				if inDataRow && tdIndex == 5 && hasAttr {
					href := ""
					for {
						key, val, more := z.TagAttr()
						if string(key) == "href" {
							href = string(val)
						}
						if !more {
							break
						}
					}
					if href != "" {
						matches := daDownloadRegex.FindStringSubmatch(href)
						if len(matches) > 1 {
							rowQPLink = matches[1]
						}
					}
				}
			case "button":
				if inDataRow && hasAttr {
					if tdIndex == 5 {
						for {
							key, val, more := z.TagAttr()
							switch string(key) {
							case "data-code":
								rowQPCode = string(val)
							case "data-classid":
								rowQPClassID = string(val)
							}
							if !more {
								break
							}
						}
					} else if tdIndex == 7 {
						for {
							key, val, more := z.TagAttr()
							if string(key) == "data-editcode" {
								rowEditCode = string(val)
							}
							if !more {
								break
							}
						}
					}
				}
			case "input":
				if inDataRow && tdIndex == 7 && hasAttr {
					nameAttr := ""
					valueAttr := ""
					for {
						key, val, more := z.TagAttr()
						switch string(key) {
						case "name":
							nameAttr = string(val)
						case "value":
							valueAttr = string(val)
						}
						if !more {
							break
						}
					}
					if nameAttr == "code" {
						rowDAcode = valueAttr
					}
				}
			}
		case html.TextToken:
			if inTD {
				cellText.Write(z.Text())
			}
		case html.EndTagToken:
			tagName, _ := z.TagName()
			switch string(tagName) {
			case "td":
				if inTD {
					inTD = false
					text := cleanCellText(cellText.String())
					if inHeaderRow {
						headerCells = append(headerCells, text)
					} else if inDataRow {
						rowCells = append(rowCells, text)
					}
				}
			case "tr":
				if inHeaderRow {
					inHeaderRow = false
					if len(headerCells) >= 6 &&
						strings.EqualFold(headerCells[0], "Sl.No.") &&
						strings.EqualFold(headerCells[1], "Title") &&
						strings.EqualFold(headerCells[4], "Due Date") &&
						strings.EqualFold(headerCells[5], "QP") {
						tableIsTarget = true
					}
				} else if inDataRow {
					inDataRow = false
					if !tableIsTarget || len(rowCells) < 9 {
						break
					}

					title := strings.TrimSpace(rowCells[1])
					if title == "" || daMap[title] || title == subject.Code {
						break
					}
					daMap[title] = true

					daCode := rowDAcode
					if daCode == "" {
						daCode = rowEditCode
					}

					dueDateStr := strings.TrimSpace(rowCells[4])
					var dueDate time.Time
					if dueDateStr == "-" || dueDateStr == "" {
						dueDate = time.Time{}
					} else {
						date, err := time.Parse("02-Jan-2006", dueDateStr)
						if err != nil {
							if debug.Debug {
								helpers.Printf("Error parsing date %s: %v\n", dueDateStr, err)
							}
							dueDate = time.Time{}
						} else {
							dueDate = date
						}
					}

					qp := "No"
					downloadLinkQP := ""
					if rowQPLink != "" {
						qp = "Yes"
						downloadLinkQP = rowQPLink
					} else if rowQPCode != "" && rowQPClassID != "" {
						qp = "Yes"
						downloadLinkQP = fmt.Sprintf("examinations/doDownloadQuestion/?code=%s&classIdNumber=%s", rowQPCode, rowQPClassID)
					}

					lastUpdated := strings.TrimSpace(rowCells[6])
					if lastUpdated == "" {
						lastUpdated = "N/A"
					} else if isValidDateTime(lastUpdated) {
						lastUpdated = helpers.FormatDateTime(lastUpdated)
					}

					tempDA := types.DAEvent{
						Title:        title,
						Description:  subject.Name,
						QP:           qp,
						Last_upload:  lastUpdated,
						DownloadLink: downloadLinkQP,
						DueDate:      dueDate,
						Code:         daCode,
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
						helpers.Printf("Parsed DA: %+v\n", tempDA)
					}
				}
			case "table":
				if inTable {
					tableDepth--
					if tableDepth <= 0 {
						inTable = false
						tableIsTarget = false
						inHeaderRow = false
						inDataRow = false
					}
				}
			}
		}
	}
}

func cleanCellText(s string) string {
	if s == "" {
		return ""
	}
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
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
