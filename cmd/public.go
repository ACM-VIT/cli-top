package cmd

import (
	"cli-top/debug"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/types"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sync"
)

// Dependencies allows callers to exercise the real command tree with a fixture session.
type Dependencies struct {
	Authenticate      func() (features.Session, error)
	CheckPolicy       func() int
	ProxyAuthenticate func(string, string) (features.Session, error)
}

var invocationMu sync.Mutex // Legacy helpers still have process-wide presentation/session state.

type invocation struct {
	output         string
	nonInteractive bool
	debug          bool
	version        bool
	update         bool
	reply          *contract.Result
	proxyWritten   bool
	executing      bool
	deps           Dependencies
}

var commandDescriptions = map[string]string{
	"profile": "Show VTOP student profile", "marks": "Show marks for a semester", "grades": "Show grades for a semester", "attendance": "Show attendance", "exams": "Show upcoming exams", "cgpa": "Show CGPA", "hostel": "Show hostel details", "receipts": "Show receipts", "library-dues": "Show library dues", "msg": "Show class messages", "events": "Show upcoming and registered events", "timetable": "Show a semester timetable", "calendar": "Show academic calendar", "holiday": "Show upcoming class-impacting holidays", "today": "Show today's classes and exams", "tomorrow": "Show tomorrow's classes and exams", "dayafter": "Show classes and exams two days ahead", "course-page": "List or download consolidated course materials", "course-page-archive": "List or download archived course materials", "course-allocation": "Show allocated course sections", "syllabus": "Download a course syllabus", "da": "Show digital assignments", "leave": "Show leave requests or apply with --apply", "nightslip": "Show nightslip requests or apply with --apply", "facility": "Show facilities or register with --confirm", "semesters": "List academic semester IDs", "class-groups": "List class group IDs for a semester", "courses": "List course IDs within a source", "faculties": "List faculty IDs within a course",
}

