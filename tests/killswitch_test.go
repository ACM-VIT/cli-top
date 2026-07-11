package tests

import (
	"cli-top/debug"
	"cli-top/helpers"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

const latestJSONURLEnv = "CLI_TOP_LATEST_JSON_URL"

func newLatestJSONServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", request.Method)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCheckKillSwitchStates(t *testing.T) {
	for _, killSwitch := range []int{0, 1, 2, 3, 4} {
		t.Run(fmt.Sprintf("state %d", killSwitch), func(t *testing.T) {
			body := fmt.Sprintf(`{"version":"1.0.0","killSwitch":%d}`, killSwitch)
			server := newLatestJSONServer(t, http.StatusOK, body)
			t.Setenv(latestJSONURLEnv, server.URL)

			if got := helpers.CheckKillSwitch(); got != killSwitch {
				t.Fatalf("CheckKillSwitch() = %d, want %d", got, killSwitch)
			}
		})
	}
}

func TestCheckKillSwitchFailsSafe(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "invalid JSON", status: http.StatusOK, body: `{"version":`},
		{name: "missing version", status: http.StatusOK, body: `{"killSwitch":0}`},
		{name: "missing kill switch", status: http.StatusOK, body: `{"version":"1.0.0"}`},
		{name: "invalid kill switch", status: http.StatusOK, body: `{"version":"1.0.0","killSwitch":5}`},
		{name: "server error", status: http.StatusServiceUnavailable, body: `{"version":"1.0.0","killSwitch":0}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newLatestJSONServer(t, test.status, test.body)
			t.Setenv(latestJSONURLEnv, server.URL)

			if got := helpers.CheckKillSwitch(); got != 1 {
				t.Fatalf("CheckKillSwitch() = %d, want safe state 1", got)
			}
			if _, _, err := helpers.CheckUpdateSilently(); err == nil {
				t.Fatal("CheckUpdateSilently() error = nil, want malformed response error")
			}
		})
	}
}

func TestCheckKillSwitchFailsSafeWhenServerIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	serverURL := server.URL
	server.Close()
	t.Setenv(latestJSONURLEnv, serverURL)

	if got := helpers.CheckKillSwitch(); got != 1 {
		t.Fatalf("CheckKillSwitch() = %d, want safe state 1", got)
	}
}

func TestCheckKillSwitchCachesSuccessUntilURLReset(t *testing.T) {
	t.Setenv(latestJSONURLEnv, "")
	originalURL := helpers.GetLatestJSONURL()
	t.Cleanup(func() { helpers.SetLatestJSONURL(originalURL) })

	var requests atomic.Int32
	var state atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = fmt.Fprintf(w, `{"version":"1.0.0","killSwitch":%d}`, state.Load())
	}))
	t.Cleanup(server.Close)
	helpers.SetLatestJSONURL(server.URL)

	if got := helpers.CheckKillSwitch(); got != 0 {
		t.Fatalf("first CheckKillSwitch() = %d, want 0", got)
	}
	state.Store(3)
	if got := helpers.CheckKillSwitch(); got != 0 {
		t.Fatalf("cached CheckKillSwitch() = %d, want 0", got)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1 for cached read", got)
	}

	helpers.SetLatestJSONURL(server.URL)
	if got := helpers.CheckKillSwitch(); got != 3 {
		t.Fatalf("CheckKillSwitch() after URL reset = %d, want 3", got)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("request count after URL reset = %d, want 2", got)
	}
}

func TestLatestJSONEnvironmentOverride(t *testing.T) {
	t.Setenv(latestJSONURLEnv, "")
	originalURL := helpers.GetLatestJSONURL()
	t.Cleanup(func() { helpers.SetLatestJSONURL(originalURL) })

	fallback := newLatestJSONServer(t, http.StatusOK, `{"version":"1.0.0","killSwitch":0}`)
	override := newLatestJSONServer(t, http.StatusOK, `{"version":"1.0.0","killSwitch":3}`)
	helpers.SetLatestJSONURL(fallback.URL)
	if got := helpers.GetLatestJSONURL(); got != fallback.URL {
		t.Fatalf("GetLatestJSONURL() = %q, want configured URL %q", got, fallback.URL)
	}
	t.Setenv(latestJSONURLEnv, override.URL)

	if got := helpers.GetLatestJSONURL(); got != override.URL {
		t.Fatalf("GetLatestJSONURL() = %q, want environment override %q", got, override.URL)
	}
	if got := helpers.CheckKillSwitch(); got != 3 {
		t.Fatalf("CheckKillSwitch() = %d, want state from environment override 3", got)
	}
}

func TestCheckUpdateSilently(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		available bool
	}{
		{name: "current version", version: debug.Version, available: false},
		{name: "new version", version: debug.Version + "-new", available: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"version":%q,"killSwitch":0}`, test.version)
			}))
			t.Cleanup(server.Close)
			t.Setenv(latestJSONURLEnv, server.URL)

			available, version, err := helpers.CheckUpdateSilently()
			if err != nil {
				t.Fatalf("CheckUpdateSilently() error = %v", err)
			}
			if available != test.available || version != test.version {
				t.Fatalf("CheckUpdateSilently() = (%v, %q), want (%v, %q)", available, version, test.available, test.version)
			}
		})
	}
}
