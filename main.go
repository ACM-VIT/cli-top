// main.go

package main

import (
	"fmt"
	"log"
	"net/http"
	"vtop-cli/features"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://vtop.vit.ac.in/vtop/login", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("authority", "vtop.vit.ac.in")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("accept-language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("cache-control", "max-age=0")
	req.Header.Set("cookie", "JSESSIONID=3B90C635C7B9FB58A44F5D0604CEF571; SERVERID=s2")
	req.Header.Set("referer", "https://vtop.vit.ac.in/vtop/open/page")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="117", "Not;A=Brand";v="8", "Chromium";v="117"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	imageSrc, exists := doc.Find("#captchaBlock img").Attr("src")
	if exists {
		fmt.Println(features.SolveCaptcha(imageSrc))
	} else {
		fmt.Println("No captcha ez login  ᕙ(`▿´)ᕗ")
	}
}
