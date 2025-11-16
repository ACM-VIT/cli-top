package helpers

import (
	"archive/zip"
	"bufio"
	"cli-top/debug"
	types "cli-top/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

var latestJSONURL = "https://cli-top.acmvit.in/latest.json"

const releaseNotesURL = "https://cli-top.acmvit.in/releases.json"

func SetLatestJSONURL(url string) {
	latestJSONURL = url
}

func GetLatestJSONURL() string {
	return latestJSONURL
}

func displayUpdateLogo() {
	logo := `

	▄████▄   ██▓     ██▓▄▄▄█████▓ ▒█████   ██▓███  
	▒██▀ ▀█  ▓██▒    ▓██▒▓  ██▒ ▓▒▒██▒  ██▒▓██░  ██▒
	▒▓█    ▄ ▒██░    ▒██▒▒ ▓██░ ▒░▒██░  ██▒▓██░ ██▓▒
	▒▓▓▄ ▄██▒▒██░    ░██░░ ▓██▓ ░ ▒██   ██░▒██▄█▓▒ ▒
	▒ ▓███▀ ░░██████▒░██░  ▒██▒ ░ ░ ████▓▒░▒██▒ ░  ░
	░ ░▒ ▒  ░░ ▒░▓  ░░▓    ▒ ░░   ░ ▒░▒░▒░ ▒▓▒░ ░  ░
	  ░  ▒   ░ ░ ▒  ░ ▒ ░    ░      ░ ▒ ▒░ ░▒ ░     
	░          ░ ░    ▒ ░  ░      ░ ░ ░ ▒  ░░       
	░ ░          ░  ░ ░               ░ ░           
	░                                               

	`

	pink := color.New(color.FgHiMagenta)
	cyan := color.New(color.FgHiCyan)

	pink.Print(logo)
	cyan.Println("                    AUTO-UPDATER")
	fmt.Println()
}

// Version info structure to match the JSON response

func CheckUpdate() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://cli-top.acmvit.in/latest.json", nil)

	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}
	if resp == nil {
		fmt.Println("Failed to connect to update server")
		return
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return
	}

	// Parse the response as JSON
	var versionInfo types.VersionInfo
	if err := json.Unmarshal(bodyText, &versionInfo); err != nil {
		if debug.Debug {
			fmt.Println("Error parsing version info:", err)
		}
		// Fallback to the old string comparison method
		if !strings.Contains(string(bodyText), debug.Version) {
			fmt.Println("A new version of cli-top is available.\nCheck out: https://cli-top.acmvit.in/ for the latest release.")
		} else {
			fmt.Println("You are using the latest stable version of cli-top.")
		}
		return
	}

	// Compare versions
	if versionInfo.Version != debug.Version {
		fmt.Printf("A new version %s is available (you have %s).\n", versionInfo.Version, debug.Version)
		fmt.Print("Would you like to update now? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			fmt.Println("Starting update process...")
			Update()
		} else {
			fmt.Println("Update skipped. You can update later with 'cli-top update'")
			fmt.Println("or visit: https://cli-top.acmvit.in/ for the latest release.")
		}
	} else {
		fmt.Println("You are using the latest stable version of cli-top.")
	}
}

func OpenURLInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			if _, err := exec.LookPath("wslview"); err == nil {
				cmd = exec.Command("wslview", url)
			}
		}
		if cmd == nil {
			if _, err := exec.LookPath("xdg-open"); err == nil {
				cmd = exec.Command("xdg-open", url)
			} else if _, err := exec.LookPath("gio"); err == nil {
				cmd = exec.Command("gio", "open", url)
			} else {
				return fmt.Errorf("no suitable browser opener found")
			}
		}
	}
	return cmd.Start()
}

