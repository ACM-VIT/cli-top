package login

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func performLogin(userInfo types.LogIn, cookies types.Cookies, captcha string) types.Cookies {

	client := helpers.GetHTTPClient()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() { client.CheckRedirect = nil }()
	var data = strings.NewReader(fmt.Sprintf(`_csrf=%s&username=%s&password=%s&captchaStr=%s`, cookies.CSRF, userInfo.Username, userInfo.Password, captcha))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/login", data)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Origin", "https://vtop.vit.ac.in")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	if errorCheck(cookies) {
		return types.Cookies{}
	}

	tokens := helpers.ExtractCookies(resp)

	return tokens
}

func errorCheck(cookies types.Cookies) bool {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/login/error", nil)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}

	if strings.Contains(string(bodyText), "Invalid Captcha") {
		fmt.Println("\nInvalid Captcha. The captcha solver can sometimes confuse between B and 8, please retry...")
		return true
	}
	if strings.Contains(string(bodyText), "Invalid LoginId/Password") {
		fmt.Println("\nInvalid LoginId/Password. Please check your cli-top config and try again...")
		return true
	}

	if strings.Contains(string(bodyText), "Invalid Username/Password") {
		fmt.Println("\nInvalid Username/Password. Please check your cli-top config and try again...")
		return true
	}

	if strings.Contains(string(bodyText), "Maximum Fail Attempts") {
		fmt.Println("\nNumber Of Maximum Fail Attempts Reached. Use Forgot Password on VTOP to reset your password.")
		return true
	}

	return false
}

func Login(regNo string, password string) types.Cookies {
	vtopTokens, captcha := getLoginPage()
	// fmt.Println(captcha)

	userInfo := types.LogIn{
		Username: regNo,
		Password: password,
	}

	loginCreds := performLogin(userInfo, vtopTokens, captcha)
	// fmt.Println(loginCreds)
	vtopTokens.JSESSIONID = loginCreds.JSESSIONID

	return vtopTokens
}

func HomePage(vtopTokens types.Cookies) (types.Cookies, string) {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/init/page", nil)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", vtopTokens.JSESSIONID, vtopTokens.SERVERID))
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	bodyText := helpers.ExtractBodyText(resp)

	if strings.Contains(string(bodyText), "Session Timed Out") {
		if debug.Debug {
			fmt.Println("Session Timed Out, login failed. Retrying...")
		}
		return vtopTokens, ""
	}

	vtopTokens.CSRF = helpers.ExtractCSRF2(bodyText)

	RegNo, err := helpers.ExtractRegNo(bodyText)
	if err != nil && debug.Debug {
		// Handle the error
		fmt.Println("Error:", err)
		// You might want to return or log the error, or take other appropriate actions
	}

	if debug.Debug {
		fmt.Println("(Helper - ExtractRegNo):", RegNo)
	}

	return vtopTokens, RegNo
}

func AutoRelogin(regNo string) (types.Cookies, bool) {
	username := os.Getenv("VTOP_USERNAME")
	password := os.Getenv("PASSWORD")
	if username == "" || password == "" {
		if debug.Debug {
			fmt.Println("No stored credentials found for auto-relogin.")
		}
		return types.Cookies{}, false
	}
	if debug.Debug {
		fmt.Println("Attempting auto-relogin with stored credentials...")
	}
	newCookies := Login(username, password)
	if helpers.ValidateCookies(newCookies) {
		if debug.Debug {
			fmt.Println("Auto-relogin successful.")
		}
		return newCookies, true
	}
	if debug.Debug {
		fmt.Println("Auto-relogin failed.")
	}
	return types.Cookies{}, false
}
