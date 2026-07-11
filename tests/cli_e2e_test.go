package tests

import (
	"bytes"
	"cli-top/debug"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lpernett/godotenv"
)

var cliE2EVisibleCommands = []string{
	"attendance",
	"calendar",
	"cgpa",
	"completion",
	"course-allocation",
	"course-page",
	"course-page-archive",
	"da",
	"dayafter",
	"events",
	"exams",
	"facility",
	"grades",
	"holiday",
	"hostel",
	"leave",
	"library-dues",
	"login",
	"logout",
	"marks",
	"msg",
	"nightslip",
	"profile",
	"receipts",
	"syllabus",
	"timetable",
	"today",
	"tomorrow",
}

func TestCLIEndToEndSmoke(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binPath := buildLocalBinary(t, repoRoot)

	var latestJSONRequests atomic.Int32
	latestJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		latestJSONRequests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("latest.json request method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"version":%q,"killSwitch":0}`, debug.Version)
	}))
	t.Cleanup(latestJSONServer.Close)

	var unexpectedNetworkRequests atomic.Int32
	networkTrap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		unexpectedNetworkRequests.Add(1)
		http.Error(w, "external network disabled by CLI E2E test", http.StatusBadGateway)
	}))
	t.Cleanup(networkTrap.Close)

	t.Run("visible command help", func(t *testing.T) {
		cwd, env, _ := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, "--help")
		if stderr != "" {
			t.Fatalf("cli-top --help wrote to stderr:\n%s", stderr)
		}
		if got := parseCommands(stdout); !reflect.DeepEqual(got, cliE2EVisibleCommands) {
			t.Fatalf("visible command set mismatch\n got: %q\nwant: %q", got, cliE2EVisibleCommands)
		}

		commandsAndAliases := append([]string{}, cliE2EVisibleCommands...)
		commandsAndAliases = append(commandsAndAliases, "day-after", "class-messages")
		for _, command := range commandsAndAliases {
			command := command
			t.Run(command, func(t *testing.T) {
				stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, command, "--help")
				if stderr != "" {
					t.Fatalf("cli-top %s --help wrote to stderr:\n%s", command, stderr)
				}
				if !strings.Contains(stdout, "Usage:") {
					t.Fatalf("cli-top %s --help did not render usage:\n%s", command, stdout)
				}
			})
		}

		if got := latestJSONRequests.Load(); got != requestsBefore {
			t.Fatalf("help commands made %d latest.json requests, want 0", got-requestsBefore)
		}
	})

	t.Run("version", func(t *testing.T) {
		cwd, env, _ := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, "--version")
		if stderr != "" {
			t.Fatalf("cli-top --version wrote to stderr:\n%s", stderr)
		}
		want := fmt.Sprintf("Version: %s", debug.Version)
		if got := strings.TrimSpace(stdout); got != want {
			t.Fatalf("cli-top --version = %q, want %q", got, want)
		}
		if got := latestJSONRequests.Load(); got != requestsBefore {
			t.Fatalf("--version made %d latest.json requests, want 0", got-requestsBefore)
		}
	})

	t.Run("completion generates bash script", func(t *testing.T) {
		cwd, env, _ := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, "completion", "bash")
		if stderr != "" {
			t.Fatalf("cli-top completion bash wrote to stderr:\n%s", stderr)
		}
		for _, want := range []string{"__start_cli-top", "complete -o default -F __start_cli-top cli-top"} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("completion script is missing %q:\n%s", want, stdout)
			}
		}
		if got := latestJSONRequests.Load(); got != requestsBefore {
			t.Fatalf("completion made %d latest.json requests, want 0", got-requestsBefore)
		}
	})

	t.Run("root welcome", func(t *testing.T) {
		cwd, env, configPath := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		writeCLIE2EConfig(t, configPath, "UUID=e2e-welcome-uuid\n", 0o600)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd)
		if stderr != "" {
			t.Fatalf("cli-top wrote to stderr:\n%s", stderr)
		}
		if !strings.Contains(stdout, "Welcome to CLI-TOP!") || !strings.Contains(stdout, `Use "cli-top help"`) {
			t.Fatalf("root command did not render the welcome screen:\n%s", stdout)
		}
		if got := latestJSONRequests.Load(); got != requestsBefore+1 {
			t.Fatalf("root command made %d latest.json requests, want 1", got-requestsBefore)
		}
	})

	t.Run("update confirms current version", func(t *testing.T) {
		cwd, env, configPath := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		writeCLIE2EConfig(t, configPath, "UUID=e2e-update-uuid\n", 0o600)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, "--update")
		if stderr != "" {
			t.Fatalf("cli-top --update wrote to stderr:\n%s", stderr)
		}
		if !strings.Contains(stdout, "You are using the latest stable version of cli-top.") {
			t.Fatalf("update command did not report the current version:\n%s", stdout)
		}
		if got := latestJSONRequests.Load(); got != requestsBefore+2 {
			t.Fatalf("--update made %d latest.json requests, want 2", got-requestsBefore)
		}
	})

	t.Run("login encrypts credentials", func(t *testing.T) {
		cwd, env, configPath := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		const (
			userUUID = "e2e-login-uuid"
			username = "22bce9999"
			password = "e2e-plaintext-password"
		)
		writeCLIE2EConfig(t, configPath, "UUID="+userUUID+"\n", 0o644)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd,
			"login", "--username", username, "--password", password,
		)
		if stderr != "" {
			t.Fatalf("cli-top login wrote to stderr:\n%s", stderr)
		}
		if !strings.Contains(stdout, "Username and encrypted password stored") {
			t.Fatalf("cli-top login did not report success:\n%s", stdout)
		}
		if got := latestJSONRequests.Load(); got != requestsBefore+1 {
			t.Fatalf("login made %d latest.json requests, want 1", got-requestsBefore)
		}

		raw, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read login config: %v", err)
		}
		if bytes.Contains(raw, []byte(password)) {
			t.Fatalf("login config contains plaintext password: %s", raw)
		}
		config, err := godotenv.Read(configPath)
		if err != nil {
			t.Fatalf("parse login config: %v", err)
		}
		if got := config["UUID"]; got != userUUID {
			t.Fatalf("UUID = %q, want %q", got, userUUID)
		}
		if got := config["VTOP_USERNAME"]; got != strings.ToUpper(username) {
			t.Fatalf("VTOP_USERNAME = %q, want %q", got, strings.ToUpper(username))
		}
		encryptedPassword := config["PASSWORD"]
		if encryptedPassword == "" || encryptedPassword == password {
			t.Fatalf("PASSWORD was not encrypted: %q", encryptedPassword)
		}
		decodedPassword, err := base64.URLEncoding.DecodeString(encryptedPassword)
		if err != nil {
			t.Fatalf("PASSWORD is not URL-safe base64 ciphertext: %v", err)
		}
		if len(decodedPassword) <= len(password) {
			t.Fatalf("ciphertext length = %d, want more than plaintext length %d", len(decodedPassword), len(password))
		}
		if got := len(config["KEY"]); got != 32 {
			t.Fatalf("KEY length = %d, want 32", got)
		}

		if runtime.GOOS != "windows" && runtime.GOOS != "plan9" {
			info, err := os.Stat(configPath)
			if err != nil {
				t.Fatalf("stat login config: %v", err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("login config mode = %04o, want 0600", got)
			}
		}
	})

	t.Run("logout preserves only UUID", func(t *testing.T) {
		cwd, env, configPath := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		const userUUID = "e2e-logout-uuid"
		writeCLIE2EConfig(t, configPath, strings.Join([]string{
			"UUID=" + userUUID,
			`VTOP_USERNAME="22BCE9999"`,
			`PASSWORD="encrypted-password"`,
			`KEY="01234567890123456789012345678901"`,
			`CSRF="csrf"`,
			`JSESSIONID="session"`,
			`SERVERID="server"`,
			`REGNO="22BCE9999"`,
		}, "\n")+"\n", 0o600)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, "logout")
		if stderr != "" {
			t.Fatalf("cli-top logout wrote to stderr:\n%s", stderr)
		}
		if !strings.Contains(stdout, "Logged out successfully.") {
			t.Fatalf("cli-top logout did not report success:\n%s", stdout)
		}
		if got := latestJSONRequests.Load(); got != requestsBefore+1 {
			t.Fatalf("logout made %d latest.json requests, want 1", got-requestsBefore)
		}

		config, err := godotenv.Read(configPath)
		if err != nil {
			t.Fatalf("parse logout config: %v", err)
		}
		want := map[string]string{"UUID": userUUID}
		if !reflect.DeepEqual(config, want) {
			t.Fatalf("logout config = %#v, want %#v", config, want)
		}
	})

	t.Run("proxy rejects empty credentials as JSON", func(t *testing.T) {
		cwd, env, configPath := newCLIE2ESandbox(t, latestJSONServer.URL, networkTrap.URL)
		writeCLIE2EConfig(t, configPath, "UUID=e2e-proxy-uuid\n", 0o600)
		requestsBefore := latestJSONRequests.Load()

		stdout, stderr := runCLIE2ECommand(t, binPath, env, cwd, "proxy", "", "", "profile")
		if stderr != "" {
			t.Fatalf("cli-top proxy wrote to stderr:\n%s", stderr)
		}
		if got := latestJSONRequests.Load(); got != requestsBefore+1 {
			t.Fatalf("proxy made %d latest.json requests, want 1", got-requestsBefore)
		}

		var response struct {
			Success bool   `json:"success"`
			Command string `json:"command"`
			Version string `json:"version"`
			Error   *struct {
				Kind    string `json:"kind"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(stdout), &response); err != nil {
			t.Fatalf("proxy output is not valid JSON: %v\n%s", err, stdout)
		}
		if response.Success {
			t.Fatalf("proxy success = true, want false: %s", stdout)
		}
		if response.Command != "profile" {
			t.Fatalf("proxy command = %q, want profile", response.Command)
		}
		if response.Version != debug.Version {
			t.Fatalf("proxy version = %q, want %q", response.Version, debug.Version)
		}
		if response.Error == nil || response.Error.Kind != "auth_failed" || response.Error.Message != "missing VTOP credentials" {
			t.Fatalf("proxy error = %#v, want auth_failed for missing credentials", response.Error)
		}
	})

	if got := unexpectedNetworkRequests.Load(); got != 0 {
		t.Fatalf("CLI attempted %d external network requests", got)
	}
}

func newCLIE2ESandbox(t *testing.T, latestJSONURL, networkTrapURL string) (cwd string, env []string, configPath string) {
	t.Helper()

	root := t.TempDir()
	home := filepath.Join(root, "home")
	cwd = filepath.Join(root, "work")
	for _, dir := range []string{home, cwd, filepath.Join(root, "config"), filepath.Join(root, "appdata")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("create CLI E2E sandbox directory: %v", err)
		}
	}

	removed := map[string]bool{
		"HOME": true, "USERPROFILE": true, "XDG_CONFIG_HOME": true,
		"APPDATA": true, "LOCALAPPDATA": true,
		"CLI_TOP_LATEST_JSON_URL": true, "CLI_TOP_PROXY_MODE": true,
		"NO_COLOR": true, "TERM": true,
		"HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "NO_PROXY": true,
		"http_proxy": true, "https_proxy": true, "all_proxy": true, "no_proxy": true,
		"UUID": true, "UNREGISTERED_UUID": true,
		"VTOP_USERNAME": true, "PASSWORD": true, "KEY": true,
		"CSRF": true, "JSESSIONID": true, "SERVERID": true, "REGNO": true,
	}
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if !removed[key] {
			env = append(env, value)
		}
	}
	env = append(env,
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CONFIG_HOME="+filepath.Join(root, "config"),
		"APPDATA="+filepath.Join(root, "appdata"),
		"LOCALAPPDATA="+filepath.Join(root, "appdata"),
		"CLI_TOP_LATEST_JSON_URL="+latestJSONURL,
		"NO_COLOR=1",
		"TERM=dumb",
		"HTTP_PROXY="+networkTrapURL,
		"HTTPS_PROXY="+networkTrapURL,
		"ALL_PROXY="+networkTrapURL,
		"NO_PROXY=127.0.0.1,localhost",
		"http_proxy="+networkTrapURL,
		"https_proxy="+networkTrapURL,
		"all_proxy="+networkTrapURL,
		"no_proxy=127.0.0.1,localhost",
	)

	return cwd, env, filepath.Join(cwd, "cli-top-config.env")
}

func writeCLIE2EConfig(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write CLI E2E config: %v", err)
	}
	if runtime.GOOS != "windows" && runtime.GOOS != "plan9" {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatalf("chmod CLI E2E config: %v", err)
		}
	}
}

func runCLIE2ECommand(t *testing.T, binPath string, env []string, cwd string, args ...string) (stdout, stderr string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, binPath, args...)
	command.Dir = cwd
	command.Env = env
	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer
	err := command.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("cli-top %q timed out\nstdout:\n%s\nstderr:\n%s", args, stdoutBuffer.String(), stderrBuffer.String())
	}
	if err != nil {
		t.Fatalf("cli-top %q failed: %v\nstdout:\n%s\nstderr:\n%s", args, err, stdoutBuffer.String(), stderrBuffer.String())
	}
	return stdoutBuffer.String(), stderrBuffer.String()
}
