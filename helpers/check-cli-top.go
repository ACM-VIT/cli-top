package helpers

import (
	"cli-top/debug"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func CheckUpdate() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://cli-top.acmvit.in/latest.json", nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if !strings.Contains(string(bodyText), debug.Version) {
		fmt.Println("A new version of cli-top is available.\nCheck out: https://cli-top.acmvit.in/ for the latest release.")
	} else {
		fmt.Println("You are using the latest stable version of cli-top.")
	}

}

func CheckKillSwitch() int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://cli-top.acmvit.in/latest.json", nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if strings.Contains(string(bodyText), "\"killSwitch\": 2") {
		return 2
	} else if strings.Contains(string(bodyText), "\"killSwitch\": 0") {
		return 0
	}
	return 1
}
