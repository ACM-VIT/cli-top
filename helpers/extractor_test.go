package helpers

import (
	"cli-top/types"
	"net/http"
	"testing"
)

func TestExtractorsHandleExpectedVTOPMarkup(t *testing.T) {
	if got := ExtractCSRF(`<script>var csrfValue = /*token-123*/'ignored';</script>`); got != "token-123" {
		t.Fatalf("ExtractCSRF() = %q", got)
	}
	if got := ExtractCSRF2(`<script>var csrfValue = "abc123-def";</script>`); got != "abc123-def" {
		t.Fatalf("ExtractCSRF2() = %q", got)
	}
	if got, err := ExtractRegNo(`<script>let id = "22BCE0001";</script>`); err != nil || got != "22BCE0001" {
		t.Fatalf("ExtractRegNo() = %q, %v", got, err)
	}
	if got := ExtractImage(`<div id="captchaBlock"><img src="data:image/jpeg;base64,abc"></div>`); got != "data:image/jpeg;base64,abc" {
		t.Fatalf("ExtractImage() = %q", got)
	}
}

func TestExtractCookiesHandlesNilResponse(t *testing.T) {
	if got := ExtractCookies(nil); got != (types.Cookies{}) {
		t.Fatalf("ExtractCookies(nil) = %#v", got)
	}

	response := &http.Response{Header: http.Header{
		"Set-Cookie": {"SERVERID=server; Path=/", "JSESSIONID=session; Path=/"},
	}}
	got := ExtractCookies(response)
	if got.SERVERID != "server" || got.JSESSIONID != "session" {
		t.Fatalf("ExtractCookies() = %#v", got)
	}
}
