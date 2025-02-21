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

func getSessionServer() types.Cookies {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/", nil)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating new request:", err)
		}
		return types.Cookies{}
	}
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.159 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error in getSessionServer:", err)
		}
		return types.Cookies{}
	}
	if resp == nil {
		if debug.Debug {
			fmt.Println("Received nil response in getSessionServer")
		}
		return types.Cookies{}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error reading response body:", err)
		}
		return types.Cookies{}
	}
	bodyString := string(bodyBytes)

	vtopCookies := helpers.ExtractCookies(resp)
	// Use the buffered response body for CSRF extraction
	vtopCookies.CSRF = helpers.ExtractCSRF(bodyString)
	return vtopCookies
}

func getLoginPage() (types.Cookies, string) {
	cookies := getSessionServer()
	if cookies.CSRF == "" {
		if debug.Debug {
			fmt.Println("Empty CSRF token; attempting fresh session retrieval.")
		}
		cookies = getSessionServer()
	}

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	data := strings.NewReader(fmt.Sprintf(`_csrf=%s&flag=VTOP`, cookies.CSRF))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/prelogin/setup", data)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error creating pre-login request:", err)
		}
		return cookies, ""
	}
	req.Header.Set("Host", "vtop.vit.ac.in")
	req.Header.Set("Content-Length", "52")
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
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/open/page")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))

	resp, err := client.Do(req)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error in getLoginPage:", err)
		}
		return cookies, ""
	}
	if resp == nil {
		if debug.Debug {
			fmt.Println("Nil response in getLoginPage")
		}
		return cookies, ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if debug.Debug {
			fmt.Println("Error reading pre-login response body:", err)
		}
		return cookies, ""
	}
	bodyString := string(bodyBytes)
	captchaImage := helpers.ExtractImage(bodyString)
	if captchaImage == "nocaptcha" {
		return getLoginPage()
	}
	captcha := helpers.SolveCaptcha(captchaImage)
	if strings.Contains(captcha, "disabled") {
		fmt.Println("Captcha auto-solver has been disabled.\nPlease manually solve the captcha and enter the answer:")
		fmt.Scanln(&captcha)
	}

	return cookies, captcha
}