func CheckKillSwitch() int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", latestJSONURL, nil)

	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	if resp == nil {
		fmt.Println()
		fmt.Println("Internet connection not available")
		fmt.Println("Please reconnect and try again")
		fmt.Println()
		os.Exit(1)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	var versionInfo types.VersionInfo
	if err := json.Unmarshal(bodyText, &versionInfo); err != nil && debug.Debug {
		fmt.Println("Error parsing version info:", err)
		return 1
	}

	return versionInfo.KillSwitch
}

// Update checks for a new version and auto-updates the current binary.
func Update() {
	fmt.Println("Checking for updates...")
	resp, err := http.Get("https://cli-top.acmvit.in/latest.json")
	if err != nil {
		fmt.Println("Error checking for update:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var vi types.VersionInfo
	if json.Unmarshal(body, &vi) != nil || vi.Version == debug.Version {
		fmt.Println("You are using the latest stable version of cli-top.")
		return
	}

	fmt.Printf("A new version %s is available. Downloading update…\n", vi.Version)

	candidates := resolveDownloadCandidates(vi, runtime.GOOS)
	if len(candidates) == 0 {
		fmt.Println("Auto-update is not supported on", runtime.GOOS)
		return
	}

	expectZip := runtime.GOOS != "windows"
	data, downloadSource, err := downloadUpdateArtifact(candidates, expectZip)
	if err != nil {
		fmt.Println("Error downloading update:", err)
		fmt.Println("You can install the update manually from https://cli-top.acmvit.in/ or the GitHub releases page.")
		return
	}

	execPath, _ := os.Executable()
	execPath, _ = filepath.EvalSymlinks(execPath)
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", "cls").Run()
		displayUpdateLogo()

		green := color.New(color.FgHiGreen)
		yellow := color.New(color.FgHiYellow)
		cyan := color.New(color.FgHiCyan)

		fmt.Println("┌─────────────────────────────────────────────────────────┐")
		fmt.Printf("│ %-55s │\n", fmt.Sprintf("Updating CLI-TOP to version %s", vi.Version))
		fmt.Println("├─────────────────────────────────────────────────────────┤")
		fmt.Printf("│ Current Version: %-38s │\n", debug.Version)
		fmt.Printf("│ New Version:     %-38s │\n", vi.Version)
		fmt.Println("└─────────────────────────────────────────────────────────┘")
		fmt.Println()
		yellow.Println("WARNING: Please do not close this window during the update!")
		fmt.Println()

		cyan.Print("[*] Preparing installer... ")
		installer := filepath.Join(os.TempDir(), fmt.Sprintf("cli-top_update_%s.exe", vi.Version))
		os.WriteFile(installer, data, 0755)
		green.Println("[DONE]")
		cyan.Print("[*] Creating update script... ")
		bat := filepath.Join(os.TempDir(), "cli-top_update.bat")
		script := fmt.Sprintf(`@echo off
title CLI-TOP Auto-Updater
echo.
echo ========================================================
echo                CLI-TOP AUTO-UPDATER
echo ========================================================
echo.
set FAILED=0
echo [*] Stopping CLI-TOP processes...
taskkill /IM cli-top.exe /F >nul 2>&1
timeout /t 2 /nobreak >nul
echo.
echo [*] Installing update v%s...
start /wait "" "%s" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
if errorlevel 1 (
    echo [-] Installer reported an error.
    echo     You can download the update manually from:
    echo     %s
    set FAILED=1
    goto cleanup
)
echo.
echo [+] Update completed successfully!
echo.
echo [*] Restarting CLI-TOP...
timeout /t 2 /nobreak >nul
start "" "%s"
echo.
echo [*] Cleaning up temporary files...
del "%s" 2>nul
echo.
:cleanup
if %%FAILED%%==1 (
    echo [!] Update process encountered an error. CLI-TOP was not updated.
) else (
    echo [+] Update process completed! Enjoy the new version!
)
timeout /t 3 /nobreak >nul`, vi.Version, installer, downloadSource, execPath, installer)
		os.WriteFile(bat, []byte(script), 0644)
		green.Println("[DONE]")

		fmt.Println()
		green.Println("[*] Starting update process...")
		yellow.Println("    The application will close and restart automatically.")
		fmt.Println()

		time.Sleep(2 * time.Second)

		exec.Command("cmd", "/C", "start", "", bat).Start()

		cyan.Println("[+] Update is in progress. CLI-TOP will restart automatically.")
		fmt.Println()
		os.Exit(0)
	}

	/* ---------- non-Windows path unchanged: download ZIP, replace binary ---------- */
	if expectZip {
		if b, err := extractBinaryFromZipToBytes(data); err == nil {
			data = b
		} else {
			fmt.Println("Error extracting binary:", err)
			return
		}
	}
	backup := execPath + ".bak"
	os.Remove(backup)
	os.Rename(execPath, backup)
	if os.WriteFile(execPath, data, 0755) != nil {
		os.Rename(backup, execPath)
		fmt.Println("Update failed; restored previous version.")
		return
	}
	fmt.Printf("Successfully updated to %s. Restart the application to use the new version.\n", vi.Version)
}

// extractBinaryFromZip extracts the binary file from a zip archive.
// It assumes that the archive contains a single binary.
func extractBinaryFromZip(zipPath string) ([]byte, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var binaryData []byte
	for _, f := range r.File {
		// Skip directories.
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		binaryData, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		// Found a file; assume it is the binary.
		break
	}

	if len(binaryData) == 0 {
		return nil, fmt.Errorf("no binary file found in zip")
	}
	return binaryData, nil
}

// extractBinaryFromZipToBytes extracts the binary file from zip data in memory.
// It assumes that the archive contains a single binary.
func extractBinaryFromZipToBytes(zipData []byte) ([]byte, error) {
	reader := strings.NewReader(string(zipData))
	r, err := zip.NewReader(reader, int64(len(zipData)))
	if err != nil {
		return nil, err
	}

	var binaryData []byte
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		binaryData, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		break
	}

	if len(binaryData) == 0 {
		return nil, fmt.Errorf("no binary file found in zip")
	}
	return binaryData, nil
}

