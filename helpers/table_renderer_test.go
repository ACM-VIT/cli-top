package helpers

import (
	"reflect"
	"testing"
)

func TestPrintTableDoesNotMutateInput(t *testing.T) {
	t.Setenv("CLI_TOP_PROXY_MODE", "1")
	table := [][]string{{"Name", "Status"}, {"Example", "Ready"}}
	original := cloneTableData(table)

	var captured TableSnapshot
	restore := RegisterTableCaptureHook(func(snapshot TableSnapshot) { captured = snapshot })
	defer restore()

	PrintTable(table, 1)

	if !reflect.DeepEqual(table, original) {
		t.Fatalf("PrintTable mutated input: got %#v want %#v", table, original)
	}
	if !reflect.DeepEqual(captured.Headers, []string{"INDEX", "NAME", "STATUS"}) {
		t.Fatalf("captured headers = %#v", captured.Headers)
	}
}
