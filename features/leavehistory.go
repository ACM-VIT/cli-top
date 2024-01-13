package features

import (
	"fmt"
	"io"
	"time"
	"bytes"
	// "log"
	"net/http"
	// "strings"

	"vtop-cli/types"

	// "github.com/PuerkitoBio/goquery"
	// "github.com/olekukonko/tablewriter"
)

func FetchReq2(regNo string, cookies types.Cookies, url string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request
	// currentTime := time.Now()

	// Format the time according to the desired format
	// formattedTime := currentTime.UTC().Format("02 Jan 2006 15:04:05 MST")
	// fmt.Println(ime.Now().UnixNano())
	fmt.Println(time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))


	// payload := fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
	payload := fmt.Sprintf("_csrf=%s&authorizedID=%s&history=&form=undefined&control=history&x=%s)",cookies.CSRF,regNo,time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	fmt.Println()

	//fmt.Println(payload)
	// Create a new request with POST method and payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, err
	}

	// Set headers or cookies if needed
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	fmt.Println("response body:",string(body))

	return body, nil
}

// func PrintLeaveHistory(regNo string, cookies types.Cookies, url string) {
//     body, err := FetchReq2(regNo, cookies, url)
//     if err != nil {
//         log.Fatal("Error fetching data:", err)
//         return
//     }

//     doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
//     if err != nil {
//         log.Fatal("Error parsing HTML:", err)
//         return
//     }

//     // Define headers and skipFirstColumn
//     var headers []string  // Add this line to define headers
//     skipFirstColumn := false // Add this line to define skipFirstColumn

//     table := tablewriter.NewWriter(log.Writer())
//     table.SetHeader(headers)

//     doc.Find("tr").Each(func(i int, trSelection *goquery.Selection) {
//         var rowData []string

//         if skipFirstColumn {
//             trSelection.Find("td:not(:first-child)").Each(func(j int, tdSelection *goquery.Selection) {
//                 rowData = append(rowData, tdSelection.Text())
//             })
//         } else {
//             trSelection.Find("td").Each(func(j int, tdSelection *goquery.Selection) {
//                 rowData = append(rowData, tdSelection.Text())
//             })
//         }

//         table.Append(rowData)
//     })

//     table.Render()
// }


// func LeaveHistory(RegNo string, cookies types.Cookies) {
// 	leaveStatus, err := getLeaveStatus(RegNo,cookies)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Parse and print human-readable leave status
// 	parseAndPrintHumanReadable(leaveStatus)
// }

// func getLeaveStatus(authorizedID string, cookies types.Cookies) (string, error) {
// 	tr := &http.Transport{
// 		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
// 	}
// 	client := &http.Client{Transport: tr}
// 	var data = strings.NewReader(fmt.Sprintf("_csrf=%s&authorizedID=%s&history=&form=undefined&control=history&x=Wed, 20 Dec 2023 14:12:17 GMT", cookies.CSRF,authorizedID))
// 	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/hostels/student/leave/6", data)
// 	if err != nil {
// 		return "", err
// 	}

// 	// Set headers (unchanged)
// 	req.Header.Set("authority", "vtop.vit.ac.in")
// 	req.Header.Set("accept", "*/*")
// 	req.Header.Set("accept-language", "en-US,en;q=0.9")
// 	req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
// 	// req.Header.Set("cookie", "JSESSIONID=2F755A5CE11E8EAA8DF054862294A0F1	; SERVERID=s1")
// 	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))
// 	req.Header.Set("origin", "https://vtop.vit.ac.in")
// 	req.Header.Set("priority", "u=1, i")
// 	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content?")
// 	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120"`)
// 	req.Header.Set("sec-ch-ua-mobile", "?0")
// 	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
// 	req.Header.Set("sec-fetch-dest", "empty")
// 	req.Header.Set("sec-fetch-mode", "cors")
// 	req.Header.Set("sec-fetch-site", "same-origin")
// 	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.71 Safari/537.36")
// 	req.Header.Set("x-requested-with", "XMLHttpRequest")

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	bodyText, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(bodyText), nil
// }

// func parseAndPrintHumanReadable(htmlContent string) {
// 	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
// 	if err != nil {
// 		log.Fatal("Error parsing HTML:", err)
// 		return
// 	}

// 	// Create a new table and set its properties
// 	table := tablewriter.NewWriter(log.Writer())
// 	table.SetHeader([]string{"Leave ID", "Visit Place", "Reason", "Leave Type", "From", "To", "Status", "Remarks"})

// 	// Find and print data from the HTML
// 	doc.Find("tr").Each(func(i int, trSelection *goquery.Selection) {
// 		var rowData []string
// 		// Skip the first column (Leave ID)
// 		trSelection.Find("td:not(:first-child)").Each(func(j int, tdSelection *goquery.Selection) {
// 			// Add text content of each table cell to the row data
// 			rowData = append(rowData, tdSelection.Text())
// 		})
// 		// Add row data to the table
// 		table.Append(rowData)
// 	})

// 	// Render the table
// 	table.Render()
// }




