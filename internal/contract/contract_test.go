package contract

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestErrorCauseSanitizationAndCancellation(t *testing.T) {
	secret := errors.New("secret-cookie-internal-diagnostic")
	err := Wrap("parse_error", "Unable to parse attendance.", secret)
	if !errors.Is(err, secret) || ExitCode(err) != 5 {
		t.Fatal("cause or exit lost")
	}
	raw, e := json.Marshal(Reply("attendance", nil, Context{}, nil, err))
	if e != nil || strings.Contains(string(raw), "secret-cookie") {
		t.Fatalf("unsafe result: %s %v", raw, e)
	}
	if From(secret).Kind != "internal_error" || From(secret).Message == secret.Error() {
		t.Fatal("unknown error exposed")
	}
	if ExitCode(context.Canceled) != 130 {
		t.Fatal("cancellation classified incorrectly")
	}
}
