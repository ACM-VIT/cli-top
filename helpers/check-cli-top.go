package helpers

import (
	"cli-top/debug"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func CheckUpdate() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://cli-top.acmvit.in/latest.json", nil)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	if !strings.Contains(string(bodyText), debug.Version) {
		fmt.Println("A new version of cli-top is available.\nCheck out: https://cli-top.acmvit.in/ for the latest release.")
	} else {
		fmt.Println("You are using the latest stable version of cli-top.")
	}

}

func CheckKillSwitch() int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://cli-top.acmvit.in/latest.json", nil)
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

	if strings.Contains(string(bodyText), "\"killSwitch\": 2") {
		// Disables the app completely
		return 2
	} else if strings.Contains(string(bodyText), "\"killSwitch\": 0") {
		// Allows automated captcha solver to run
		return 0
	}
	// Disables the automated captcha solver - manual captcha solving required
	return 1
}
