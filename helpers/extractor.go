package helpers

import (
	"cli-top/debug"
	"cli-top/types"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	csrfLineRegex     = regexp.MustCompile(`.*csrfValue.*`)
	csrfPatternRegex  = regexp.MustCompile(`var csrfValue = /\*(.*?)\*/'.*';`)
	csrfAltRegex      = regexp.MustCompile(`var csrfValue = "([a-fA-F0-9-]+)";`)
	regNoRegex        = regexp.MustCompile(`let id\s*=\s*"(.*?)";`)
)

func extractImageSrc(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil && debug.Debug {
		return "", err
	}

	src := doc.Find("#captchaBlock img").AttrOr("src", "")
	if src == "" {
		if debug.Debug {
			fmt.Println("No captcha image found, retrying...")
		}
		return "nocaptcha", nil
	}

	return src, nil
}

func ExtractImage(html string) string {
	src, err := extractImageSrc(html)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return ""
	}
	// fmt.Println(src)

	return src
}

func ExtractCookies(resp *http.Response) types.Cookies {
	if resp == nil {
		fmt.Println("Response is nil")
	}

	secrets := make(map[string]string)
	respCookies := resp.Cookies()
	for _, cookie := range respCookies {
		// fmt.Println(cookie.Name, cookie.Value)
		secrets[cookie.Name] = cookie.Value
	}

	// if debug.Debug {
	// 	fmt.Println("(Helper - ExtractCookies):", secrets)
	// }

	cookies := types.Cookies{
		SERVERID:   secrets["SERVERID"],
		CSRF:       "",
		JSESSIONID: secrets["JSESSIONID"],
	}

	return cookies
}

func ExtractBodyText(resp *http.Response) string {
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println(err)
		return ""
	}

	return string(bodyText)
}

func ExtractCSRF(bodyString string) string {

	// Define a regular expression that matches lines containing "csrfValue"
	// Find the matches
	lines := csrfLineRegex.FindAllString(bodyString, -1)

	csrf := ""
	// Compile pattern for extracting csrfValue only once

	// Iterate over the lines
	for _, line := range lines {
		// fmt.Println("Found line:", line)

		// Find the match
		match := csrfPatternRegex.FindStringSubmatch(line)

		// If a match was found, print the value of the variable
		if len(match) > 1 {
			csrf = match[1]
			break
		}
	}

	if debug.Debug {
		fmt.Println("(Helper - ExtractCSRF):", csrf)
	}

	return csrf
}

func ExtractCSRF2(bodyString string) string {
	matches := csrfAltRegex.FindStringSubmatch(bodyString)
	csrf := ""

	if len(matches) > 1 {
		csrfValue := matches[1]
		csrf = csrfValue
	}

	if debug.Debug {
		fmt.Println("(Helper - ExtractCSRF2):", csrf)
	}

	return csrf
}
func ExtractRegNo(bodyString string) (string, error) {
	// Define a regular expression to match the assignment of id variable
	// Find the first match
	match := regNoRegex.FindStringSubmatch(bodyString)
	if len(match) != 2 {
		return "", fmt.Errorf("unable to extract id from HTML")
	}

	return match[1], nil
}
