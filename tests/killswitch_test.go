package tests

import (
	"bytes"
	"cli-top/helpers"
	"cli-top/types"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type MockServer struct {
	server *httptest.Server
}

func NewMockServer(killSwitch int) *MockServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versionInfo := types.VersionInfo{
			Version:    "1.0.0",
			KillSwitch: killSwitch,
		}
		json.NewEncoder(w).Encode(versionInfo)
	})

	server := httptest.NewServer(handler)
	return &MockServer{server: server}
}

func (m *MockServer) Close() {
	m.server.Close()
}

func (m *MockServer) URL() string {
	return m.server.URL
}

func getRealCaptcha() string {
	img := image.NewRGBA(image.Rect(0, 0, 200, 40))

	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())

	return "data:image/jpeg;base64," + base64Img
}

func TestKillSwitchScenarios(t *testing.T) {
	originalURL := helpers.GetLatestJSONURL()

	testCases := []struct {
		name       string
		killSwitch int
		expected   int
		verifyFunc func(*testing.T, *MockServer)
	}{
		{
			name:       "KillSwitch 0 - Allow automated captcha",
			killSwitch: 0,
			expected:   0,
			verifyFunc: func(t *testing.T, mockServer *MockServer) {
				captcha := getRealCaptcha()
				if captcha == "" {
					t.Skip("Could not get real captcha, skipping test")
					return
				}

				result := helpers.SolveCaptcha(captcha)
				if result == "disabled" {
					t.Error("Automated captcha solving should be enabled for killswitch 0")
				}
				os.Remove("captcha.jpg")
			},
		},
		{
			name:       "KillSwitch 1 - Disable automated captcha",
			killSwitch: 1,
			expected:   1,
			verifyFunc: func(t *testing.T, mockServer *MockServer) {
				helpers.SetLatestJSONURL(mockServer.URL())
				helpers.CheckKillSwitch()

				captcha := getRealCaptcha()
				if captcha == "" {
					t.Skip("Could not get real captcha, skipping test")
					return
				}

				oldStdin := os.Stdin
				defer func() { os.Stdin = oldStdin }()

				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				os.Stdin = r

				go func() {
					defer w.Close()
					w.Write([]byte("TEST123\n"))
				}()

				result := helpers.SolveCaptcha(captcha)
				if result != "TEST123" {
					t.Error("Expected manual captcha input to be returned for killswitch 1")
				}
				if _, err := os.Stat("captcha.jpg"); os.IsNotExist(err) {
					t.Error("captcha.jpg should be created for manual solving")
				}
				os.Remove("captcha.jpg")
			},
		},
		{
			name:       "KillSwitch 2 - Disable app completely",
			killSwitch: 2,
			expected:   2,
			verifyFunc: func(t *testing.T, mockServer *MockServer) {
				tmpDir := t.TempDir()
				tmpBin := filepath.Join(tmpDir, "cli-top-test.exe")

				if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\ngo 1.21\n"), 0644); err != nil {
					t.Fatal(err)
				}

				testCode := `
					package main
					import "fmt"
					func main() {
						fmt.Println("This version of cli-top has been decommissioned.")
					}
				`
				tmpGo := filepath.Join(tmpDir, "main.go")
				if err := os.WriteFile(tmpGo, []byte(testCode), 0644); err != nil {
					t.Fatal(err)
				}

				cmd := exec.Command("go", "build", "-o", tmpBin, tmpGo)
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatal(err)
				}

				output, err := exec.Command(tmpBin).CombinedOutput()
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Contains(output, []byte("This version of cli-top has been decommissioned.")) {
					t.Error("Expected decommissioned message for killswitch 2")
				}
			},
		},
		{
			name:       "KillSwitch 3 - Open VTOP in browser",
			killSwitch: 3,
			expected:   3,
			verifyFunc: func(t *testing.T, mockServer *MockServer) {
				url := "https://vtop.vit.ac.in"
				err := helpers.OpenURLInBrowser(url)
				if err != nil {
					t.Errorf("Failed to open URL in browser: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := NewMockServer(tc.killSwitch)
			defer mockServer.Close()

			helpers.SetLatestJSONURL(mockServer.URL())

			result := helpers.CheckKillSwitch()
			if result != tc.expected {
				t.Errorf("Expected killswitch value %d, got %d", tc.expected, result)
			}

			if tc.verifyFunc != nil {
				tc.verifyFunc(t, mockServer)
			}
		})
	}

	helpers.SetLatestJSONURL(originalURL)
}

func TestBrowserOpening(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping browser opening test in CI environment")
	}

	url := "https://vtop.vit.ac.in"
	err := helpers.OpenURLInBrowser(url)
	if err != nil {
		t.Errorf("Failed to open URL in browser: %v", err)
	}
}
