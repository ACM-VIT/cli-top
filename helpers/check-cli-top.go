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

	"github.com/spf13/viper"
)

var latestJSONURL = "https://cli-top.acmvit.in/latest.json"

func SetLatestJSONURL(url string) {
	latestJSONURL = url
}

func GetLatestJSONURL() string {
	return latestJSONURL
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading update info:", err)
		return
	}

	var versionInfo types.VersionInfo
	if err := json.Unmarshal(body, &versionInfo); err != nil {
		fmt.Println("Error parsing version info:", err)
		return
	}

	if versionInfo.Version == debug.Version {
		fmt.Println("You are using the latest stable version of cli-top.")
		return
	}

	fmt.Printf("A new version %s is available. Downloading update...\n", versionInfo.Version)

	// Select download URL based on current OS.
	baseURL := "https://github.com/technical-director-acmvit/cli-top-website/raw/main/buildFiles"
	var downloadURL string
	switch runtime.GOOS {
	case "windows":
		downloadURL = fmt.Sprintf("%s/v%s/cli-top-windows-installer_v%s.exe", baseURL, versionInfo.Version, versionInfo.Version)
	case "linux":
		downloadURL = fmt.Sprintf("%s/v%s/cli-top-linux_v%s.zip", baseURL, versionInfo.Version, versionInfo.Version)
	case "android":
		downloadURL = fmt.Sprintf("%s/v%s/cli-top-android_v%s.zip", baseURL, versionInfo.Version, versionInfo.Version)
	case "darwin":
		downloadURL = fmt.Sprintf("%s/v%s/cli-top-macos_v%s.zip", baseURL, versionInfo.Version, versionInfo.Version)
	default:
		fmt.Println("Auto update is not supported for your operating system:", runtime.GOOS)
		return
	}

	// Download the update.
	resp, err = http.Get(downloadURL)
	if err != nil {
		fmt.Println("Error downloading update:", err)
		return
	}
	defer resp.Body.Close()

	tmpFile := filepath.Join(os.TempDir(), "cli-top-update")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading update file:", err)
		return
	}

	err = os.WriteFile(tmpFile, data, 0755)
	if err != nil {
		fmt.Println("Error writing temporary update file:", err)
		return
	}

	if strings.HasSuffix(downloadURL, ".zip") {
		updatedBinary, err := extractBinaryFromZip(tmpFile)
		if err != nil {
			fmt.Println("Error extracting binary from zip:", err)
			return
		}
		data = updatedBinary
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Println("Error determining executable path:", err)
		return
	}

	// Resolve any symlinks to get the actual binary path
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		fmt.Println("Error resolving symbolic links:", err)
		return
	}

	// Check if we have write permissions to the binary location
	if err := checkWritePermission(realPath); err != nil {
		fmt.Printf("Insufficient permissions to update. Please run with elevated privileges: %v\n", err)
		return
	}

	if runtime.GOOS == "windows" {
		updatePath := realPath + ".update.exe"
		err = os.WriteFile(updatePath, data, 0755)
		if err != nil {
			fmt.Println("Error writing updated binary:", err)
			return
		}

		batchScript := fmt.Sprintf(`@echo off
setlocal
REM Wait for cli-top.exe to exit
:loop
tasklist | find /I "cli-top.exe" >nul
if not errorlevel 1 (
    timeout /t 1 >nul
    goto loop
)
REM Replace the old exe with the new one
move /Y "%s" "%s"
REM Restart the app
start "" "%s"
endlocal
del "%%~f0"
`, updatePath, realPath, realPath)

		batPath := filepath.Join(filepath.Dir(realPath), "update.bat")
		err = os.WriteFile(batPath, []byte(batchScript), 0644)
		if err != nil {
			fmt.Println("Error writing update batch file:", err)
			return
		}

		err = exec.Command("cmd", "/C", "start", "", batPath).Start()
		if err != nil {
			fmt.Println("Error launching update script:", err)
			return
		}

		fmt.Println("Update is in progress. The application will restart shortly.")
		os.Exit(0)
	}

	// ----- Non-Windows update code -----
	backupPath := realPath + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		os.Remove(backupPath)
	}

	// Create new backup
	err = os.Rename(realPath, backupPath)
	if err != nil {
		fmt.Println("Error backing up current binary:", err)
		return
	}

	err = os.WriteFile(realPath, data, 0755)
	if err != nil {
		fmt.Println("Error writing updated binary:", err)
		// Attempt to restore backup
		os.Rename(backupPath, realPath)
		return
	}

	fmt.Printf("Successfully updated to version %s. Restart the application to use the new version.\n", versionInfo.Version)
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

func ShouldShowUpdateNotification() (bool, string) {
	lastNotifiedVersion := viper.GetString("LAST_UPDATE_NOTIFIED_VERSION")
	currentVersion := debug.Version

	if lastNotifiedVersion == currentVersion {
		return false, ""
	}

	updateAvailable, latestVersion, err := CheckUpdateSilently()
	if err != nil || !updateAvailable {
		return false, ""
	}

	viper.Set("LAST_UPDATE_NOTIFIED_VERSION", currentVersion)
	if err := viper.WriteConfig(); err != nil && debug.Debug {
		fmt.Println("Error updating last notified version in config:", err)
	}

	return true, latestVersion
}

func ShowUpdateNotification(latestVersion string) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ A new version of cli-top is available!                  │\n")
	fmt.Printf("│ Current version: %-10s   Latest version: %-10s│\n", debug.Version, latestVersion)
	fmt.Printf("│                                                         │\n")
	fmt.Printf("│ Update with: cli-top -u                                 │\n")
	fmt.Printf("└─────────────────────────────────────────────────────────┘\n")
	fmt.Printf("\n")
}
