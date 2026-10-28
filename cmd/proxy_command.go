package cmd

import (
	"bytes"
	"cli-top/debug"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/internal/proxyutil"
	"cli-top/login"
	"cli-top/types"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"strings"
	"time"
)

type proxyResponse struct {
	Success       bool              `json:"success"`
	Command       string            `json:"command"`
	Version       string            `json:"version"`
	RequestedAt   string            `json:"requested_at"`
	CompletedAt   string            `json:"completed_at"`
	DurationMs    int64             `json:"duration_ms"`
	RegNo         string            `json:"reg_no,omitempty"`
	Flags         map[string]string `json:"flags,omitempty"`
	Output        string            `json:"output,omitempty"`
	StructuredRaw map[string]any    `json:"structured_data,omitempty"`
	Error         *contract.Error   `json:"error,omitempty"`
}

var defaultSyncCommands = []string{"profile", "timetable", "attendance", "marks", "cgpa", "exams"}

func canonicalCommand(name string) string {
	switch name {
	case "class-messages":
		return "msg"
	case "day-after":
		return "dayafter"
	}
	return name
}
func proxyFlags(args []string) (map[string]string, error) {
	out := map[string]string{}
	for n := 0; n < len(args); n++ {
		arg := args[n]
		if !strings.HasPrefix(arg, "-") {
			return nil, contract.New("invalid_arguments", "Unexpected proxy positional argument.")
		}
		raw := strings.TrimLeft(arg, "-")
		key, value, equals := strings.Cut(raw, "=")
		key = proxyutil.CanonicalFlagName(key)
		if key == "classGroup" {
			key = "class-group"
		}
		if !equals {
			value = "true"
			if n+1 < len(args) && !strings.HasPrefix(args[n+1], "-") {
				n++
				value = args[n]
			}
		}
		if _, exists := out[key]; exists {
			return nil, contract.New("invalid_arguments", "Duplicate proxy flag.")
		}
		out[key] = value
	}
	return out, nil
}
func proxyOptions(command string, flags map[string]string, shared bool) (features.Options, error) {
	c := &cobra.Command{Use: command}
	addFeatureFlags(c)
	c.Flags().Bool("debug", false, "")
	for k, v := range flags {
		if k == "commands" {
			continue
		}
		f := c.Flags().Lookup(k)
		if f == nil {
			if shared {
				continue
			}
			return features.Options{}, contract.New("invalid_arguments", "Unknown flag for proxy command.")
		}
		if err := c.Flags().Set(k, v); err != nil {
			return features.Options{}, contract.New("invalid_arguments", "Invalid proxy flag value.")
		}
	}
	return featureOptions(c)
}
func proxyEntry(command string, data any, ctx contract.Context, warnings []contract.Warning, err error) map[string]any {
	var output bytes.Buffer
	if data != nil {
		_ = renderData(&output, data)
	}
	r := contract.Reply(command, data, ctx, warnings, err)
	enrichRecovery(&r)
	structured := map[string]any{"data": data, "context": ctx, "warnings": r.Warnings, "schema_version": contract.Version}
	// Legacy presentation fields are generated from the DTO, never stdout capture.
	structured["tables"] = legacyTables(data)
	if output.Len() > 0 {
		_, messages := proxyutil.NormalizeOutput(output.String())
		structured["messages"] = messages
	}
	if e := r.Error; e != nil && len(e.Choices) > 0 {
		choices := [][]string{}
		for _, c := range e.Choices {
			choices = append(choices, []string{c.ID, c.Label})
		}
		structured["selection_requests"] = []helpers.ProxySelectionRequest{{Subject: strings.TrimSuffix(strings.TrimPrefix(e.RequiredFlags[0], "--"), "-id"), Message: e.Message, Table: helpers.TableSnapshot{Headers: []string{"ID", "LABEL"}, Rows: choices}}}
	}
	return map[string]any{"command": command, "success": err == nil, "output": strings.TrimSpace(output.String()), "structured_data": structured, "error": r.Error}
}
func (i *invocation) proxyCommand() *cobra.Command {
	return &cobra.Command{Use: "proxy <username> <password> <command> [flags]", Short: "Compatibility entrypoint for proxy services", Hidden: true, DisableFlagParsing: true, RunE: func(c *cobra.Command, args []string) error {
		i.output = "json"
		policy := helpers.CurrentExecutionPolicy()
		policy.Machine = true
		policy.NonInteractive = true
		policy.Output = io.Discard
		helpers.SetExecutionPolicy(*policy)
		debug.Debug = false
		start := time.Now()
		command := ""
		if len(args) > 2 {
			command = canonicalCommand(strings.ToLower(strings.TrimSpace(args[2])))
		}
		resp := proxyResponse{Command: command, Version: debug.Version, RequestedAt: start.UTC().Format(time.RFC3339Nano)}
		finish := func(err error) error {
			resp.Success = err == nil
			resp.Error = contract.From(err)
			resp.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
			resp.DurationMs = time.Since(start).Milliseconds()
			if encodeErr := json.NewEncoder(c.OutOrStdout()).Encode(resp); encodeErr != nil {
				return contract.Wrap("io_error", "Unable to write proxy response.", encodeErr)
			}
			i.proxyWritten = true
			return err
		}
		if len(args) < 3 {
			return finish(contract.New("invalid_arguments", "proxy requires username, password and command."))
		}
		flags, err := proxyFlags(args[3:])
		if err != nil {
			return finish(err)
		}
		resp.Flags = flags
		if command == "facility" && flags["faculty"] != "" {
			flags["facility"] = flags["faculty"]
			delete(flags, "faculty")
		}
		if command != "sync" && flags["commands"] != "" {
			return finish(contract.New("invalid_arguments", "--commands is only valid for sync."))
		}
		commands := []string{command}
		if command == "sync" {
			commands = defaultSyncCommands
			if flags["commands"] != "" {
				commands = strings.Split(flags["commands"], ",")
			}
		}
		options := []features.Options{}
		allowed := map[string]bool{"commands": true, "debug": true}
		for n, name := range commands {
			name = canonicalCommand(strings.TrimSpace(name))
			commands[n] = name
			if _, ok := commandDescriptions[name]; !ok {
				return finish(contract.New("invalid_arguments", "Unsupported proxy command."))
			}
			fc := &cobra.Command{Use: name}
			addFeatureFlags(fc)
			fc.Flags().VisitAll(func(f *pflag.Flag) { allowed[f.Name] = true })
			o, e := proxyOptions(name, flags, command == "sync")
			if e != nil {
				return finish(e)
			}
			if command == "sync" && (o.Apply || o.Confirm || o.Materials != "" || name == "syllabus" || o.AssignmentID != "" || o.AssignmentQuery != "") {
				return finish(contract.New("invalid_arguments", "sync supports read operations only."))
			}
			options = append(options, o)
		}
		for key := range flags {
			if !allowed[key] {
				return finish(contract.New("invalid_arguments", "Unknown proxy flag."))
			}
		}
		if strings.TrimSpace(args[0]) == "" || args[1] == "" {
			return finish(contract.New("auth_failed", "missing VTOP credentials"))
		}
		if err = i.policy(); err != nil {
			return finish(err)
		}
		authenticateProxy := i.deps.ProxyAuthenticate
		if authenticateProxy == nil {
			authenticateProxy = func(username, password string) (features.Session, error) {
				cookies, e := login.LoginE(username, password)
				if e != nil {
					return features.Session{}, e
				}
				cookies, regNo, e := login.HomePageE(cookies)
				return features.Session{Cookies: cookies, RegNo: regNo}, e
			}
		}
		s, err := authenticateProxy(args[0], args[1])
		if err != nil {
			return finish(err)
		}
		resp.RegNo = s.RegNo
		s.CheckPolicy = i.deps.CheckPolicy
		restore := helpers.SetVtopLoginHandler(func() (types.Cookies, string, error) {
			fresh, e := authenticateProxy(args[0], args[1])
			return fresh.Cookies, fresh.RegNo, e
		})
		defer restore()

		results := []any{}
		var finalErr error
		for n, name := range commands {
			data, ctx, warnings, e := s.Execute(name, options[n])
			entry := proxyEntry(name, data, ctx, warnings, e)
			if command == "sync" {
				results = append(results, entry)
				if e != nil {
					finalErr = contract.New("partial_failure", "One or more sync commands failed.")
				}
			} else {
				resp.Output = entry["output"].(string)
				resp.StructuredRaw = entry["structured_data"].(map[string]any)
				finalErr = e
			}
		}
		if command == "sync" {
			resp.StructuredRaw = map[string]any{"results": results}
		}
		return finish(finalErr)
	}}
}
