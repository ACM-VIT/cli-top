package login

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type cookies struct {
	server     string
	_csrf      string
	jsessionID string
}

func getSessionServer() map[string]string {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/", nil)
	if err != nil {
		log.Fatal(err)
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
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=0, i")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	secrets := make(map[string]string)
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		// fmt.Println(cookie.Name, cookie.Value)
		secrets[cookie.Name] = cookie.Value
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("%s\n", bodyText)

	// Convert bodyText to a string
	bodyString := string(bodyText)

	// Define a regular expression that matches lines containing "csrfValue"
	re := regexp.MustCompile(`.*csrfValue.*`)

	// Find the matches
	lines := re.FindAllString(bodyString, -1)

	// Iterate over the lines
	for _, line := range lines {
		// fmt.Println("Found line:", line)

		// Define a regular expression that matches the pattern of the variable assignment
		re := regexp.MustCompile(`var csrfValue = /\*(.*?)\*/'.*';`)

		// Find the match
		match := re.FindStringSubmatch(line)

		// If a match was found, print the value of the variable
		if len(match) > 1 {
			secrets["_csrf"] = match[1]
			break
		}

	}

	fmt.Println(secrets)

	return secrets

}

func getLoginPage() {

	secrets := getSessionServer()
	vtopCookies := cookies{
		server:     secrets["SERVERID"],
		_csrf:      secrets["_csrf"],
		jsessionID: secrets["JSESSIONID"],
	}

	fmt.Println("VTOP Cookies:", vtopCookies)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf(`_csrf=%s&flag=VTOP`, vtopCookies._csrf))
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
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s; SERVERID=%s", vtopCookies.jsessionID, vtopCookies.server))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", bodyText)
}

func main() {
	getLoginPage()
}
