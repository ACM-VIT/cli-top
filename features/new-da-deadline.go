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

func PrintAllDAs(regNo string, cookies types.Cookies, course_name string) {
	allSems, err := helpers.GetSemDetails(cookies, regNo)
	if err != nil {
		if debug.Debug {
			fmt.Println(err)
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

	var subjectsWithDAs []types.LastestDA
	var subjDAs []types.SubjectDAs

	for _, detail := range listOfSubjects {
		doc := getOneSub(regNo, cookies, detail.ID)
		tempLatestDA, singleSubAllDa := pendingDAs(doc, detail)
		subjDAs = append(subjDAs, singleSubAllDa)
		subjectsWithDAs = append(subjectsWithDAs, tempLatestDA)
	}

	var onlyLatestDATable [][]string
	onlyLatestDATable = append(onlyLatestDATable, []string{"Subject", "Title", "Due Date", "Days Left", "ID"})

	for _, subject := range subjectsWithDAs {
		var tableRecord []string
		tableRecord = append(tableRecord, subject.Subject.Name)

		if subject.DA.Title == "" {
			continue
		}

		title := helpers.TruncateWithEllipses(subject.DA.Title, 40)
		tableRecord = append(tableRecord, title)
		tableRecord = append(tableRecord, subject.DA.DueDate.Format("02-Jan-2006"))
		tableRecord = append(tableRecord, fmt.Sprintf("%d", subject.DA.DaysLeft))
		tableRecord = append(tableRecord, subject.Subject.ID)

		onlyLatestDATable = append(onlyLatestDATable, tableRecord)
	}

	if len(onlyLatestDATable) <= 1 {
		fmt.Println("No pending DAs found.")
		return
	}

	DAindex := helpers.TableSelectorFuzzy("subject", onlyLatestDATable, course_name)
	if DAindex < 1 || DAindex >= len(onlyLatestDATable) {
		fmt.Println("Invalid subject selection.")
		return
	}

	selectedSubjectID := onlyLatestDATable[DAindex][len(onlyLatestDATable[0])-1]

	var singleSubDownload [][]string
	singleSubDownload = append(singleSubDownload, []string{"Title", "Due Date", "Days Left", "QP", "Last Upload", "Download Link"})

	for _, everyDA := range subjDAs {
		if everyDA.Subject.ID == selectedSubjectID {
			for _, singleDA := range everyDA.DAs {
				var singleDownloadDA []string
				singleDownloadDA = append(singleDownloadDA, singleDA.Title)
				singleDownloadDA = append(singleDownloadDA, singleDA.DueDate.Format("02-Jan-2006"))
				daysLeftStr := strconv.Itoa(singleDA.DaysLeft)
				singleDownloadDA = append(singleDownloadDA, daysLeftStr)
				singleDownloadDA = append(singleDownloadDA, singleDA.QP)
				singleDownloadDA = append(singleDownloadDA, singleDA.Last_upload)
				singleDownloadDA = append(singleDownloadDA, singleDA.DownloadLink)
				singleSubDownload = append(singleSubDownload, singleDownloadDA)
			}
		}
	}

	if len(singleSubDownload) <= 1 {
		fmt.Println("No DAs available for download.")
	} else {
		downloadChoice := helpers.TableSelector("DA index", singleSubDownload, 0)
		if downloadChoice < 1 || downloadChoice >= len(singleSubDownload) {
			fmt.Println("Invalid DA selection.")
			return
		}

		downloadLink := singleSubDownload[downloadChoice][len(singleSubDownload[downloadChoice])-1]
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
		fileName := singleSubDownload[downloadChoice][0] + ".pdf"
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
		fmt.Printf("\033[34m\033[4m\033]8;;file://%s\033\\%s\033]8;;\033\\\033[0m to open the folder.\n", filePath, "Click Here")
		fmt.Println()

		var icsEvents []helpers.ICSEvent
		for _, everyDA := range subjDAs {
			for _, singleDA := range everyDA.DAs {
				if singleDA.DueDate.IsZero() || singleDA.DueDate.Before(time.Now()) {
					continue
				}

				startTime := singleDA.DueDate.Format("20060102T000000")
				endTime := singleDA.DueDate.Format("20060102T235900")

				event := helpers.ICSEvent{
					UID:         helpers.GenerateUID("DA"),
					DtStamp:     time.Now().UTC().Format("20060102T150405Z"),
					DtStart:     startTime,
					DtEnd:       endTime,
					Summary:     course_name,
					Description: singleDA.Title,
				}

				icsEvents = append(icsEvents, event)
			}
		}

		if len(icsEvents) > 0 {
			icsFileName := "DA_Deadlines.ics"
			icsFilePath := filepath.Join(downloadsDir, icsFileName)

			err := helpers.GenerateICSFileWithFilename(icsEvents, icsFilePath, "CLI-TOP DAs")
			if err != nil {
				fmt.Println("Error generating ICS file:", err)
			} else {
				serverURL := "https://cli-calendar.acmvit.in"
				uploadedFileURL, err := helpers.UploadICSFile(icsFilePath, serverURL)
				if err != nil {
					fmt.Println("Error uploading ICS file:", err)
					fmt.Println("Please import the 'DA_Deadlines.ics' file manually from your Downloads folder.")
				} else {
					fmt.Printf("ICS file for DA deadlines generated at: %s\n", icsFilePath)
					helpers.GenerateCalendarImportLinks(uploadedFileURL, "DAs")
				}
			}
		} else {
			fmt.Println("No valid DA deadlines found to generate ICS.")
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
	doc.Find("tr.tableContent").Each(func(i int, s *goquery.Selection) {
		td := s.Find("td")
		id := strings.TrimSpace(td.Eq(1).Text())
		code := strings.TrimSpace(td.Eq(2).Text())
		name := strings.TrimSpace(td.Eq(3).Text())
		tempsub := types.DAsubject{Name: name, Code: code, ID: id}
		allsubs = append(allsubs, tempsub)
	})
	if debug.Debug {
		fmt.Printf("Found %d subjects.\n", len(allsubs))
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

func pendingDAs(doc *goquery.Document, subject types.DAsubject) (types.LastestDA, types.SubjectDAs) {
	var events types.SubjectDAs
	events.Subject = subject
	var latestDA types.LastestDA
	latestDA.Subject = subject
	doc.Find("tr.fixedContent.tableContent").Each(func(i int, s *goquery.Selection) {
		var tempDA types.DAEvent
		td := s.Find("td")
		if td.Length() < 5 {
			return
		}
		title := strings.TrimSpace(td.Eq(1).Text())
		span := td.Eq(4).Find("span")
		dateStr := strings.TrimSpace(span.Text())
		if dateStr == "-" {
			return
		}
		style, exists := span.Attr("style")
		if exists && strings.Contains(style, "color: green;") {
			date, err := time.Parse("02-Jan-2006", dateStr)
			if err != nil {
				if debug.Debug {
					fmt.Printf("Error parsing date %s: %v\n", dateStr, err)
				}
				return
			}
			qp := "N"
			functionName := ""
			if td.Eq(5).Find("span").Length() > 0 {
				qp = "Y"
				href, _ := td.Eq(5).Find("a").Attr("href")
				re := regexp.MustCompile(`vtopDownload\('([^']+)'\)`)
				matches := re.FindStringSubmatch(href)
				if len(matches) > 1 {
					functionName = matches[1]
				}
			}
			lastUpdated := td.Eq(6).Find("span").Text()
			if lastUpdated == "" {
				lastUpdated = "N/A"
			}
			tempDA.Title = title
			tempDA.DueDate = date
			tempDA.DaysLeft = int(date.Sub(time.Now()).Hours()/24) + 1
			tempDA.QP = qp
			tempDA.Last_upload = lastUpdated
			tempDA.DownloadLink = functionName
			if latestDA.DA.Title == "" {
				latestDA.DA = tempDA
			}
			events.DAs = append(events.DAs, tempDA)
		}
	})
	return latestDA, events
}
