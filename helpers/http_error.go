package helpers

import (
	"cli-top/internal/contract"
	"fmt"
	"net/http"
)

// HTTPError exposes status metadata without leaking response bodies or URLs.
func HTTPError(status int) *contract.Error {
	kind := "upstream_error"
	message := fmt.Sprintf("VTOP returned status %d %s.", status, http.StatusText(status))
	if status == 401 || status == 403 {
		kind = "auth_required"
		message = "VTOP rejected the session; run cli-top login."
	}
	e := contract.New(kind, message)
	e.Status = status
	e.Retryable = status == 429 || status >= 500
	return e
}
