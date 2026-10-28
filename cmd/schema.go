package cmd

import (
	"cli-top/internal/contract"
	"cli-top/types"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"reflect"
	"sort"
	"strings"
)

type flagSchema struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Default     string   `json:"default"`
	Description string   `json:"description"`
	Conflicts   []string `json:"conflicts,omitempty"`
	Allowed     []string `json:"enum,omitempty"`
}
type commandSchema struct {
	Command                 string              `json:"command"`
	Aliases                 []string            `json:"aliases"`
	Description             string              `json:"description"`
	AuthRequired            bool                `json:"auth_required"`
	Effect                  string              `json:"effect"`
	Flags                   []flagSchema        `json:"flags"`
	Result                  any                 `json:"result_schema"`
	Selection               string              `json:"selection"`
	Defaults                map[string]string   `json:"defaults"`
	ConditionalRequirements map[string][]string `json:"conditional_requirements"`
	OutputProtocol          string              `json:"output_protocol"`
}

func resultType(name string) any {
	switch name {
	case "profile":
		return types.ProfileResult{}
	case "marks":
		return types.MarksResult{}
	case "grades":
		return types.GradesResult{}
	case "attendance":
		return types.AttendanceResult{}
	case "exams":
		return types.ExamsResult{}
	case "cgpa":
		return types.CGPAResult{}
	case "hostel":
		return types.HostelResult{}
	case "library-dues":
		return types.LibraryResult{}
	case "receipts":
		return types.ReceiptsResult{}
	case "msg":
		return types.MessagesResult{}
	case "events":
		return types.EventsResult{}
	case "timetable":
		return types.TimetableResult{}
	case "calendar", "holiday":
		return types.CalendarResult{}
	case "today", "tomorrow", "dayafter":
		return types.PlanningResult{}
	case "course-page", "course-page-archive":
		return types.MaterialsResult{}
	case "course-allocation":
		return types.AllocationResult{}
	case "da":
		return types.AssignmentsResult{}
	case "leave":
		return types.LeaveResult{}
	case "nightslip":
		return types.NightslipResult{}
	case "facility":
		return types.FacilityResult{}
	case "semesters", "class-groups", "courses", "faculties":
		return types.OptionsResult{}
	default:
		return types.OperationResult{}
	}
}
func jsonSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{jsonSchema(t.Elem()), map[string]any{"type": "null"}}}
	}
	switch t.Kind() {
	case reflect.Struct:
		properties := map[string]any{}
		required := []string{}
		for n := 0; n < t.NumField(); n++ {
			f := t.Field(n)
			tag := f.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			if name == "-" || f.PkgPath != "" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			properties[name] = jsonSchema(f.Type)
			if !strings.Contains(tag, ",omitempty") {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": jsonSchema(t.Elem())}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64, reflect.Int32:
		return map[string]any{"type": "integer"}
	case reflect.Float64:
		return map[string]any{"type": "number"}
	default:
		return map[string]any{}
	}
}
func describe(c *cobra.Command) commandSchema {
	c.InitDefaultHelpFlag()
	c.InheritedFlags()
	effect := "read"
	switch c.Name() {
	case "leave", "nightslip":
		effect = "read; submits with --apply"
	case "events":
		effect = "read; registers with --event-id and --confirm"
	case "da":
		effect = "read; downloads a question paper with an assignment selector"
	case "facility":
		effect = "read; registers with a selector and --confirm"
	case "syllabus":
		effect = "downloads files"
	case "course-page", "course-page-archive":
		effect = "read; downloads files with --materials"
	case "login", "logout":
		effect = "writes local credentials"
	}
	envelope := jsonSchema(reflect.TypeOf(contract.Result{}))
	props := envelope["properties"].(map[string]any)
	props["data"] = map[string]any{"anyOf": []any{jsonSchema(reflect.TypeOf(resultType(c.Name()))), map[string]any{"type": "null"}}}
	envelope["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	props["schema_version"] = map[string]any{"const": contract.Version}
	props["command"] = map[string]any{"const": c.Name()}
	defaults := map[string]string{}
	switch c.Name() {
	case "attendance", "exams":
		defaults["semester"] = "recent semester with data; a failed candidate makes the result incomplete"
	case "marks", "grades", "calendar", "class-groups":
		defaults["semester"] = "select; automatically resolved if only one option exists"
	case "timetable":
		defaults["semester"] = "latest; select during June, July and December"
	case "holiday", "today", "tomorrow", "dayafter", "da":
		defaults["semester"] = "latest"
	case "course-page", "courses", "faculties":
		defaults["semester"] = "all consolidated course options unless a local semester index is supplied"
	}
	requirements := map[string][]string{}
	if c.Name() == "leave" {
		requirements["--apply"] = []string{"--leave-code", "--visiting-place", "--reason", "--from-date", "--from-time", "--to-date", "--to-time"}
	}
	if c.Name() == "nightslip" {
		requirements["--apply"] = []string{"--details", "--from-date", "--from-time", "--to-date", "--to-time"}
	}
	if has(c.Name(), "facility", "events") {
		flag := "--facility-id"
		if c.Name() == "events" {
			flag = "--event-id"
		}
		requirements["registration"] = []string{flag, "--confirm"}
	}
	d := commandSchema{Defaults: defaults, ConditionalRequirements: requirements, OutputProtocol: "versioned result envelope with --output json; terminal tables otherwise", Command: c.Name(), Aliases: append([]string{}, c.Aliases...), Description: c.Short, AuthRequired: !has(c.Name(), "login", "logout", "schema", "completion", "cli-top"), Effect: effect, Flags: []flagSchema{}, Result: envelope, Selection: "Exact IDs are opaque and scoped by source, semester and parent selections. Indexes are 1-based and may change. Missing or ambiguous selections return error.choices; JSON never prompts."}
	flags := pflag.NewFlagSet("schema", pflag.ContinueOnError)
	flags.AddFlagSet(c.Flags())
	flags.AddFlagSet(c.InheritedFlags())
	if has(c.Name(), "schema", "completion") {
		d.OutputProtocol = "raw JSON for schema; raw shell script for completion"
		if c.Name() == "completion" {
			d.Result = map[string]any{"type": "string"}
		} else {
			d.Result = map[string]any{"type": "object"}
		}
	}
	flags.VisitAll(func(f *pflag.Flag) {
		v := flagSchema{Name: f.Name, Type: f.Value.Type(), Default: f.DefValue, Description: f.Usage}
		if strings.HasSuffix(f.Name, "-id") {
			base := strings.TrimSuffix(f.Name, "-id")
			if flags.Lookup(base) != nil {
				v.Conflicts = []string{"--" + base}
			}
		}
		if flags.Lookup(f.Name+"-id") != nil {
			v.Conflicts = []string{"--" + f.Name + "-id"}
		}
		switch f.Name {
		case "source":
			v.Allowed = []string{"consolidated", "archive", "syllabus", "allocation", "assignments"}
			if c.Name() == "faculties" {
				v.Allowed = []string{"consolidated", "archive"}
			}
		case "output":
			v.Allowed = []string{"table", "json"}
		}
		d.Flags = append(d.Flags, v)
	})
	return d
}
func (i *invocation) schemaCommand(root *cobra.Command) *cobra.Command {
	c := &cobra.Command{Use: "schema [command]", Short: "Inspect command flags and result schemas offline", Args: cobra.MaximumNArgs(1)}
	c.RunE = func(c *cobra.Command, args []string) error {
		var data any
		if len(args) == 0 {
			names := []string{}
			for _, cmd := range root.Commands() {
				if !cmd.Hidden && cmd.Name() != "help" {
					names = append(names, cmd.Name())
				}
			}
			sort.Strings(names)
			data = map[string]any{"schema_version": contract.Version, "commands": names, "next": []contract.Action{{Command: "cli-top", Args: []string{"schema", "marks"}}}, "exit_codes": map[int]string{0: "success", 1: "internal or I/O error", 2: "invalid arguments or selection", 3: "authentication required or failed", 4: "selection, confirmation or human action required", 5: "network, upstream or parse error", 6: "policy blocked or client decommissioned", 7: "partial failure", 130: "canceled"}}
		} else {
			target, rest, err := root.Find(args)
			if err != nil || len(rest) > 0 || target == root || target.Hidden {
				return contract.New("invalid_arguments", "Unknown schema command.")
			}
			data = describe(target)
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(data)
	}
	return c
}
