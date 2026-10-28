package cmd

import (
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/types"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
)

func label(s string) string { return strings.ToUpper(strings.ReplaceAll(s, "_", " ")) }
func scalar(v reflect.Value) string {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "—"
		}
		return scalar(v.Elem())
	}
	if v.Kind() == reflect.Bool {
		if v.Bool() {
			return "yes"
		}
		return "no"
	}
	return fmt.Sprint(v.Interface())
}

// renderData presents the same DTO used by JSON, without capturing global stdout.
func renderData(out io.Writer, data any) error {
	if op, ok := data.(types.OperationResult); ok {
		switch op.Status {
		case "credentials_stored":
			_, err := fmt.Fprintln(out, "Username and encrypted password stored in .env file successfully.")
			return err
		case "logged_out":
			_, err := fmt.Fprintln(out, "Logged out successfully.")
			return err
		}
	}
	return renderValue(out, reflect.ValueOf(data), "")
}
func renderValue(out io.Writer, v reflect.Value, title string) error {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if title != "" {
		if _, err := fmt.Fprintln(out, label(title)); err != nil {
			return err
		}
	}
	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			_, err := fmt.Fprintln(out, "No entries.")
			return err
		}
		elem := v.Type().Elem()
		if elem.Kind() != reflect.Struct {
			for n := 0; n < v.Len(); n++ {
				if _, err := fmt.Fprintln(out, scalar(v.Index(n))); err != nil {
					return err
				}
			}
			return nil
		}
		w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		cols := []int{}
		for n := 0; n < elem.NumField(); n++ {
			f := elem.Field(n)
			if f.Type.Kind() != reflect.Slice && f.Type.Kind() != reflect.Struct {
				cols = append(cols, n)
				fmt.Fprint(w, label(strings.Split(f.Tag.Get("json"), ",")[0]), "\t")
			}
		}
		fmt.Fprintln(w)
		for n := 0; n < v.Len(); n++ {
			row := v.Index(n)
			for _, col := range cols {
				fmt.Fprint(w, scalar(row.Field(col)), "\t")
			}
			fmt.Fprintln(w)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		for n := 0; n < v.Len(); n++ {
			row := v.Index(n)
			for col := 0; col < elem.NumField(); col++ {
				if row.Field(col).Kind() == reflect.Slice {
					if err := renderValue(out, row.Field(col), fmt.Sprintf("%s %d", elem.Field(col).Name, n+1)); err != nil {
						return err
					}
				}
			}
		}
		return nil
	case reflect.Struct:
		for n := 0; n < v.NumField(); n++ {
			f := v.Type().Field(n)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			field := v.Field(n)
			if field.Kind() == reflect.Slice || field.Kind() == reflect.Struct {
				if err := renderValue(out, field, name); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(out, "%s: %s\n", label(name), scalar(field)); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		_, err := fmt.Fprintln(out, scalar(v))
		return err
	}
}
func enrichRecovery(r *contract.Result) {
	e := r.Error
	if e == nil {
		return
	}
	if len(e.Next) == 0 {
		if e.Kind == "auth_required" || e.Kind == "auth_failed" {
			e.Next = []contract.Action{{Command: "cli-top", Args: []string{"login"}}}
		} else {
			e.Next = []contract.Action{{Command: "cli-top", Args: []string{"schema", r.Command}}}
		}
	}
	for _, flag := range e.RequiredFlags {
		var args []string
		switch flag {
		case "--semester-id":
			args = []string{"semesters"}
		case "--class-group-id":
			args = []string{"class-groups"}
		case "--course-id":
			args = []string{"courses"}
			if r.Context.Source != "" {
				args = append(args, "--source", r.Context.Source)
			}
		}
		if len(args) > 0 {
			if flag != "--semester-id" && r.Context.Semester != nil {
				args = append(args, "--semester-id", r.Context.Semester.ID)
			}
			args = append(args, "--output", "json")
			e.Next = append(e.Next, contract.Action{Command: "cli-top", Args: args})
		}
	}
}

func renderRecovery(out io.Writer, e *contract.Error) {
	for _, choice := range e.Choices {
		fmt.Fprintf(out, "  %s  %s\n", choice.ID, choice.Label)
	}
	for _, a := range e.Next {
		fmt.Fprintf(out, "Next: %s %s\n", a.Command, strings.Join(a.Args, " "))
	}
}

// legacyTables preserves the proxy's table-shaped presentation fields while
// obtaining every value directly from the shared domain model.
func legacyTables(data any) []helpers.TableSnapshot {
	tables := []helpers.TableSnapshot{}
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Pointer {
			if !v.IsNil() {
				walk(v.Elem())
			}
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		case reflect.Slice:
			if v.Type().Elem().Kind() != reflect.Struct {
				return
			}
			t := helpers.TableSnapshot{Headers: []string{}, Rows: [][]string{}}
			cols := []int{}
			typ := v.Type().Elem()
			for j := 0; j < typ.NumField(); j++ {
				f := typ.Field(j)
				if f.Type.Kind() != reflect.Struct && f.Type.Kind() != reflect.Slice {
					cols = append(cols, j)
					t.Headers = append(t.Headers, label(strings.Split(f.Tag.Get("json"), ",")[0]))
				}
			}
			for i := 0; i < v.Len(); i++ {
				row := []string{}
				for _, j := range cols {
					row = append(row, scalar(v.Index(i).Field(j)))
				}
				t.Rows = append(t.Rows, row)
				walk(v.Index(i))
			}
			if len(cols) > 0 {
				tables = append(tables, t)
			}
		}
	}
	walk(reflect.ValueOf(data))
	return tables
}
