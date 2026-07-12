package helpers

import (
	"reflect"
	"testing"

	"cli-top/types"
)

func TestFindAndSaveSemIDsPrefersSemesterSelect(t *testing.T) {
	body := []byte(`
		<select id="unrelated"><option value="X">Not a semester</option></select>
		<select name="semesterSubId">
			<option value="">Choose</option>
			<option value="SEM1">Fall Semester 2025-26</option>
			<option value="SEM2"><span>Winter Semester 2025-26</span></option>
		</select>`)

	got, err := FindAndSaveSemIds(body)
	if err != nil {
		t.Fatalf("FindAndSaveSemIds() error = %v", err)
	}
	want := []types.Semester{
		{SemID: "SEM1", SemName: "Fall Semester 2025-26"},
		{SemID: "SEM2", SemName: "Winter Semester 2025-26"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindAndSaveSemIds() = %#v, want %#v", got, want)
	}
}

func TestFindAndSaveSemIDsRejectsMissingOptions(t *testing.T) {
	if _, err := FindAndSaveSemIds([]byte(`<html><select><option value="">Choose</option></select></html>`)); err == nil {
		t.Fatal("FindAndSaveSemIds() expected an error")
	}
}
