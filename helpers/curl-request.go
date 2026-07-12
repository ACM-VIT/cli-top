package helpers

import (
	"bytes"
	"cli-top/types"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const sessionRefreshReuseWindow = 2 * time.Second

var sessionRefresh = struct {
	sync.Mutex
	login       func() (types.Cookies, string, error)
	cookies     types.Cookies
	regNo       string
	refreshedAt time.Time
}{}

// SetVtopLoginHandler configures the login used when VTOP expires a session.
// The returned function restores the previous handler.
func SetVtopLoginHandler(login func() (types.Cookies, string, error)) func() {
	sessionRefresh.Lock()
	previous := sessionRefresh.login
	sessionRefresh.login = login
	clearRefreshedSessionLocked()
	sessionRefresh.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			sessionRefresh.Lock()
			sessionRefresh.login = previous
			clearRefreshedSessionLocked()
			sessionRefresh.Unlock()
		})
	}
}

func clearRefreshedSessionLocked() {
	sessionRefresh.cookies = types.Cookies{}
	sessionRefresh.regNo = ""
	sessionRefresh.refreshedAt = time.Time{}
}

func refreshVtopSession() (types.Cookies, string, error) {
	sessionRefresh.Lock()
	defer sessionRefresh.Unlock()

	if ValidateCookies(sessionRefresh.cookies) && sessionRefresh.regNo != "" && time.Since(sessionRefresh.refreshedAt) < sessionRefreshReuseWindow {
		return sessionRefresh.cookies, sessionRefresh.regNo, nil
	}
	if sessionRefresh.login == nil {
		return types.Cookies{}, "", fmt.Errorf("no VTOP login handler is configured")
	}

	cookies, regNo, err := sessionRefresh.login()
	if err != nil {
		return types.Cookies{}, "", err
	}
	regNo = strings.TrimSpace(regNo)
	if !ValidateCookies(cookies) || regNo == "" {
		return types.Cookies{}, "", fmt.Errorf("login returned an invalid VTOP session")
	}

	sessionRefresh.cookies = cookies
	sessionRefresh.regNo = regNo
	sessionRefresh.refreshedAt = time.Now()
	return cookies, regNo, nil
}

func FetchReq(regNo string, cookies types.Cookies, url string, semID string, payload string, method string, header string) ([]byte, error) {
	client := GetHTTPClient()
	if err := ValidateVtopURL(url); err != nil {
		return nil, err
	}

	buildRequest := func() (*http.Request, context.CancelFunc, error) {
		ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		requestPayload := buildFetchPayload(regNo, cookies.CSRF, semID, payload, header)
		var req *http.Request
		var err error
		if method == "POST" {
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(requestPayload))
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

	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, cancel, err := buildRequest()
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			return nil, err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if readErr != nil {
			return nil, readErr
		}

		sessionExpired := resp.StatusCode == http.StatusNotFound ||
			bytes.Contains(body, []byte("Session Timed Out")) ||
			bytes.Contains(body, []byte("HTTP Status 404"))
		if sessionExpired {
			if attempt == 0 {
				newCookies, newRegNo, refreshErr := refreshVtopSession()
				if refreshErr != nil {
					return nil, fmt.Errorf("refresh VTOP session: %w", refreshErr)
				}
				cookies = newCookies
				regNo = newRegNo
				continue
			}
			return nil, fmt.Errorf("Session expired or VTOP returned 404. Please run 'cli-top login' to refresh your session.")
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("VTOP request failed with status %s", resp.Status)
		}

		return body, nil
	}

	return nil, fmt.Errorf("VTOP request failed after %d attempts", maxAttempts)
}

func buildFetchPayload(regNo string, csrf string, semID string, payload string, header string) string {
	switch payload {
	case "":
		return url.Values{
			"verifyMenu":   {"true"},
			"authorizedID": {regNo},
			"_csrf":        {csrf},
			"nocache":      {fmt.Sprint(time.Now().UnixNano())},
		}.Encode()
	case "UTC":
		return url.Values{
			"authorizedID":  {regNo},
			"_csrf":         {csrf},
			"semesterSubId": {semID},
			"x":             {time.Now().UTC().Format(time.RFC1123)},
		}.Encode()
	}

	if header == "marks" {
		const csrfPart = "Content-Disposition: form-data; name=\"_csrf\"\r\n\r\n"
		start := strings.Index(payload, csrfPart)
		if start == -1 {
			return payload
		}
		valueStart := start + len(csrfPart)
		valueEnd := strings.Index(payload[valueStart:], "\r\n")
		if valueEnd == -1 {
			return payload
		}
		return payload[:valueStart] + csrf + payload[valueStart+valueEnd:]
	}

	values, err := url.ParseQuery(payload)
	if err != nil {
		return payload
	}
	values.Set("_csrf", csrf)
	return values.Encode()
}
