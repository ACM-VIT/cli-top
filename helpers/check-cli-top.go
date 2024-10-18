package helpers

import (
	"cli-top/debug"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func CheckUpdate() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://cli-top-website.vercel.app/latest.json", nil)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating request:", err)
		}
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error performing request:", err)
		}
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error reading response body:", err)
		}
		return
	}

	if !strings.Contains(string(bodyText), debug.Version) {
		fmt.Println("A new version of cli-top is available.\nCheck out: https://cli-top.acmvit.in/ for the latest release.")
	} else {
		fmt.Println("You are using the latest stable version of cli-top.")
	}
}

func CheckKillSwitch() int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://cli-top-website.vercel.app/latest.json", nil)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating request:", err)
		}
		return 1
	}

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error performing request:", err)
		}
		return 1
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error reading response body:", err)
		}
		return 1
	}

	if strings.Contains(string(bodyText), "\"killSwitch\": 2") {
		return 2
	} else if strings.Contains(string(bodyText), "\"killSwitch\": 0") {
		return 0
	}
	return 1
}
