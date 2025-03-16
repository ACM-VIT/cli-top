package features

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"cli-top/debug"
)

type VersionInfo struct {
	Version    string `json:"version"`
	KillSwitch int    `json:"killSwitch"`
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

	var versionInfo VersionInfo
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

	// Windows-specific update logic
	if runtime.GOOS == "windows" {
		updatePath := realPath + ".update.exe"
		err = os.WriteFile(updatePath, data, 0755)
		if err != nil {
			fmt.Println("Error writing updated binary:", err)
			return
		}
		// Create a batch script to replace the binary after exiting
		batchScript := fmt.Sprintf(`@echo off
ping 127.0.0.1 -n 5 > nul
move /Y "%s" "%s"
start "" "%s"
del "%%~f0"`, updatePath, realPath, realPath)
		batPath := filepath.Join(filepath.Dir(realPath), "update.bat")
		err = os.WriteFile(batPath, []byte(batchScript), 0644)
		if err != nil {
			fmt.Println("Error writing update batch file:", err)
			return
		}
		// Launch the batch script
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

// Add this new helper function at the end of the file
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

	// Check if file is readonly
	if info.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("binary file %s is read-only", path)
	}

	return nil
}
