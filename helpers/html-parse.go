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

	var src string
	doc.Find("img.form-control.img-fluid.bg-light.border-0").Each(func(i int, s *goquery.Selection) {
		src, _ = s.Attr("src")
	})

	return src, nil
}

func Extract(html string) {
	src, err := extractImageSrc(html)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(src)
}
