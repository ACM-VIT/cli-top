package helpers

import (
	"cli-top/internal/contract"
	"io"
	"strings"
	"testing"
)

func TestChooseExactScopeAmbiguityAndEOF(t *testing.T) {
	restore := SetExecutionPolicy(ExecutionPolicy{NonInteractive: true, Output: io.Discard})
	defer restore()
	choices := []contract.Choice{{ID: "C1", Label: "Algorithms", Scope: "S1", Index: 1}, {ID: "C2", Label: "Advanced Algorithms", Scope: "S1", Index: 2}}
	for _, test := range []struct {
		id, query string
		index     int
		kind      string
	}{{"Algorithms", "", 0, "invalid_selection"}, {"C1", "alg", 0, "invalid_arguments"}, {"", "alg", 0, "selection_required"}, {"", "0", 0, "invalid_selection"}, {"", "", 3, "invalid_selection"}, {"", "missing", 0, "invalid_selection"}} {
		_, err := Choose("course", choices, test.id, test.query, test.index)
		if err == nil || contract.From(err).Kind != test.kind {
			t.Fatalf("%+v => %v", test, err)
		}
	}
	c, err := Choose("course", choices, "C1", "", 0)
	if err != nil || c.ID != "C1" || c.Scope != "S1" {
		t.Fatalf("exact selection lost: %+v %v", c, err)
	}
	c, err = Choose("course", choices, "", "Algorithms", 0)
	if err != nil || c.ID != "C1" {
		t.Fatalf("exact label selection failed: %+v %v", c, err)
	}
	reset := SetExecutionPolicy(ExecutionPolicy{Input: strings.NewReader(""), Output: io.Discard})
	defer reset()
	_, err = Choose("course", choices, "", "", 0)
	if contract.From(err).Kind != "selection_required" {
		t.Fatalf("EOF => %v", err)
	}
}
