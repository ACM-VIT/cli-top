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
	test "vtop-cli/test"
)

func getLoginPage() (types.Cookies, string) {

	cookies := getSessionServer()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf(`_csrf=%s&flag=VTOP`, cookies.CSRF))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/prelogin/setup", data)
	if err != nil {
		log.Fatal(err)
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
	// fmt.Printf("%s\n", bodyText)

	stringBody := string(bodyText)
	captchaImage := helpers.Extract(stringBody)
	fmt.Println("getLoginPage() - Captcha:", captchaImage)
	helpers.DecodeCaptchaFile("captcha.jpg", captchaImage)
	// outfile := convertToGray("captcha.jpg")
	// captcha := test.CaptchaParse(outfile)

	captcha := test.CaptchaSolver()
	fmt.Println("getCaptcha() - Captcha:", captcha)

	return cookies, captcha
}

func performLogin(userInfo types.LogIn, cookies types.Cookies, captcha string) string {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
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
	// bodyText, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%s\n", bodyText)

	return ""
}
