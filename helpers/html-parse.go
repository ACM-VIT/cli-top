package helpers

import (
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func extractImageSrc(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	src := doc.Find("#captchaBlock img").AttrOr("src", "")
	if src == "" {
		fmt.Println("No captcha image found")
		return "", fmt.Errorf("no captcha image found")
	}

	return src, nil
}

func Extract(html string) string {
	src, err := extractImageSrc(html)
	if err != nil {
		log.Fatal(err)
		return ""
	}
	// fmt.Println(src)

	return src
}
