package login

import (
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func getSessionServer() (types.Cookies, error) {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/", nil)
	if err != nil {
		return types.Cookies{}, err
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Sec-Fetch-Site", "none")
	resp, err := client.Do(req)
	if err != nil {
		return types.Cookies{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return types.Cookies{}, fmt.Errorf("VTOP session request failed with status %s", resp.Status)
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.Cookies{}, err
	}

	vtopCookies := helpers.ExtractCookies(resp)
	vtopCookies.CSRF = helpers.ExtractCSRF(string(bodyText))

	return vtopCookies, nil
}

func getLoginPage() (types.Cookies, string, error) {
	const maxPreloginAttempts = 3
	client := helpers.GetHTTPClient()
	for attempt := 0; attempt < maxPreloginAttempts; attempt++ {
		cookies, err := getSessionServer()
		if err != nil {
			return types.Cookies{}, "", err
		}

		data := strings.NewReader(url.Values{
			"_csrf": {cookies.CSRF},
			"flag":  {"VTOP"},
		}.Encode())
		req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/prelogin/setup", data)
		if err != nil {
			return types.Cookies{}, "", err
		}
		helpers.SetVtopHeaders(req)
		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("Origin", "https://vtop.vit.ac.in")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/open/page")
		req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
		resp, err := client.Do(req)
		if err != nil {
			return types.Cookies{}, "", err
		}

		bodyText, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return types.Cookies{}, "", readErr
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return types.Cookies{}, "", fmt.Errorf("VTOP prelogin request failed with status %s", resp.Status)
		}

		captchaImage := helpers.ExtractImage(string(bodyText))
		if captchaImage == "" || captchaImage == "nocaptcha" {
			continue
		}

		captcha := helpers.SolveCaptcha(captchaImage)
		if captcha == "" || captcha == "disabled" {
			return types.Cookies{}, "", fmt.Errorf("captcha solving is unavailable")
		}

		return cookies, captcha, nil
	}

	return types.Cookies{}, "", fmt.Errorf("VTOP did not provide a captcha after %d attempts", maxPreloginAttempts)
}
