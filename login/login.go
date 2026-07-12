package login

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// performLogin executes a single login attempt and returns any high-level error type.
// errorType is one of: "", "invalid_captcha", "invalid_credentials", "max_attempts", "network_error".
func performLogin(userInfo types.LogIn, cookies types.Cookies, captcha string) (types.Cookies, string) {
	client := *helpers.GetHTTPClient()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	data := strings.NewReader(url.Values{
		"_csrf":      {cookies.CSRF},
		"username":   {userInfo.Username},
		"password":   {userInfo.Password},
		"captchaStr": {captcha},
	}.Encode())
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/login", data)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return types.Cookies{}, "network_error"
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Origin", "https://vtop.vit.ac.in")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return types.Cookies{}, "network_error"
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		if debug.Debug {
			helpers.Printf("VTOP login request failed with status %s\n", resp.Status)
		}
		return types.Cookies{}, "network_error"
	}

	if errType := errorCheck(cookies); errType != "" {
		return types.Cookies{}, errType
	}

	tokens := helpers.ExtractCookies(resp)

	return tokens, ""
}

// errorCheck inspects the login/error page to classify failures.
// Returns one of: "", "invalid_captcha", "invalid_credentials", "max_attempts", "network_error".
func errorCheck(cookies types.Cookies) string {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/login/error", nil)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return "network_error"
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return "network_error"
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if debug.Debug {
			helpers.Printf("VTOP login error check failed with status %s\n", resp.Status)
		}
		return "network_error"
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return "network_error"
	}

	if strings.Contains(string(bodyText), "Invalid Captcha") {
		if debug.Debug {
			helpers.Println("\nInvalid Captcha detected during login.")
		}
		return "invalid_captcha"
	}
	if strings.Contains(string(bodyText), "Invalid LoginId/Password") {
		helpers.Println("\nInvalid LoginId/Password. Please check your cli-top config and try again...")
		return "invalid_credentials"
	}

	if strings.Contains(string(bodyText), "Invalid Username/Password") {
		helpers.Println("\nInvalid Username/Password. Please check your cli-top config and try again...")
		return "invalid_credentials"
	}

	if strings.Contains(string(bodyText), "Maximum Fail Attempts") {
		helpers.Println("\nNumber Of Maximum Fail Attempts Reached. Use Forgot Password on VTOP to reset your password.")
		return "max_attempts"
	}

	return ""
}

func Login(regNo string, password string) types.Cookies {
	const maxCaptchaRetries = 3

	userInfo := types.LogIn{
		Username: regNo,
		Password: password,
	}

	for attempt := 0; attempt < maxCaptchaRetries; attempt++ {
		vtopTokens, captcha, err := getLoginPage()
		if err != nil {
			if debug.Debug {
				helpers.Println(err)
			}
			return types.Cookies{}
		}
		loginCreds, errType := performLogin(userInfo, vtopTokens, captcha)

		if errType == "" {
			vtopTokens.JSESSIONID = loginCreds.JSESSIONID
			return vtopTokens
		}

		if errType == "invalid_captcha" {
			if debug.Debug {
				helpers.Printf("Login attempt %d failed due to invalid captcha, retrying...\n", attempt+1)
			}
			continue
		}

		return types.Cookies{}
	}

	helpers.Println("\nCaptcha could not be solved after multiple attempts. Please try again later.")
	return types.Cookies{}
}

func HomePage(vtopTokens types.Cookies) (types.Cookies, string) {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/init/page", nil)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return vtopTokens, ""
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", vtopTokens.JSESSIONID, vtopTokens.SERVERID))
	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return vtopTokens, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if debug.Debug {
			helpers.Printf("VTOP home page request failed with status %s\n", resp.Status)
		}
		return vtopTokens, ""
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return vtopTokens, ""
	}

	if strings.Contains(string(bodyText), "Session Timed Out") {
		if debug.Debug {
			helpers.Println("Session Timed Out, login failed. Retrying...")
		}
		return vtopTokens, ""
	}

	vtopTokens.CSRF = helpers.ExtractCSRF2(string(bodyText))

	RegNo, err := helpers.ExtractRegNo(string(bodyText))
	if err != nil && debug.Debug {
		helpers.Println("Error:", err)
	}

	return vtopTokens, RegNo
}
