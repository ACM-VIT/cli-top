package login

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"cli-top/helpers"
	"cli-top/types"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(req *http.Request, statusCode int, status string, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func replaceSharedClientState(t *testing.T, transport http.RoundTripper, checkRedirect func(*http.Request, []*http.Request) error) {
	t.Helper()
	client := helpers.GetHTTPClient()
	oldTransport := client.Transport
	oldCheckRedirect := client.CheckRedirect
	client.Transport = transport
	client.CheckRedirect = checkRedirect
	t.Cleanup(func() {
		client.Transport = oldTransport
		client.CheckRedirect = oldCheckRedirect
	})
}

func TestPerformLoginCopiesClientBeforeChangingRedirectPolicy(t *testing.T) {
	sentinel := errors.New("shared redirect policy")
	var loginForm url.Values
	replaceSharedClientState(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/vtop/login":
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read login body: %v", err)
			}
			loginForm, err = url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("parse login body: %v", err)
			}
			resp := response(req, http.StatusFound, "302 Found", "")
			resp.Header.Set("Location", "https://vtop.vit.ac.in/vtop/redirected")
			resp.Header.Add("Set-Cookie", "JSESSIONID=logged-in; Path=/")
			return resp, nil
		case "/vtop/login/error":
			return response(req, http.StatusOK, "200 OK", ""), nil
		default:
			t.Fatalf("unexpected request path %q", req.URL.Path)
			return nil, errors.New("unexpected request")
		}
	}), func(*http.Request, []*http.Request) error { return sentinel })

	tokens, errType := performLogin(
		types.LogIn{Username: "student", Password: "p&=word"},
		types.Cookies{CSRF: "csrf", JSESSIONID: "session", SERVERID: "server"},
		"123abc",
	)
	if errType != "" {
		t.Fatalf("performLogin error type = %q", errType)
	}
	if tokens.JSESSIONID != "logged-in" {
		t.Errorf("JSESSIONID = %q, want logged-in", tokens.JSESSIONID)
	}
	if got := loginForm.Get("password"); got != "p&=word" {
		t.Errorf("encoded password = %q, want p&=word", got)
	}

	sharedPolicy := helpers.GetHTTPClient().CheckRedirect
	if sharedPolicy == nil {
		t.Fatal("performLogin cleared the shared redirect policy")
	}
	if err := sharedPolicy(nil, nil); !errors.Is(err, sentinel) {
		t.Fatalf("shared redirect policy returned %v, want %v", err, sentinel)
	}
}

func TestGetLoginPageBoundsMissingCaptchaRetries(t *testing.T) {
	rootRequests := 0
	preloginRequests := 0
	replaceSharedClientState(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/":
			rootRequests++
			resp := response(req, http.StatusOK, "200 OK", "")
			resp.Header.Add("Set-Cookie", "JSESSIONID=session; Path=/")
			resp.Header.Add("Set-Cookie", "SERVERID=server; Path=/")
			return resp, nil
		case "/vtop/prelogin/setup":
			preloginRequests++
			return response(req, http.StatusOK, "200 OK", "<html></html>"), nil
		default:
			return nil, errors.New("unexpected request")
		}
	}), nil)

	_, _, err := getLoginPage()
	if err == nil || !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("getLoginPage error = %v, want bounded retry error", err)
	}
	if rootRequests != 3 || preloginRequests != 3 {
		t.Fatalf("request counts = root:%d prelogin:%d, want 3 each", rootRequests, preloginRequests)
	}
}

func TestSessionRequestReturnsReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	replaceSharedClientState(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := response(req, http.StatusOK, "200 OK", "")
		resp.Body = errorReadCloser{err: wantErr}
		return resp, nil
	}), nil)

	_, err := getSessionServer()
	if !errors.Is(err, wantErr) {
		t.Fatalf("getSessionServer error = %v, want %v", err, wantErr)
	}
}

type errorReadCloser struct {
	err error
}

func (r errorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (errorReadCloser) Close() error               { return nil }
