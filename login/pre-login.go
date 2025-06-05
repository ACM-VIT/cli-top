package login

import (
	"cli-top/debug"
	"cli-top/helpers"
	types "cli-top/types"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func getSessionServer() types.Cookies {
	client := helpers.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/", nil)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Sec-Fetch-Site", "none")
	resp, err := client.Do(req)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	vtopCookies := helpers.ExtractCookies(resp)
	vtopCookies.CSRF = helpers.ExtractCSRF(helpers.ExtractBodyText(resp))

	return vtopCookies
}

func getLoginPage() (types.Cookies, string) {

	cookies := getSessionServer()

	client := helpers.GetHTTPClient()
	var data = strings.NewReader(fmt.Sprintf(`_csrf=%s&flag=VTOP`, cookies.CSRF))
	req, err := http.NewRequest("POST", "https://vtop.vit.ac.in/vtop/prelogin/setup", data)
	if err != nil && debug.Debug {
		fmt.Println(err)
	}
	helpers.SetVtopHeaders(req)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Origin", "https://vtop.vit.ac.in")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", "https://vtop.vit.ac.in/vtop/open/page")
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
	// fmt.Printf("%s\n", bodyText)

	stringBody := string(bodyText)
	captchaImage := helpers.ExtractImage(stringBody)
	if captchaImage == "nocaptcha" {
		// Vtop does not always send a captcha image, so try again
		return getLoginPage()
	}
	// fmt.Println("getLoginPage() - Captcha:", captchaImage)

	captcha := helpers.SolveCaptcha(captchaImage)
	if strings.Contains(captcha, "disabled") {
		fmt.Println("Captcha auto-solver has been disabled. \nPlease manually solve the captcha and answer here:")
		fmt.Scanln(&captcha)
	}

	return cookies, captcha
}
