package features

import (
	"bytes"
	"fmt"

	// "io"
	"log"
	// "net/http"
	// "net/url"
	"strings"
	// "time"
	"cli-top/types"
	// "github.com/charmbracelet/glamour"
	"cli-top/helpers"

	"github.com/PuerkitoBio/goquery"
	// "golang.org/x/net/html"
)

func PrintHostelInfo(regNo string, cookies types.Cookies, url string) {
	body, err := helpers.FetchReq(regNo, cookies, url, "", "", "POST")
	if err != nil {
		log.Fatal("Error fetching HTML:", err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
		return
	}

	// fmt.Println("+-----------------------------+------------------------------------------------------+")
	fmt.Println("Student Accommodation Info")
	fmt.Println("+-----------------------------+------------------------------------------------------+")

	table := doc.Find("div.table-responsive table.table tbody tr")
	lastFiveRows := table.Slice(-5, table.Length())

	lastFiveRows.Each(func(j int, rowSelection *goquery.Selection) {
		header := rowSelection.Find("td").Eq(0).Text()
		value := rowSelection.Find("td").Eq(1).Text()

		fmt.Printf("| %-27s | %-52s |\n", strings.TrimSpace(header), strings.TrimSpace(value))
	})

	fmt.Println("+-----------------------------+------------------------------------------------------+")
}