// checkWritePermission checks if we have permission to write to the given path
func checkWritePermission(path string) error {
	// Check if file exists and we have write permission
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Check if we can write to the directory containing the binary
	dir := filepath.Dir(path)
	tmpFile := filepath.Join(dir, ".write_test")
	err = os.WriteFile(tmpFile, []byte{}, 0666)
	if err != nil {
		return fmt.Errorf("cannot write to directory %s: %v", dir, err)
	}
	os.Remove(tmpFile)

	// On Unix-like systems, check if we're root when binary is in system directories
	if runtime.GOOS != "windows" {
		if strings.HasPrefix(dir, "/usr/bin") || strings.HasPrefix(dir, "/usr/local/bin") {
			if os.Geteuid() != 0 {
				return fmt.Errorf("root privileges required to modify %s", dir)
			}
		}
	}

	if info.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("binary file %s is read-only", path)
	}

	return nil
}

func resolveDownloadCandidates(vi types.VersionInfo, goos string) []string {
	seen := make(map[string]struct{})
	urls := make([]string, 0, 5)
	add := func(url string) {
		if url == "" {
			return
		}
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		urls = append(urls, url)
	}

	if vi.Downloads != nil {
		add(vi.Downloads[goos])
	}
	switch goos {
	case "windows":
		add(vi.WindowsURL)
	case "linux":
		add(vi.LinuxURL)
	case "darwin":
		add(vi.MacURL)
	case "android":
		add(vi.AndroidURL)
	}

	add(defaultLegacyDownload(goos, vi.Version))

	filtered := make([]string, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			filtered = append(filtered, url)
		}
	}
	return filtered
}

