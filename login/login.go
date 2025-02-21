package login

import (
	"cli-top/debug"
	"cli-top/helpers"
	"cli-top/types"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func performLogin(userInfo types.LogIn, cookies types.Cookies, captcha string) types.Cookies {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Prevent automatic redirection
			return http.ErrUseLastResponse
		},
	}
	data := strings.NewReader(fmt.Sprintf(`_csrf=%s&username=%s&password=%s&captchaStr=%s`, cookies.CSRF, userInfo.Username, userInfo.Password, captcha))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/login", data)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating login request:", err)
		}
		return types.Cookies{}
	}
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Content-Length", "96")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Origin", "https://vtop.vit.ac.in")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.159 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error in performLogin:", err)
		}
		return types.Cookies{}
	}
	if resp == nil {
		if debug.Debug {
			fmt.Println("Nil response in performLogin")
		}
		return types.Cookies{}
	}
	defer resp.Body.Close()

	// Check for common error messages from VTOP
	if errorCheck(cookies) {
		return types.Cookies{}
	}

	// Read (and discard) the body before extracting cookies if needed
	_, err = io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println("Error reading login response body:", err)
	}

	tokens := helpers.ExtractCookies(resp)
	return tokens
}

func errorCheck(cookies types.Cookies) bool {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/login/error", nil)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating error check request:", err)
		}
		return false
	}
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.159 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error in errorCheck:", err)
		}
		return false
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil && debug.Debug {
		fmt.Println("Error reading error response body:", err)
	}
	bodyString := string(bodyBytes)
	if strings.Contains(bodyString, "Invalid Captcha") {
		fmt.Println("\nInvalid Captcha. The captcha solver can sometimes confuse characters; please retry...")
		return true
	}
	if strings.Contains(bodyString, "Invalid LoginId/Password") ||
		strings.Contains(bodyString, "Invalid Username/Password") {
		fmt.Println("\nInvalid Username/Password. Please check your cli-top configuration and try again...")
		return true
	}
	if strings.Contains(bodyString, "Maximum Fail Attempts") {
		fmt.Println("\nMaximum number of fail attempts reached. Use Forgot Password on VTOP to reset your password.")
		return true
	}
	return false
}

func Login(regNo string, password string) types.Cookies {
	vtopTokens, captcha := getLoginPage()
	if captcha == "" {
		if debug.Debug {
			fmt.Println("Failed to retrieve captcha; restarting login process.")
		}
		vtopTokens, captcha = getLoginPage()
	}
	userInfo := types.LogIn{
		Username: regNo,
		Password: password,
	}
	loginCreds := performLogin(userInfo, vtopTokens, captcha)
	// If login fails (e.g. empty session token), try one more time
	if loginCreds.JSESSIONID == "" {
		if debug.Debug {
			fmt.Println("Login failed; restarting fresh login instance.")
		}
		vtopTokens, captcha = getLoginPage()
		loginCreds = performLogin(userInfo, vtopTokens, captcha)
	}
	vtopTokens.JSESSIONID = loginCreds.JSESSIONID
	return vtopTokens
}

func HomePage(vtopTokens types.Cookies) (types.Cookies, string) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/init/page", nil)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating HomePage request:", err)
		}
		return vtopTokens, ""
	}
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.71 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/login")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", vtopTokens.JSESSIONID, vtopTokens.SERVERID))

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error in HomePage:", err)
		}
		return vtopTokens, ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error reading HomePage response body:", err)
		}
		return vtopTokens, ""
	}
	bodyString := string(bodyBytes)
	if strings.Contains(bodyString, "Session Timed Out") {
		if debug.Debug {
			fmt.Println("Session timed out, login failed. Retrying fresh login instance...")
		}
		return vtopTokens, ""
	}
	// Extract new CSRF token from the buffered body string
	vtopTokens.CSRF = helpers.ExtractCSRF2(bodyString)
	RegNo, err := helpers.ExtractRegNo(bodyString)
	if err != nil && debug.Debug {
		fmt.Println("Error extracting RegNo:", err)
	}
	if debug.Debug {
		fmt.Println("(Helper - ExtractRegNo):", RegNo)
	}
	return vtopTokens, RegNo
}
