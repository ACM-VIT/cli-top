package helpers

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cli-top/types"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testResponse(req *http.Request, statusCode int, status string, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Header:     make(http.Header),
		Body:       body,
		Request:    req,
	}
}

func TestFetchReqReloginRefreshesCookiesAndCSRF(t *testing.T) {
	oldClient := sharedHTTPClient
	t.Cleanup(func() { sharedHTTPClient = oldClient })

	var bodies []url.Values
	var cookieHeaders []string
	sharedHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse request body: %v", err)
		}
		bodies = append(bodies, values)
		cookieHeaders = append(cookieHeaders, req.Header.Get("Cookie"))

		if len(bodies) == 1 {
			return testResponse(req, http.StatusOK, "200 OK", io.NopCloser(strings.NewReader("Session Timed Out"))), nil
		}
		return testResponse(req, http.StatusOK, "200 OK", io.NopCloser(strings.NewReader("ok"))), nil
	})}

	loginCalls := 0
	restoreLogin := SetVtopLoginHandler(func() (types.Cookies, string, error) {
		loginCalls++
		return types.Cookies{CSRF: "fresh-csrf", JSESSIONID: "fresh-session", SERVERID: "fresh-server"}, "REG1", nil
	})
	t.Cleanup(restoreLogin)

	body, err := FetchReq(
		"REG1",
		types.Cookies{CSRF: "old-csrf", JSESSIONID: "old-session", SERVERID: "old-server"},
		"https://vtop.vit.ac.in/vtop/example",
		"",
		"authorizedID=REG1&_csrf=old-csrf",
		http.MethodPost,
		"",
	)
	if err != nil {
		t.Fatalf("FetchReq returned an error: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
	if loginCalls != 1 {
		t.Fatalf("login callback calls = %d, want 1", loginCalls)
	}
	if len(bodies) != 2 {
		t.Fatalf("request count = %d, want 2", len(bodies))
	}
	if got := bodies[0].Get("_csrf"); got != "old-csrf" {
		t.Errorf("first CSRF = %q, want old-csrf", got)
	}
	if got := bodies[1].Get("_csrf"); got != "fresh-csrf" {
		t.Errorf("second CSRF = %q, want fresh-csrf", got)
	}
	if got := cookieHeaders[1]; got != "SERVERID=fresh-server; JSESSIONID=fresh-session" {
		t.Errorf("second Cookie header = %q", got)
	}
}

func TestFetchReqBoundsSessionRetries(t *testing.T) {
	oldClient := sharedHTTPClient
	t.Cleanup(func() { sharedHTTPClient = oldClient })

	requestCount := 0
	sharedHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		return testResponse(req, http.StatusNotFound, "404 Not Found", io.NopCloser(strings.NewReader("not found"))), nil
	})}
	loginCalls := 0
	restoreLogin := SetVtopLoginHandler(func() (types.Cookies, string, error) {
		loginCalls++
		return types.Cookies{CSRF: "new", JSESSIONID: "new-session", SERVERID: "new-server"}, "REG1", nil
	})
	t.Cleanup(restoreLogin)

	_, err := FetchReq("REG1", types.Cookies{CSRF: "old"}, "https://vtop.vit.ac.in/vtop/example", "", "", http.MethodPost, "")
	if err == nil {
		t.Fatal("FetchReq returned nil error for repeated session failures")
	}
	if requestCount != 2 {
		t.Errorf("request count = %d, want 2", requestCount)
	}
	if loginCalls != 1 {
		t.Errorf("login callback calls = %d, want 1", loginCalls)
	}
}

func TestFetchReqRejectsInvalidRefreshedSession(t *testing.T) {
	oldClient := sharedHTTPClient
	t.Cleanup(func() { sharedHTTPClient = oldClient })

	var requestCount atomic.Int32
	sharedHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount.Add(1)
		return testResponse(req, http.StatusOK, "200 OK", io.NopCloser(strings.NewReader("Session Timed Out"))), nil
	})}

	restoreLogin := SetVtopLoginHandler(func() (types.Cookies, string, error) {
		return types.Cookies{}, "", nil
	})
	t.Cleanup(restoreLogin)

	_, err := FetchReq("REG1", types.Cookies{CSRF: "old", JSESSIONID: "old", SERVERID: "old"}, "https://vtop.vit.ac.in/vtop/example", "", "", http.MethodPost, "")
	if err == nil || !strings.Contains(err.Error(), "invalid VTOP session") {
		t.Fatalf("FetchReq error = %v, want invalid refreshed-session error", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("request count = %d, want no unauthenticated retry", got)
	}
}

func TestFetchReqSharesConcurrentSessionRefresh(t *testing.T) {
	oldClient := sharedHTTPClient
	t.Cleanup(func() { sharedHTTPClient = oldClient })

	sharedHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.Header.Get("Cookie"), "JSESSIONID=fresh-session") {
			return testResponse(req, http.StatusOK, "200 OK", io.NopCloser(strings.NewReader("ok"))), nil
		}
		return testResponse(req, http.StatusOK, "200 OK", io.NopCloser(strings.NewReader("Session Timed Out"))), nil
	})}

	var loginCalls atomic.Int32
	restoreLogin := SetVtopLoginHandler(func() (types.Cookies, string, error) {
		loginCalls.Add(1)
		time.Sleep(20 * time.Millisecond)
		return types.Cookies{CSRF: "fresh-csrf", JSESSIONID: "fresh-session", SERVERID: "fresh-server"}, "REG1", nil
	})
	t.Cleanup(restoreLogin)

	const callers = 12
	var wg sync.WaitGroup
	errCh := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, err := FetchReq(
				"REG1",
				types.Cookies{CSRF: "old-csrf", JSESSIONID: "old-session", SERVERID: "old-server"},
				"https://vtop.vit.ac.in/vtop/example",
				"",
				"",
				http.MethodPost,
				"",
			)
			if err != nil {
				errCh <- err
				return
			}
			if string(body) != "ok" {
				errCh <- errors.New("unexpected response body: " + string(body))
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
	if got := loginCalls.Load(); got != 1 {
		t.Fatalf("login callback calls = %d, want one shared refresh", got)
	}
}

func TestFetchReqRejectsNonSuccessfulStatus(t *testing.T) {
	oldClient := sharedHTTPClient
	t.Cleanup(func() { sharedHTTPClient = oldClient })

	sharedHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testResponse(req, http.StatusServiceUnavailable, "503 Service Unavailable", io.NopCloser(strings.NewReader("unavailable"))), nil
	})}

	_, err := FetchReq("REG1", types.Cookies{}, "https://vtop.vit.ac.in/vtop/example", "", "", http.MethodGet, "")
	if err == nil || !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Fatalf("FetchReq error = %v, want status error", err)
	}
}

func TestFetchReqReturnsReadErrors(t *testing.T) {
	oldClient := sharedHTTPClient
	t.Cleanup(func() { sharedHTTPClient = oldClient })

	wantErr := errors.New("read failed")
	sharedHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testResponse(req, http.StatusOK, "200 OK", errorReadCloser{err: wantErr}), nil
	})}

	_, err := FetchReq("REG1", types.Cookies{}, "https://vtop.vit.ac.in/vtop/example", "", "", http.MethodGet, "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("FetchReq error = %v, want %v", err, wantErr)
	}
}

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (errorReadCloser) Close() error               { return nil }
