package features

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"cli-top/types"
)

func fetchReq(regNo string, cookies types.Cookies, target_url string, semID string) ([]byte, error) {
	client := &http.Client{}

	//variables that need to be set with every request

	var data = strings.NewReader(fmt.Sprintf("------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"authorizedID\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"semesterSubId\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J\r\nContent-Disposition: form-data; name=\"_csrf\"\r\n\r\n%s\r\n------WebKitFormBoundary9yjNZXu7BBjgQK7J--\r\n", regNo, semID, cookies.CSRF))
	req, err := http.NewRequest("POST", target_url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.6")
	req.Header.Set("content-type", "multipart/form-data; boundary=----WebKitFormBoundary9yjNZXu7BBjgQK7J")
	req.Header.Set("cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", cookies.JSESSIONID, cookies.SERVERID))
	req.Header.Set("origin", "https://vtop.vit.ac.in")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/content?")
	req.Header.Set("sec-ch-ua", `"Brave";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-platform", `"Android"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-gpc", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyText, nil
}
