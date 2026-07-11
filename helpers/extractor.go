package helpers

import (
	"cli-top/types"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	csrfPatternRegex = regexp.MustCompile(`var csrfValue = /\*(.*?)\*/'.*';`)
	csrfAltRegex     = regexp.MustCompile(`var csrfValue = "([a-fA-F0-9-]+)";`)
	regNoRegex       = regexp.MustCompile(`let id\s*=\s*"(.*?)";`)
)

func extractImageSrc(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	src := doc.Find("#captchaBlock img").AttrOr("src", "")
	if src == "" {
		return "nocaptcha", nil
	}

	return src, nil
}

func ExtractImage(html string) string {
	src, err := extractImageSrc(html)
	if err != nil {
		return ""
	}

	return src
}

func ExtractCookies(resp *http.Response) types.Cookies {
	if resp == nil {
		return types.Cookies{}
	}

	secrets := make(map[string]string)
	respCookies := resp.Cookies()
	for _, cookie := range respCookies {
		secrets[cookie.Name] = cookie.Value
	}

	cookies := types.Cookies{
		SERVERID:   secrets["SERVERID"],
		CSRF:       "",
		JSESSIONID: secrets["JSESSIONID"],
	}

	return cookies
}

func ExtractCSRF(bodyString string) string {
	match := csrfPatternRegex.FindStringSubmatch(bodyString)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func ExtractCSRF2(bodyString string) string {
	matches := csrfAltRegex.FindStringSubmatch(bodyString)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
func ExtractRegNo(bodyString string) (string, error) {
	match := regNoRegex.FindStringSubmatch(bodyString)
	if len(match) != 2 {
		return "", fmt.Errorf("unable to extract id from HTML")
	}

	return match[1], nil
}
