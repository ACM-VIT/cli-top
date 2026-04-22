package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"cli-top/debug"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/internal/proxyutil"
	"cli-top/login"
	"cli-top/types"

	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:                "proxy <username> <password> <command> [flags]",
	Short:              "Machine-facing entrypoint used by the proxy + MCP services",
	Hidden:             true,
	Args:               cobra.MinimumNArgs(3),
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		runProxyCommand(args)
	},
}

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
	Error         *proxyError       `json:"error,omitempty"`
}

type proxyError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

var interactiveProxyCommands = map[string]struct{}{
	"timetable": {},
	"holiday":   {},
	"today":     {},
	"tomorrow":  {},
	"dayafter":  {},
	"marks":     {},
	"grades":    {},
	"calendar":  {},
	"events":    {},

	"course-page":         {},
	"course-page-archive": {},
	"course-allocation":   {},
	"da":                  {},
	"facility":            {},
	"syllabus":            {},
}

var defaultSyncCommands = []string{"profile", "timetable", "attendance", "marks", "cgpa", "exams"}

type syncResultEntry struct {
	Command       string         `json:"command"`
	Success       bool           `json:"success"`
	Output        string         `json:"output,omitempty"`
	StructuredRaw map[string]any `json:"structured_data,omitempty"`
	Error         *proxyError    `json:"error,omitempty"`
}

func runProxyCommand(args []string) {
	start := time.Now()
	username := strings.TrimSpace(args[0])
	password := args[1]
	command := strings.ToLower(strings.TrimSpace(args[2]))
	flagArgs := []string{}
	if len(args) > 3 {
		flagArgs = args[3:]
	}
	flags := proxyutil.ParseFlags(flagArgs)

	resp := proxyResponse{
		Command:     command,
		Version:     debug.Version,
		RequestedAt: start.UTC().Format(time.RFC3339Nano),
	}
	if len(flags) > 0 {
		resp.Flags = flags
	}

	if flags != nil {
		if v := strings.ToLower(flags["debug"]); v == "true" || v == "1" {
			debug.Debug = true
		}
	}

	if username == "" || password == "" {
		resp.Error = &proxyError{Kind: "auth_failed", Message: "missing VTOP credentials"}
		resp.Success = false
		resp.print(start)
		return
	}

	cookies, regNo, err := proxyLogin(username, password)
	if err != nil {
		resp.Error = &proxyError{Kind: "auth_failed", Message: err.Error()}
		resp.Success = false
		resp.print(start)
		return
	}
	resp.RegNo = regNo

	prevProxyMode := os.Getenv("CLI_TOP_PROXY_MODE")
	os.Setenv("CLI_TOP_PROXY_MODE", "1")
	defer func() {
		if prevProxyMode == "" {
			os.Unsetenv("CLI_TOP_PROXY_MODE")
		} else {
			os.Setenv("CLI_TOP_PROXY_MODE", prevProxyMode)
		}
	}()

	if command == "sync" {
		runSyncProxyCommand(&resp, flags, cookies, regNo, start)
		return
	}

	prevLogin := helpers.VtopLoginGlobal
	helpers.VtopLoginGlobal = func() (types.Cookies, string) {
		freshCookies, freshReg, loginErr := proxyLogin(username, password)
		if loginErr != nil {
			return types.Cookies{}, ""
		}
		return freshCookies, freshReg
	}
	defer func() {
		helpers.VtopLoginGlobal = prevLogin
	}()

	tableSnapshots := []helpers.TableSnapshot{}
	restore := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tableSnapshots = append(tableSnapshots, snapshot)
	})
	defer restore()

	selectionRequests := []helpers.ProxySelectionRequest{}
	restoreSelection := helpers.RegisterProxySelectionHook(func(request helpers.ProxySelectionRequest) {
		selectionRequests = append(selectionRequests, request)
	})
	defer restoreSelection()

	output, execErr := captureCommandOutput(func() error {
		return dispatchProxyCommand(command, flags, cookies, regNo)
	}, nil)

	cleanedOutput, messages := proxyutil.NormalizeOutput(output)
	if cleanedOutput != "" {
		resp.Output = cleanedOutput
	}

	resp.StructuredRaw = proxyutil.BuildStructuredData(tableSnapshots, selectionRequests, messages)

	if len(selectionRequests) > 0 {
		resp.Error = &proxyError{Kind: "selection_required", Message: proxyutil.SelectionRequiredMessage(selectionRequests)}
	} else if execErr != nil {
		resp.Error = &proxyError{Kind: "execution_error", Message: execErr.Error()}
	}
	resp.Success = resp.Error == nil
	resp.print(start)
}

