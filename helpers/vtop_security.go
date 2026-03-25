package helpers

import (
	"fmt"
	"net/url"
	"strings"
)

const VtopBaseURL = "https://vtop.vit.ac.in"
const vtopHost = "vtop.vit.ac.in"
const vtopDefaultReferer = VtopBaseURL + "/vtop/content"

func isVtopHost(host string) bool {
	return strings.EqualFold(host, vtopHost)
}

func ValidateVtopURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty VTOP URL")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid VTOP URL: %w", err)
	}
	if parsed.Scheme != "https" || !isVtopHost(parsed.Hostname()) {
		return fmt.Errorf("refusing to send non-VTOP request to %s", rawURL)
	}
	return nil
}