func has(name string, names ...string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
func addFeatureFlags(c *cobra.Command) {
	n := c.Name()
	f := c.Flags()
	if has(n, "marks", "grades", "attendance", "exams", "timetable", "calendar", "holiday", "today", "tomorrow", "dayafter", "class-groups", "course-page", "course-page-archive", "courses", "faculties", "da") {
		f.IntP("semester", "s", 0, "Semester index (1-based)")
		if n != "course-page" {
			f.String("semester-id", "", "Exact academic semester ID (see semesters)")
		}
	}
	if has(n, "calendar", "holiday", "today", "tomorrow", "dayafter", "timetable") {
		f.IntP("class-group", "g", 0, "Class group index (1-based)")
		f.String("class-group-id", "", "Exact class group ID (see class-groups)")
	}
	if has(n, "course-page", "course-page-archive", "faculties", "course-allocation", "syllabus", "da") {
		if has(n, "course-page", "course-page-archive", "faculties") {
			f.IntP("course", "c", 0, "Course index (1-based)")
		} else {
			f.StringP("course", "c", "", "Course label query or index")
		}
		f.String("course-id", "", "Exact portal course ID scoped by source and semester")
	}
	if has(n, "course-page", "course-page-archive") {
		f.StringP("faculty", "f", "", "Faculty label query or index")
		f.String("faculty-id", "", "Exact faculty ID scoped by course")
		f.String("materials", "", "Download material indexes/ranges (e.g. 1,2-4; 0 means all)")
	}
	if has(n, "courses", "faculties") {
		f.String("source", "consolidated", "Course source: consolidated, archive, syllabus, allocation, assignments")
	}
	if has(n, "syllabus", "course-allocation", "courses") {
		f.String("category", "", "Curriculum category label query or index")
		f.String("category-id", "", "Exact curriculum category ID")
	}
	if n == "da" {
		f.StringP("assignment", "a", "", "Assignment label query or index")
		f.String("assignment-id", "", "Exact assignment ID when exposed by VTOP")
	}
	if n == "events" {
		f.String("event-id", "", "Exact event ID to register for")
		f.Bool("confirm", false, "Confirm registration for the selected event")
	}
	if n == "facility" {
		f.StringP("facility", "f", "", "Facility label query or index")
		f.String("facility-id", "", "Exact facility ID")
		f.Bool("confirm", false, "Confirm registration for the selected facility")
	}
	if has(n, "leave", "nightslip") {
		f.Bool("apply", false, "Submit a new request using the supplied application fields")
		for _, key := range []string{"from-date", "from-time", "to-date", "to-time"} {
			f.String(key, "", "Application "+key)
		}
		keys := []string{"leave-code", "visiting-place", "reason"}
		if n == "nightslip" {
			keys = []string{"cost-centre-id", "applied-to", "room-type-id", "building-id", "venue", "event-id", "details"}
		}
		for _, key := range keys {
			f.String(key, "", "Application "+key)
		}
	}
}
func featureOptions(c *cobra.Command) (features.Options, error) {
	f := c.Flags()
	str := func(k string) string { v, _ := f.GetString(k); return v }
	num := func(k string) int { v, _ := f.GetInt(k); return v }
	boolean := func(k string) bool { v, _ := f.GetBool(k); return v }
	o := features.Options{Semester: num("semester"), SemesterID: str("semester-id"), ClassGroup: num("class-group"), ClassGroupID: str("class-group-id"), Course: num("course"), CourseID: str("course-id"), CourseQuery: str("course"), FacultyID: str("faculty-id"), FacultyQuery: str("faculty"), CategoryID: str("category-id"), CategoryQuery: str("category"), AssignmentID: str("assignment-id"), AssignmentQuery: str("assignment"), EventID: str("event-id"), FacilityID: str("facility-id"), FacilityQuery: str("facility"), Source: str("source"), Materials: str("materials"), Confirm: boolean("confirm"), Apply: boolean("apply"), Values: map[string]string{}}
	for _, key := range []string{"leave-code", "visiting-place", "reason", "from-date", "from-time", "to-date", "to-time", "cost-centre-id", "applied-to", "room-type-id", "building-id", "venue", "event-id", "details"} {
		if f.Lookup(key) != nil && has(c.Name(), "leave", "nightslip") {
			o.Values[key] = str(key)
			if f.Changed(key) && !o.Apply {
				return o, contract.New("invalid_arguments", "Application fields require --apply.")
			}
		}
	}
	for _, pair := range [][2]string{{"semester-id", "semester"}, {"class-group-id", "class-group"}, {"course-id", "course"}, {"faculty-id", "faculty"}, {"category-id", "category"}, {"facility-id", "facility"}, {"assignment-id", "assignment"}} {
		if f.Changed(pair[0]) && f.Changed(pair[1]) {
			return o, contract.New("invalid_arguments", "Use either --"+pair[0]+" or --"+pair[1]+".")
		}
	}
	for _, key := range []string{"semester", "class-group", "course"} {
		if flag := f.Lookup(key); flag != nil && flag.Value.Type() == "int" && f.Changed(key) && num(key) <= 0 {
			return o, contract.New("invalid_arguments", "--"+key+" must be a positive index.")
		}
	}
	if o.Source != "" && !has(o.Source, "consolidated", "archive", "syllabus", "allocation", "assignments") {
		return o, contract.New("invalid_arguments", "Unknown course --source.")
	}
	if c.Name() == "faculties" && !has(o.Source, "consolidated", "archive") {
		return o, contract.New("invalid_arguments", "faculties supports --source consolidated or archive.")
	}
	if o.Confirm && ((c.Name() == "events" && o.EventID == "") || (c.Name() == "facility" && o.FacilityID == "" && o.FacilityQuery == "")) {
		e := contract.New("invalid_arguments", "A registration selector is required with --confirm.")
		flag := "--event-id"
		if c.Name() == "facility" {
			flag = "--facility-id"
		}
		e.RequiredFlags = []string{flag}
		return o, e
	}
	if err := features.ValidateOptions(c.Name(), o); err != nil {
		return o, err
	}
	return o, nil
}
func (i *invocation) prepare(c *cobra.Command) error {
	if i.output != "table" && i.output != "json" {
		return contract.New("invalid_arguments", "--output must be table or json.")
	}
	return nil
}
func (i *invocation) policy() error {
	switch i.deps.CheckPolicy() {
	case 2:
		return contract.New("client_decommissioned", "This version of cli-top has been decommissioned.")
	case 3:
		return contract.New("policy_blocked", "Use the VTOP website to continue.")
	}
	return nil
}
func (i *invocation) feature(c *cobra.Command, args []string) error {
	o, err := featureOptions(c)
	if err != nil {
		return err
	}
	if err = i.policy(); err != nil {
		return err
	}
	session, err := i.deps.Authenticate()
	if err != nil {
		return err
	}
	session.CheckPolicy = i.deps.CheckPolicy
	restore := helpers.SetVtopLoginHandler(func() (types.Cookies, string, error) {
		refreshed, e := i.deps.Authenticate()
		return refreshed.Cookies, refreshed.RegNo, e
	})
	defer restore()
	data, ctx, warnings, err := session.Execute(c.Name(), o)
	reply := contract.Reply(c.Name(), data, ctx, warnings, err)
	i.reply = &reply
	return err
}
func (i *invocation) root() *cobra.Command {
	root := &cobra.Command{Use: "cli-top", Short: "A CLI for VTOP", SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs}
	root.SetUsageTemplate(`Usage:
  {{.CommandPath}} [flags]{{if .HasAvailableSubCommands}} <command>{{end}}
{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
{{if .HasAvailableSubCommands}}
Available Subcommands:{{range .Commands}}{{if (and .IsAvailableCommand (not .Hidden))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}
`)
	f := root.PersistentFlags()
	f.StringVar(&i.output, "output", i.output, "Output format: table or json")
	f.BoolVar(&i.nonInteractive, "non-interactive", false, "Never prompt for input")
	f.BoolVarP(&i.debug, "debug", "d", false, "Print debug diagnostics")
	f.BoolVarP(&i.version, "version", "v", false, "Print version")
	f.BoolVarP(&i.update, "update", "u", false, "Check for updates")
	root.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		i.executing = true
		if c.Name() == "schema" {
			i.output = "json"
		}
		if err := i.prepare(c); err != nil {
			return err
		}
		p := helpers.CurrentExecutionPolicy()
		p.Machine = i.output == "json"
		p.NonInteractive = p.NonInteractive || i.nonInteractive || p.Machine
		if p.Machine {
			p.Output = io.Discard
		}
		helpers.SetExecutionPolicy(*p)
		debug.Debug = i.debug && !p.Machine
		return nil
	}
	root.RunE = func(c *cobra.Command, args []string) error {
		if i.version {
			if i.output == "json" {
				r := contract.Reply("cli-top", map[string]string{"version": debug.Version}, contract.Context{}, nil, nil)
				i.reply = &r
			} else {
				fmt.Fprintln(c.OutOrStdout(), "Version:", debug.Version)
			}
			return nil
		}
		if err := i.policy(); err != nil {
			return err
		}
		if i.update {
			if helpers.CanPrompt() {
				helpers.CheckUpdate()
				return nil
			}
			available, latest, err := helpers.CheckUpdateSilently()
			if err != nil {
				return contract.Wrap("upstream_error", "Unable to check for updates.", err)
			}
			if i.output == "json" {
				r := contract.Reply("cli-top", types.UpdateResult{Current: debug.Version, Latest: latest, UpdateAvailable: available}, contract.Context{}, nil, nil)
				i.reply = &r
			} else if available {
				fmt.Fprintf(c.OutOrStdout(), "A new version %s is available (you have %s). Run cli-top --update in a terminal to install.\n", latest, debug.Version)
			} else {
				fmt.Fprintln(c.OutOrStdout(), "You are using the latest stable version of cli-top.")
			}
			return nil
		}
		if i.output == "json" {
			r := contract.Reply("cli-top", map[string]string{"version": debug.Version, "schema_version": contract.Version}, contract.Context{}, nil, nil)
			i.reply = &r
			return nil
		}
		fmt.Fprintln(c.OutOrStdout(), "Welcome to CLI-TOP!\nUse \"cli-top help\" to show available commands.")
		return nil
	}
	for name, description := range commandDescriptions {
		c := &cobra.Command{Use: name, Short: description, Args: cobra.NoArgs, RunE: i.feature}
		if name == "dayafter" {
			c.Aliases = []string{"day-after"}
		}
		if name == "msg" {
			c.Aliases = []string{"class-messages"}
		}
		addFeatureFlags(c)
		root.AddCommand(c)
	}
	loginCmd := &cobra.Command{Use: "login", Short: "Store encrypted VTOP credentials", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		if err := i.policy(); err != nil {
			return err
		}
		u, _ := c.Flags().GetString("username")
		p, _ := c.Flags().GetString("password")
		data, err := storeCredentials(u, p, c.InOrStdin(), helpers.OutputWriter())
		r := contract.Reply("login", data, contract.Context{}, nil, err)
		i.reply = &r
		return err
	}}
	loginCmd.Flags().String("username", "", "VTOP username")
	loginCmd.Flags().String("password", "", "VTOP password")
	root.AddCommand(loginCmd)
	root.AddCommand(&cobra.Command{Use: "logout", Short: "Clear stored credentials and session", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		if err := i.policy(); err != nil {
			return err
		}
		data, err := logout()
		r := contract.Reply("logout", data, contract.Context{}, nil, err)
		i.reply = &r
		return err
	}})
	root.AddCommand(i.schemaCommand(root), i.proxyCommand())
	return root
}