func defaultLegacyDownload(goos, version string) string {
	base := "https://raw.githubusercontent.com/technical-director-acmvit/cli-top-website/main/buildFiles"
	switch goos {
	case "windows":
		return fmt.Sprintf("%s/v%s/cli-top-windows-installer_v%s.exe", base, version, version)
	case "linux":
		return fmt.Sprintf("%s/v%s/cli-top-linux_v%s.zip", base, version, version)
	case "android":
		return fmt.Sprintf("%s/v%s/cli-top-android_v%s.zip", base, version, version)
	case "darwin":
		return fmt.Sprintf("%s/v%s/cli-top-macos_v%s.zip", base, version, version)
	default:
		return ""
	}
}

func downloadUpdateArtifact(urls []string, expectZip bool) ([]byte, string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	var lastErr error
	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("download failed with status %d for %s", resp.StatusCode, url)
			resp.Body.Close()
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if expectZip {
			if !looksLikeZip(body) {
				lastErr = fmt.Errorf("downloaded file from %s is not a valid ZIP archive", url)
				continue
			}
		} else {
			if !looksLikePE(body) {
				lastErr = fmt.Errorf("downloaded file from %s is not a valid Windows executable", url)
				continue
			}
		}
		return body, url, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no download URLs available")
	}
	return nil, "", lastErr
}

func looksLikePE(data []byte) bool {
	return len(data) > 2 && data[0] == 0x4d && data[1] == 0x5a
}

func looksLikeZip(data []byte) bool {
	return len(data) > 3 && data[0] == 0x50 && data[1] == 0x4b
}

type releaseList struct {
	Releases []releaseEntry `json:"releases"`
}

type releaseEntry struct {
	Version string   `json:"version"`
	Changes []string `json:"changes"`
}

func fetchReleaseHighlight(version string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(releaseNotesURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var list releaseList
	if err := json.Unmarshal(body, &list); err != nil {
		return ""
	}
	for _, rel := range list.Releases {
		if strings.EqualFold(rel.Version, version) {
			if len(rel.Changes) == 0 {
				return ""
			}
			summary := rel.Changes[0]
			if len(rel.Changes) > 1 {
				summary = fmt.Sprintf("%s (and more)", summary)
			}
			return summary
		}
	}
	return ""
}

func CheckUpdateSilently() (bool, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://cli-top.acmvit.in/latest.json", nil)
	if err != nil {
		return false, "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, "", err
	}
	if resp == nil {
		return false, "", fmt.Errorf("failed to connect to update server")
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	var versionInfo types.VersionInfo
	if err := json.Unmarshal(bodyText, &versionInfo); err != nil {
		return false, "", err
	}

	if versionInfo.Version != debug.Version {
		return true, versionInfo.Version, nil
	}

	return false, versionInfo.Version, nil
}

func ShouldShowUpdateNotification() (bool, string, string) {
	lastNotifiedVersion := viper.GetString("LAST_UPDATE_NOTIFIED_VERSION")
	currentVersion := debug.Version

	if lastNotifiedVersion == currentVersion {
		return false, "", ""
	}

	updateAvailable, latestVersion, err := CheckUpdateSilently()
	if err != nil || !updateAvailable {
		return false, "", ""
	}

	viper.Set("LAST_UPDATE_NOTIFIED_VERSION", currentVersion)
	if err := viper.WriteConfig(); err != nil && debug.Debug {
		fmt.Println("Error updating last notified version in config:", err)
	}

	highlight := fetchReleaseHighlight(latestVersion)
	return true, latestVersion, highlight
}

func ShowUpdateNotification(latestVersion, highlight string) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ A new version of cli-top is available!                  │\n")
	fmt.Printf("│ Current version: %-10s   Latest version: %-10s│\n", debug.Version, latestVersion)
	fmt.Printf("│                                                         │\n")
	if highlight != "" {
		short := TruncateWithEllipsis(highlight, 43)
		fmt.Printf("│ What's new: %-43s │\n", short)
		fmt.Printf("│                                                         │\n")
	}
	fmt.Printf("│ Update with: cli-top -u                                 │\n")
	fmt.Printf("└─────────────────────────────────────────────────────────┘\n")
	fmt.Printf("\n")
}