func (r *proxyResponse) print(start time.Time) {
	if r.CompletedAt == "" {
		r.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if r.Error != nil {
		r.Success = false
	}
	r.DurationMs = time.Since(start).Milliseconds()

	payload, err := json.Marshal(r)
	if err != nil {
		fallback := map[string]any{
			"success": false,
			"command": r.Command,
			"error": map[string]string{
				"kind":    "encoding_error",
				"message": err.Error(),
			},
		}
		encoded, _ := json.Marshal(fallback)
		fmt.Println(string(encoded))
		return
	}

	fmt.Println(string(payload))
}

func captureCommandOutput(run func() error, tee io.Writer) (string, error) {
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w
	var buf bytes.Buffer
	done := make(chan error, 1)

	go func() {
		writer := io.Writer(&buf)
		if tee != nil {
			writer = io.MultiWriter(writer, tee)
		}
		_, copyErr := io.Copy(writer, r)
		r.Close()
		done <- copyErr
	}()

	runErr := run()

	w.Close()
	os.Stdout = originalStdout
	copyErr := <-done
	if runErr == nil && copyErr != nil {
		runErr = copyErr
	}
	return buf.String(), runErr
}

func proxyLogin(username, password string) (types.Cookies, string, error) {
	tokens := login.Login(username, password)
	if !helpers.ValidateCookies(tokens) {
		return types.Cookies{}, "", fmt.Errorf("failed to initialize VTOP session")
	}
	cookies, regNo := login.HomePage(tokens)
	if !helpers.ValidateCookies(cookies) || regNo == "" {
		return types.Cookies{}, "", fmt.Errorf("invalid VTOP credentials")
	}
	return cookies, regNo, nil
}

func dispatchProxyCommand(command string, flags map[string]string, cookies types.Cookies, regNo string) error {
	return executeFeatureCommand(command, flags, cookies, regNo)
}

func executeFeatureCommand(command string, flags map[string]string, cookies types.Cookies, regNo string) error {
	switch command {
	case "profile":
		features.Profile(cookies, regNo)
	case "marks":
		features.GetMarks(regNo, cookies, "", parseIntFlag(flags, "semester"))
	case "grades":
		features.GetGrades(regNo, cookies, "", parseIntFlag(flags, "semester"))
	case "attendance":
		features.GetAttendance(regNo, cookies, parseIntFlag(flags, "semester"))
	case "timetable":
		features.GetTimeTable(regNo, cookies, parseIntFlag(flags, "semester"))
	case "holiday":
		features.GetHolidayList(regNo, cookies, parseIntFlag(flags, "semester"), parseIntFlag(flags, "classGroup"))
	case "today":
		features.GetToday(regNo, cookies, parseIntFlag(flags, "semester"), parseIntFlag(flags, "classGroup"))
	case "tomorrow":
		features.GetTomorrow(regNo, cookies, parseIntFlag(flags, "semester"), parseIntFlag(flags, "classGroup"))
	case "dayafter":
		features.GetDayAfter(regNo, cookies, parseIntFlag(flags, "semester"), parseIntFlag(flags, "classGroup"))
	case "events":
		features.GetEvents(regNo, cookies)
	case "receipts":
		features.GetReceipt(regNo, cookies)
	case "hostel":
		features.PrintHostelInfo(regNo, cookies, "https://vtop.vit.ac.in/vtop/studentsRecord/StudentProfileAllView")
	case "cgpa":
		features.PrintCgpa(regNo, cookies, "https://vtop.vit.ac.in/vtop/examinations/examGradeView/StudentGradeHistory")
	case "exams":
		features.GetExamSchedule(regNo, cookies, parseIntFlag(flags, "semester"))
	case "library-dues":
		features.GetLibraryDues(regNo, cookies)
	case "calendar":
		features.PrintCal(regNo, cookies, parseIntFlag(flags, "semester"), parseIntFlag(flags, "classGroup"))
	case "course-page":
		features.ExecuteCoursePageDownload(
			regNo,
			cookies,
			parseIntFlag(flags, "semester"),
			parseIntFlag(flags, "course"),
			flagsValue(flags, "faculty"),
			parseIntFlag(flags, "fuzzyIndex"),
			flagsValue(flags, "materials"),
		)
	case "course-page-archive":
		features.ExecuteCoursePageOldDownload(
			regNo,
			cookies,
			parseIntFlag(flags, "semester"),
			parseIntFlag(flags, "course"),
			flagsValue(flags, "faculty"),
			parseIntFlag(flags, "fuzzyIndex"),
			flagsValue(flags, "materials"),
		)
	case "course-allocation":
		features.ExecuteInteractiveCourseAllocationView(regNo, cookies, "", flagsValue(flags, "category"), flagsValue(flags, "course"))
	case "nightslip":
		return features.ExecuteNightSlip(regNo, cookies, features.NightSlipApplyInput{
			Apply:           parseBoolFlag(flags, "apply"),
			CostCentreID:    flagsValue(flags, "cost-centre-id"),
			AppliedTo:       flagsValue(flags, "applied-to"),
			RoomTypeID:      flagsValue(flags, "room-type-id"),
			LateHourEventID: flagsValue(flags, "event-id"),
			Details:         flagsValue(flags, "details"),
			FromDate:        flagsValue(flags, "from-date"),
			FromTime:        flagsValue(flags, "from-time"),
			ToDate:          flagsValue(flags, "to-date"),
			ToTime:          flagsValue(flags, "to-time"),
		})
	case "leave":
		return features.ExecuteLeave(regNo, cookies, features.LeaveApplyInput{
			Apply:         parseBoolFlag(flags, "apply"),
			LeaveCode:     flagsValue(flags, "leave-code"),
			VisitingPlace: flagsValue(flags, "visiting-place"),
			Reason:        flagsValue(flags, "reason"),
			FromDate:      flagsValue(flags, "from-date"),
			FromTime:      flagsValue(flags, "from-time"),
			ToDate:        flagsValue(flags, "to-date"),
			ToTime:        flagsValue(flags, "to-time"),
		})
	case "msg":
		features.GetClassMessage(regNo, cookies)
	case "da":
		features.PrintAllDAs(regNo, cookies, flagsValue(flags, "course"), flagsValue(flags, "assignment"))
	case "facility":
		return features.RegisterPhyFacility(regNo, cookies, flagsValue(flags, "facility"), parseBoolFlag(flags, "confirm"))
	case "syllabus":
		features.ExecuteSyllabusDownload(regNo, cookies, flagsValue(flags, "course"))
	default:
		return fmt.Errorf("unsupported proxy command: %s", command)
	}
	return nil
}

func runSyncProxyCommand(resp *proxyResponse, flags map[string]string, cookies types.Cookies, regNo string, start time.Time) {
	commands := resolveSyncCommands(flags)
	if len(commands) == 0 {
		resp.Error = &proxyError{Kind: "invalid_flags", Message: "no commands provided for sync"}
		resp.StructuredRaw = map[string]any{"results": []syncResultEntry{}}
		resp.Success = false
		resp.Output = ""
		resp.print(start)
		return
	}

	results := make([]syncResultEntry, 0, len(commands))
	allSuccess := true

	for _, cmd := range commands {
		entry := executeSyncSubcommand(cmd, flags, cookies, regNo)
		if !entry.Success {
			allSuccess = false
		}
		results = append(results, entry)
	}

	resp.StructuredRaw = map[string]any{"results": results}
	resp.Output = ""
	resp.Success = allSuccess
	if !allSuccess {
		resp.Error = &proxyError{Kind: "partial_failure", Message: "one or more sync commands failed"}
	}
	resp.print(start)
}

func resolveSyncCommands(flags map[string]string) []string {
	raw := strings.TrimSpace(flags["commands"])
	if raw == "" {
		return append([]string(nil), defaultSyncCommands...)
	}
	parts := strings.Split(raw, ",")
	var commands []string
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed == "" {
			continue
		}
		commands = append(commands, trimmed)
	}
	if len(commands) == 0 {
		return append([]string(nil), defaultSyncCommands...)
	}
	return commands
}

