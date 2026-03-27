package tests

import (
	"cli-top/helpers"
	"cli-top/internal/proxyutil"
	"reflect"
	"testing"
)

func TestParseProxyFlagsCanonicalizesAliases(t *testing.T) {
	flags := proxyutil.ParseFlags([]string{
		"--curriculum-category", "score",
		"--course", "34",
		"--materials", "1,2-4",
		"--assignment", "cat1",
		"--facility", "badminton",
		"--confirm",
	})

	expected := map[string]string{
		"category":   "score",
		"course":     "34",
		"materials":  "1,2-4",
		"assignment": "cat1",
		"facility":   "badminton",
		"confirm":    "true",
	}

	if !reflect.DeepEqual(flags, expected) {
		t.Fatalf("unexpected flags: got %#v want %#v", flags, expected)
	}
}

func TestBuildProxyStructuredDataIncludesSelectionRequests(t *testing.T) {
	table := helpers.TableSnapshot{
		Headers:  []string{"INDEX", "OPTION"},
		Rows:     [][]string{{"1", "SCORE"}},
		HasIndex: true,
	}
	selectionRequest := helpers.ProxySelectionRequest{
		Subject: "category",
		Message: "selection required for category",
		Table:   table,
	}

	structured := proxyutil.BuildStructuredData(
		[]helpers.TableSnapshot{table},
		[]helpers.ProxySelectionRequest{selectionRequest},
		[]string{"selection required for category"},
	)
	if structured == nil {
		t.Fatal("expected structured data, got nil")
	}

	tables, ok := structured["tables"].([]helpers.TableSnapshot)
	if !ok || len(tables) != 1 {
		t.Fatalf("unexpected tables payload: %#v", structured["tables"])
	}

	requests, ok := structured["selection_requests"].([]helpers.ProxySelectionRequest)
	if !ok || len(requests) != 1 {
		t.Fatalf("unexpected selection_requests payload: %#v", structured["selection_requests"])
	}

	if got := proxyutil.SelectionRequiredMessage(requests); got != selectionRequest.Message {
		t.Fatalf("unexpected selection message: got %q want %q", got, selectionRequest.Message)
	}

	messages, ok := structured["messages"].([]string)
	if !ok || len(messages) != 1 || messages[0] != selectionRequest.Message {
		t.Fatalf("unexpected messages payload: %#v", structured["messages"])
	}
}

func TestNormalizeProxyOutputSplitsMessages(t *testing.T) {
	cleaned, messages := proxyutil.NormalizeOutput("\n  One line  \n\nTwo line\n")
	if cleaned != "One line\nTwo line" {
		t.Fatalf("unexpected cleaned output: %q", cleaned)
	}

	expected := []string{"One line", "Two line"}
	if !reflect.DeepEqual(messages, expected) {
		t.Fatalf("unexpected messages: got %#v want %#v", messages, expected)
	}
}
