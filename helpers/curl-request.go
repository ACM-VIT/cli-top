package helpers

import (
	"bytes"
	"cli-top/debug"
	"cli-top/types"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lpernett/godotenv"
)

func FetchReq(regNo string, cookies types.Cookies, url string, semID string, payload string, method string, header string) ([]byte, error) {
	client := GetHTTPClient()
	if err := ValidateVtopURL(url); err != nil {
		return nil, err
	}

	buildRequest := func() (*http.Request, context.CancelFunc, error) {
		if payload == "" {
			payload = fmt.Sprintf("verifyMenu=true&authorizedID=%s&_csrf=%s&nocache=%d", regNo, cookies.CSRF, time.Now().UnixNano())
		} else if payload == "UTC" {
			payload = fmt.Sprintf("authorizedID=%s&_csrf=%s&semesterSubId=%s&x=%s", regNo, cookies.CSRF, semID, time.Now().UTC().Format(time.RFC1123))
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		var req *http.Request
		var err error
		if method == "POST" {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
		} else if method == "GET" {
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		} else {
			cancel()
			return nil, nil, fmt.Errorf("invalid method: %s", method)
		}
		if err != nil {
			cancel()
			return nil, nil, err
		}

		SetVtopHeaders(req)
		if header == "marks" {
			req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundary9yjNZXu7BBjgQK7J")
		} else {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		req.Header.Set("Cookie", fmt.Sprintf("SERVERID=%s; JSESSIONID=%s", cookies.SERVERID, cookies.JSESSIONID))
		return req, cancel, nil
	}

	retry := false

RETRY:
	req, cancel, err := buildRequest()
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	cancel()
	if err != nil {
		return nil, err
	}

	if (resp.StatusCode == 404 || bytes.Contains(body, []byte("Session Timed Out")) || bytes.Contains(body, []byte("HTTP Status 404"))) && !retry {
		if vtopLoginFunc := getVtopLoginFunc(); vtopLoginFunc != nil {
			newCookies, _ := vtopLoginFunc()
			cookies = newCookies
			retry = true
			goto RETRY
		}
		return nil, fmt.Errorf("Session expired or VTOP returned 404. Please run 'cli-top login' to refresh your session.")
	}

	return body, nil
}

func getVtopLoginFunc() func() (types.Cookies, string) {
	return func() (types.Cookies, string) {
		_ = godotenv.Load(ConfigFilePath())
		LoadSemesterCacheFromEnv()
		userInfo := types.LogIn{
			Username: os.Getenv("VTOP_USERNAME"),
			Password: os.Getenv("PASSWORD"),
		}
		key := os.Getenv("KEY")
		_, err := DecryptPasswordProxy(userInfo.Password, key)
		if err != nil && debug.Debug {
			fmt.Println("Error decrypting password during auto-relogin:", err)
		}
		if vtopLoginGlobal != nil {
			return vtopLoginGlobal()
		}
		return types.Cookies{}, ""
	}
}

var vtopLoginGlobal func() (types.Cookies, string)
var DecryptPasswordProxy func(string, string) (string, error)