func executeSyncSubcommand(command string, flags map[string]string, cookies types.Cookies, regNo string) syncResultEntry {
	entry := syncResultEntry{Command: command}
	subFlags := cloneFlags(flags)
	delete(subFlags, "commands")

	tableSnapshots := []helpers.TableSnapshot{}
	restore := helpers.RegisterTableCaptureHook(func(snapshot helpers.TableSnapshot) {
		tableSnapshots = append(tableSnapshots, snapshot)
	})
	defer restore()

	selectionRequests := []helpers.ProxySelectionRequest{}
	restoreSelection := helpers.RegisterProxySelectionHook(func(request helpers.ProxySelectionRequest) {
		selectionRequests = append(selectionRequests, request)
	})
	defer restoreSelection()

	output, execErr := captureCommandOutput(func() error {
		return executeFeatureCommand(command, subFlags, cookies, regNo)
	}, nil)

	cleaned, messages := proxyutil.NormalizeOutput(output)
	if cleaned != "" {
		entry.Output = cleaned
	}
	entry.StructuredRaw = proxyutil.BuildStructuredData(tableSnapshots, selectionRequests, messages)

	if len(selectionRequests) > 0 {
		entry.Error = &proxyError{Kind: "selection_required", Message: proxyutil.SelectionRequiredMessage(selectionRequests)}
		entry.Success = false
	} else if execErr != nil {
		entry.Error = &proxyError{Kind: "execution_error", Message: execErr.Error()}
		entry.Success = false
	} else {
		entry.Success = true
	}

	return entry
}

func cloneFlags(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	copy := make(map[string]string, len(src))
	for key, value := range src {
		copy[key] = value
	}
	return copy
}

func parseBoolFlag(flags map[string]string, key string) bool {
	if flags == nil {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(flags[key])) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseIntFlag(flags map[string]string, key string) int {
	if flags == nil {
		return 0
	}
	value := flags[key]
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func flagsValue(flags map[string]string, key string) string {
	if flags == nil {
		return ""
	}
	return flags[key]
}

func isInteractiveProxyCommand(command string) bool {
	_, ok := interactiveProxyCommands[command]
	return ok
}
