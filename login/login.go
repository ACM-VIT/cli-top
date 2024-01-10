package login

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"vtop-cli/helpers"
	types "vtop-cli/types"
)

func performLogin(userInfo types.LogIn, cookies types.Cookies, captcha string) types.Cookies {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Returning an error prevents automatic redirection
			return http.ErrUseLastResponse
		}}
	var data = strings.NewReader(fmt.Sprintf(`_csrf=%s&username=%s&password=%s&captchaStr=%s`, cookies.CSRF, userInfo.Username, userInfo.Password, captcha))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/login", data)
	if err != nil {
		log.Fatal(err)
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
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("From login page, after submitting captcha:\n%s\n", bodyText)

	if strings.Contains(string(bodyText), "Invalid Captcha") {
		fmt.Println("Invalid Captcha. The captcha solver can sometimes confuse between B and 8, please retry...")
		return types.Cookies{}
	}

	tokens := helpers.ExtractCookies(resp)

	return tokens
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
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/init/page", nil)
	if err != nil {
		log.Fatal(err)
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
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", vtopTokens.JSESSIONID, vtopTokens.SERVERID))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText := helpers.ExtractBodyText(resp)
	if strings.Contains(string(bodyText), "Session Timed Out") {
		fmt.Println("Session Timed Out, login failed.")
		return vtopTokens, ""
	}

	vtopTokens.CSRF = helpers.ExtractCSRF2(bodyText)
	RegNo, err := helpers.ExtractRegNo(bodyText)
	if err != nil {
		log.Fatal(err)
	}

	return vtopTokens, RegNo
}