// Run executes one invocation. Only main exits the process.
func Run(args []string, in io.Reader, out, errOut io.Writer) int {
	return RunWithDependencies(args, in, out, errOut, Dependencies{})
}
func RunWithDependencies(args []string, in io.Reader, out, errOut io.Writer, deps Dependencies) int {
	invocationMu.Lock()
	defer invocationMu.Unlock()
	if deps.Authenticate == nil {
		deps.Authenticate = authenticate
	}
	if deps.CheckPolicy == nil {
		deps.CheckPolicy = helpers.CheckKillSwitch
	}
	i := &invocation{output: "table", deps: deps}
	// Preserve structured syntax failures too; no command execution occurs here.
	for n, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "--output=json" || (arg == "--output" && n+1 < len(args) && args[n+1] == "json") {
			i.output = "json"
		}
	}
	proxy := len(args) > 0 && args[0] == "proxy"
	if proxy {
		i.output = "json"
	}
	interactive := false
	if file, ok := in.(*os.File); ok {
		interactive = helpers.IsInteractiveOutput(file)
	}
	restore := helpers.SetExecutionPolicy(helpers.ExecutionPolicy{Input: in, Output: out, ErrorOutput: errOut, Machine: i.output == "json", NonInteractive: !interactive})
	defer restore()
	oldDebug := debug.Debug
	defer func() { debug.Debug = oldDebug }()
	root := i.root()
	if found, _, e := root.Find(args); e == nil && has(found.Name(), "schema", "proxy") {
		i.output = "json"
	}
	root.SetArgs(args)
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errOut)
	executed, err := root.ExecuteC()
	name := "cli-top"
	if executed != nil {
		name = executed.Name()
	}
	if err != nil {
		if _, ok := err.(*contract.Error); !ok {
			if i.executing {
				err = contract.From(err)
			} else {
				err = contract.New("invalid_arguments", "Invalid command arguments. Run cli-top help or cli-top schema <command>.")
			}
		}
	}
	if i.proxyWritten {
		return contract.ExitCode(err)
	}
	if i.reply == nil && err != nil {
		r := contract.Reply(name, nil, contract.Context{}, nil, err)
		i.reply = &r
	}
	if i.reply != nil {
		enrichRecovery(i.reply)
		if i.output == "json" {
			if e := json.NewEncoder(out).Encode(i.reply); e != nil {
				return 1
			}
		} else {
			if i.reply.Data != nil {
				if e := renderData(out, i.reply.Data); e != nil {
					return 1
				}
			}
			if err != nil {
				fmt.Fprintf(errOut, "%s: %s\n", contract.From(err).Kind, contract.From(err).Message)
				renderRecovery(errOut, contract.From(err))
			}
		}
	}
	return contract.ExitCode(err)
}
