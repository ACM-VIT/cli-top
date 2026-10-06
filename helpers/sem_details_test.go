package helpers

import (
	"reflect"
	"testing"
	"time"

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

func TestSelectSemesterDefaultsToLatestAndQueuesListHint(t *testing.T) {
	const regNo = "TESTAUTOSELECT"
	semCacheMu.Lock()
	semCache[regNo] = semesterCacheEntry{
		semesters: []types.Semester{
			{SemID: "OLD", SemName: "Fall Semester 2025-26"},
			{SemID: "NEW", SemName: "Winter Semester 2025-26"},
		},
		cachedAt: time.Now(),
	}
	semCacheMu.Unlock()
	t.Cleanup(func() {
		semCacheMu.Lock()
		delete(semCache, regNo)
		semCacheMu.Unlock()
		ListSemesters = false
		semesterAutoSelected = false
	})

	ListSemesters = false
	got, err := SelectSemester(regNo, types.Cookies{}, 0)
	if err != nil {
		t.Fatalf("SelectSemester() error = %v", err)
	}
	if got.SemID != "NEW" {
		t.Fatalf("SelectSemester() = %q, want latest semester %q", got.SemID, "NEW")
	}
	if !semesterAutoSelected {
		t.Fatal("expected auto-selection to queue the --list hint")
	}

	PrintSemesterListHint()
	if semesterAutoSelected {
		t.Fatal("expected PrintSemesterListHint to reset the hint state")
	}

	got, err = SelectSemester(regNo, types.Cookies{}, 1)
	if err != nil {
		t.Fatalf("SelectSemester(1) error = %v", err)
	}
	if got.SemID != "OLD" || semesterAutoSelected {
		t.Fatalf("explicit --semester should select %q without a hint, got %q (hint=%v)", "OLD", got.SemID, semesterAutoSelected)
	}
}

func TestShouldListSemestersRespectsAutoSelectSetting(t *testing.T) {
	t.Cleanup(func() { ListSemesters = false })

	cases := []struct {
		env  string
		list bool
		want bool
	}{
		{env: "", list: false, want: false},
		{env: "true", list: false, want: false},
		{env: "false", list: false, want: true},
		{env: "true", list: true, want: true},
		{env: "garbage", list: false, want: false},
	}
	for _, tc := range cases {
		t.Setenv(autoSelectSemesterEnvKey, tc.env)
		ListSemesters = tc.list
		if got := ShouldListSemesters(); got != tc.want {
			t.Fatalf("ShouldListSemesters() with %s=%q, list=%v = %v, want %v", autoSelectSemesterEnvKey, tc.env, tc.list, got, tc.want)
		}
	}
}
