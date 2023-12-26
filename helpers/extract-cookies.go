package helpers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"vtop-cli/types"
)

func ExtractCookies(resp *http.Response) types.Cookies {
	if resp == nil {
		log.Fatal("Response is nil")
	}

	secrets := make(map[string]string)
	respCookies := resp.Cookies()
	for _, cookie := range respCookies {
		// fmt.Println(cookie.Name, cookie.Value)
		secrets[cookie.Name] = cookie.Value
	}

	fmt.Println("(Helper - ExtractCookies) VTOP Cookies:", secrets)

	cookies := types.Cookies{
		SERVERID:   secrets["SERVERID"],
		CSRF:       ExtractCSRF(resp),
		JSESSIONID: secrets["JSESSIONID"],
	}

	return cookies
}

func ExtractCSRF(resp *http.Response) string {

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("%s\n", bodyText)

	// Convert bodyText to a string
	bodyString := string(bodyText)

	// Define a regular expression that matches lines containing "csrfValue"
	re := regexp.MustCompile(`.*csrfValue.*`)

	// Find the matches
	lines := re.FindAllString(bodyString, -1)

	csrf := ""
	// Iterate over the lines
	for _, line := range lines {
		// fmt.Println("Found line:", line)

		// Define a regular expression that matches the pattern of the variable assignment
		re := regexp.MustCompile(`var csrfValue = /\*(.*?)\*/'.*';`)

		// Find the match
		match := re.FindStringSubmatch(line)

		// If a match was found, print the value of the variable
		if len(match) > 1 {
			csrf = match[1]
			break
		}
	}

	return csrf
}
