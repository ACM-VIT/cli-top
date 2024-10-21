package helpers

import (
	"bytes"
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"io"
	"net/http"
	"time"
)

func FetchReq(regNo string, cookies types.Cookies, url string, semID string, payload string, method string, header string) ([]byte, error) {
	// Create a new HTTP client
	client := &http.Client{}

	var req *http.Request
	var err error

	// Create a deafult payload if not provided
	if payload == "" {
		payload = fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
	} else if payload == "UTC" {
		payload = fmt.Sprintf("authorizedID=%s&_csrf=%s&semesterSubId=%s&x=%s", regNo, cookies.CSRF, semID, time.Now().UTC().Format(time.RFC1123))
	}
	//fmt.Println(payload)

	// Create a new request with POST/GET method and payload
	if method == "POST" {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
		if err != nil && debug.Debug {
			return nil, err
		}
	} else if method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil && debug.Debug {
			fmt.Println(err)
		}
	} else {
		fmt.Println("Invalid method")
	}

	// Set headers or cookies for specific features if needed
	if header == "marks" {
		req.Header.Set("content-type", "multipart/form-data; boundary=----WebKitFormBoundary9yjNZXu7BBjgQK7J")
	} else {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Set common headers
	req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))

	// Perform the request
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)

	if err != nil && debug.Debug {
		return nil, err
	}

	return body, nil
}
