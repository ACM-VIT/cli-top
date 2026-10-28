package login

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/internal/contract"
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
	if strings.Contains(string(bodyText), "Invalid Credentials") {
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
	c, r, _ := HomePageE(vtopTokens)
	return c, r
}

func HomePageE(vtopTokens types.Cookies) (types.Cookies, string, error) {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/init/page", nil)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return vtopTokens, "", contract.From(err)
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
		return vtopTokens, "", contract.From(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if debug.Debug {
			helpers.Printf("VTOP home page request failed with status %s\n", resp.Status)
		}
		return vtopTokens, "", helpers.HTTPError(resp.StatusCode)
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			helpers.Println(err)
		}
		return vtopTokens, "", contract.Wrap("network_error", "Unable to read the VTOP response.", err)
	}

	if strings.Contains(string(bodyText), "Session Timed Out") || strings.Contains(string(bodyText), `name="username"`) || strings.Contains(string(bodyText), `name='username'`) {
		if debug.Debug {
			helpers.Println("Session Timed Out, login failed. Retrying...")
		}
		return vtopTokens, "", contract.New("auth_required", "VTOP session is unavailable; run cli-top login.")
	}

	vtopTokens.CSRF = helpers.ExtractCSRF2(string(bodyText))

	RegNo, err := helpers.ExtractRegNo(string(bodyText))
	if err != nil && debug.Debug {
		helpers.Println("Error:", err)
	}

	if err != nil {
		return vtopTokens, "", contract.Wrap("parse_error", "VTOP did not return a student identity.", err)
	}
	return vtopTokens, RegNo, nil
}

// LoginE preserves the failure classification for command callers.
func LoginE(username, password string) (types.Cookies, error) {
	for attempt := 0; attempt < 3; attempt++ {
		tokens, captcha, err := getLoginPage()
		if err != nil {
			return types.Cookies{}, contract.From(err)
		}
		logged, kind := performLogin(types.LogIn{Username: username, Password: password}, tokens, captcha)
		switch kind {
		case "":
			tokens.JSESSIONID = logged.JSESSIONID
			return tokens, nil
		case "invalid_captcha":
			continue
		case "network_error":
			e := contract.New("network_error", "Unable to complete VTOP authentication.")
			e.Retryable = true
			return types.Cookies{}, e
		case "max_attempts":
			return types.Cookies{}, contract.New("auth_failed", "VTOP has locked further login attempts; recover your account on VTOP.")
		default:
			return types.Cookies{}, contract.New("auth_failed", "VTOP rejected the credentials.")
		}
	}
	return types.Cookies{}, contract.New("human_action_required", "VTOP authentication requires human attention; log in interactively.")
}
