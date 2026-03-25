package tests

import (
	"bytes"
	"cli-top/debug"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"
	"time"
)

const (
	perfIterations     = 3
	perfOverheadSlack  = 150 * time.Millisecond
	perfSlowdownFactor = 1.5
)

var defaultPerfTimeout = 15 * time.Second

func TestCommandExecutionTimesLocalVsGlobal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf comparison in short mode")
	}

	globalPath, err := exec.LookPath("cli-top")
	if err != nil {
		t.Skip("global cli-top not found in PATH")
	}

	repoRoot := findRepoRoot(t)

	localPath := buildLocalBinary(t, repoRoot)
	if sameBinary(localPath, globalPath) {
		t.Skip("global cli-top matches local build; skipping comparison")
	}

	workDir := t.TempDir()
	realNetwork := os.Getenv("CLI_TOP_PERF_REAL") == "1"
	showOutput := os.Getenv("CLI_TOP_PERF_SHOW_OUTPUT") == "1"
	showGlobalOutput := os.Getenv("CLI_TOP_PERF_SHOW_GLOBAL_OUTPUT") == "1"

	var env []string
	if realNetwork {
		env = perfEnvReal()
	} else {
		writePerfConfig(t, workDir)
		env = perfEnv(workDir)
	}

	t.Logf("local bin: %s", localPath)
	t.Logf("global bin: %s", globalPath)
	if localReal, err := filepath.EvalSymlinks(localPath); err == nil && localReal != localPath {
		t.Logf("local resolved: %s", localReal)
	}
	if globalReal, err := filepath.EvalSymlinks(globalPath); err == nil && globalReal != globalPath {
		t.Logf("global resolved: %s", globalReal)
	}
	if localVersion := getVersion(localPath, env, workDir); localVersion != "" {
		t.Logf("local version: %s", localVersion)
	}
	if globalVersion := getVersion(globalPath, env, workDir); globalVersion != "" {
		t.Logf("global version: %s", globalVersion)
	}
	if info := fileInfo(globalPath); info != "" {
		t.Logf("global file: %s", info)
	}
	if info := fileInfo(localPath); info != "" {
		t.Logf("local file: %s", info)
	}

	commands, err := listCommands(localPath, env, workDir)
	if err != nil {
		t.Fatalf("failed to list commands: %v", err)
	}
	if len(commands) == 0 {
		t.Fatal("no commands discovered")
	}

	if filter := os.Getenv("CLI_TOP_PERF_COMMANDS"); filter != "" {
		commands = filterCommands(commands, filter)
		if len(commands) == 0 {
			t.Fatal("no commands matched CLI_TOP_PERF_COMMANDS")
		}
	}

	results := make([]commandResult, 0, len(commands))
	for _, cmdName := range commands {
		args := []string(nil)
		stdin := ""
		skipReason := ""
		if realNetwork {
			args, stdin, skipReason = resolveCommandRun(cmdName)
			if skipReason != "" {
				results = append(results, commandResult{name: cmdName, status: skipReason})
				continue
			}
		}

		localDur, localOut, localErr := measureCommand(localPath, cmdName, args, stdin, env, workDir, realNetwork)
		globalDur, globalOut, globalErr := measureCommand(globalPath, cmdName, args, stdin, env, workDir, realNetwork)

		status := "ok"
		var mergedErr error
		switch {
		case localErr != nil && globalErr != nil:
			status = "both error"
			mergedErr = fmt.Errorf("local: %v; global: %v", localErr, globalErr)
		case localErr != nil:
			status = "local error"
			mergedErr = localErr
		case globalErr != nil:
			status = "global error"
			mergedErr = globalErr
		default:
			threshold := time.Duration(float64(globalDur)*perfSlowdownFactor) + perfOverheadSlack
			if localDur > threshold {
				status = "slower"
				t.Errorf("local %s slower than global: local=%s global=%s threshold=%s", cmdName, localDur, globalDur, threshold)
			}
		}

		results = append(results, commandResult{
			name:      cmdName,
			status:    status,
			local:     localDur,
			global:    globalDur,
			err:       mergedErr,
			localOut:  localOut,
			globalOut: globalOut,
		})
	}

	renderResults(t, results)

	if showOutput {
		renderOutputs(t, results, showGlobalOutput)
	}
}

func buildLocalBinary(t *testing.T, repoRoot string) string {
	t.Helper()
	outDir := t.TempDir()
	name := "cli-top"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binPath := filepath.Join(outDir, name)

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	return binPath
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found in parent directories")
		}
		dir = parent
	}
}

func sameBinary(localPath, globalPath string) bool {
	localReal, _ := filepath.EvalSymlinks(localPath)
	globalReal, _ := filepath.EvalSymlinks(globalPath)
	return localReal != "" && localReal == globalReal
}

func writePerfConfig(t *testing.T, dir string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	content := strings.Join([]string{
		fmt.Sprintf("KILL_SWITCH_LAST_CHECK=\"%s\"", now),
		"KILL_SWITCH_LAST_VALUE=\"0\"",
		fmt.Sprintf("UPDATE_LAST_CHECK=\"%s\"", now),
		fmt.Sprintf("UPDATE_LAST_VERSION=\"%s\"", debug.Version),
		"UPDATE_LAST_HIGHLIGHT=\"\"",
		fmt.Sprintf("LAST_UPDATE_NOTIFIED_VERSION=\"%s\"", debug.Version),
	}, "\n") + "\n"

	path := filepath.Join(dir, "cli-top-config.env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write perf config: %v", err)
	}
}

func perfEnv(workDir string) []string {
	env := append([]string{}, os.Environ()...)
	setEnv(&env, "HOME", workDir)
	setEnv(&env, "USERPROFILE", workDir)
	setEnv(&env, "XDG_CONFIG_HOME", workDir)
	setEnv(&env, "APPDATA", workDir)
	setEnv(&env, "CLI_TOP_PROXY_MODE", "1")
	setEnv(&env, "NO_COLOR", "1")
	setEnv(&env, "TERM", "dumb")
	return env
}

func perfEnvReal() []string {
	env := append([]string{}, os.Environ()...)
	setEnv(&env, "CLI_TOP_PROXY_MODE", "1")
	setEnv(&env, "NO_COLOR", "1")
	setEnv(&env, "TERM", "dumb")
	return env
}

func setEnv(env *[]string, key, value string) {
	prefix := key + "="
	for i, v := range *env {
		if strings.HasPrefix(v, prefix) {
			(*env)[i] = prefix + value
			return
		}
	}
	*env = append(*env, prefix+value)
}

func listCommands(bin string, env []string, dir string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), perfTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "--help")
	cmd.Env = env
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return parseCommands(out.String()), nil
}

func parseCommands(output string) []string {
	var commands []string
	inSection := false
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Available Subcommands") {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if trimmed == "" {
			if len(commands) > 0 {
				break
			}
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			break
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		cmd := fields[0]
		if cmd == "help" {
			continue
		}
		commands = append(commands, cmd)
	}
	sort.Strings(commands)
	return commands
}

func measureCommand(bin, command string, args []string, stdin string, env []string, dir string, realNetwork bool) (time.Duration, string, error) {
	iterations := perfIterations
	if realNetwork {
		iterations = 1
	}
	samples := make([]time.Duration, 0, iterations)
	var output string
	for i := 0; i < iterations; i++ {
		duration, out, err := runCommand(bin, command, args, stdin, env, dir, realNetwork)
		if err != nil {
			return 0, out, err
		}
		output = out
		samples = append(samples, duration)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return samples[len(samples)/2], output, nil
}

func runCommand(bin, command string, args []string, stdin string, env []string, dir string, realNetwork bool) (time.Duration, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), perfTimeout())
	defer cancel()

	fullArgs := []string{command}
	if len(args) > 0 {
		fullArgs = append(fullArgs, args...)
	} else if !realNetwork {
		fullArgs = append(fullArgs, "--help")
	}

	cmd := exec.CommandContext(ctx, bin, fullArgs...)
	cmd.Env = env
	cmd.Dir = dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		return duration, out.String(), fmt.Errorf("command timed out")
	}
	return duration, out.String(), err
}

func getVersion(bin string, env []string, dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), perfTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "--version")
	cmd.Env = env
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func fileInfo(path string) string {
	fileCmd, err := exec.LookPath("file")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), perfTimeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, fileCmd, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func perfTimeout() time.Duration {
	if value := os.Getenv("CLI_TOP_PERF_TIMEOUT"); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		if parsed, err := time.ParseDuration(value); err == nil {
			if parsed > 0 {
				return parsed
			}
		}
	}
	return defaultPerfTimeout
}

type commandResult struct {
	name      string
	status    string
	local     time.Duration
	global    time.Duration
	err       error
	localOut  string
	globalOut string
}

func filterCommands(commands []string, filter string) []string {
	allow := map[string]bool{}
	for _, item := range strings.Split(filter, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			allow[item] = true
		}
	}
	out := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if allow[cmd] {
			out = append(out, cmd)
		}
	}
	return out
}

func resolveCommandRun(cmd string) ([]string, string, string) {
	envKey := strings.ToUpper(strings.ReplaceAll(cmd, "-", "_"))
	if args := os.Getenv("CLI_TOP_PERF_ARGS_" + envKey); args != "" {
		return strings.Fields(args), os.Getenv("CLI_TOP_PERF_STDIN_" + envKey), ""
	}

	defaultArgs := map[string][]string{
		"attendance":          {"--semester", "1"},
		"calendar":            {"--semester", "1", "--class-group", "1"},
		"cgpa":                {},
		"completion":          {"bash"},
		"course-page":         {"--semester", "1", "--course", "1", "--faculty", "1"},
		"course-page-archive": {"--semester", "1", "--course", "1", "--faculty", "1"},
		"da":                  {},
		"exams":               {"--semester", "1"},
		"grades":              {"--semester", "1"},
		"holiday":             {"--semester", "1"},
		"marks":               {"--semester", "1"},
		"today":               {},
		"tomorrow":            {},
		"dayafter":            {},
		"timetable":           {"--semester", "1"},
		"syllabus":            {"--course", "1"},
		"hostel":              {},
		"leave":               {},
		"library-dues":        {},
		"msg":                 {},
		"nightslip":           {},
		"profile":             {},
		"receipts":            {},
		"logout":              {},
	}

	if args, ok := defaultArgs[cmd]; ok {
		return args, "", ""
	}

	return nil, "", "skipped (needs input)"
}

func renderResults(t *testing.T, results []commandResult) {
	if len(results) == 0 {
		return
	}

	var (
		okCount     int
		slowerCount int
		skipCount   int
		errCount    int
	)
	for _, r := range results {
		switch r.status {
		case "ok":
			okCount++
		case "slower":
			slowerCount++
		default:
			if strings.HasPrefix(r.status, "skipped") {
				skipCount++
			} else if r.status != "" {
				errCount++
			}
		}
	}

	var table bytes.Buffer
	tw := tabwriter.NewWriter(&table, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMMAND\tSTATUS\tLOCAL\tGLOBAL\tRATIO\tDELTA")
	for _, r := range results {
		localStr := "-"
		globalStr := "-"
		ratioStr := "-"
		deltaStr := "-"
		if r.local > 0 {
			localStr = r.local.String()
		}
		if r.global > 0 {
			globalStr = r.global.String()
		}
		if r.local > 0 && r.global > 0 {
			ratio := float64(r.local) / float64(r.global)
			ratioStr = fmt.Sprintf("%.2fx", ratio)
			delta := r.local - r.global
			deltaStr = delta.String()
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", r.name, r.status, localStr, globalStr, ratioStr, deltaStr)
		if r.err != nil {
			fmt.Fprintf(tw, "  error:\t%s\t\t\t\t\n", r.err)
		}
	}
	_ = tw.Flush()

	summary := fmt.Sprintf("Summary: ok=%d slower=%d skipped=%d errors=%d total=%d", okCount, slowerCount, skipCount, errCount, len(results))
	out := strings.TrimRight(table.String(), "\n")
	t.Logf("\n%s\n\n%s", summary, out)
}

func renderOutputs(t *testing.T, results []commandResult, showGlobal bool) {
	var blocks []string
	for _, r := range results {
		if r.status != "ok" && r.status != "slower" {
			continue
		}
		if r.localOut != "" {
			blocks = append(blocks, fmt.Sprintf("---- %s (local output) ----\n%s", r.name, strings.TrimSpace(r.localOut)))
		}
		if showGlobal && r.globalOut != "" {
			blocks = append(blocks, fmt.Sprintf("---- %s (global output) ----\n%s", r.name, strings.TrimSpace(r.globalOut)))
		}
	}
	if len(blocks) > 0 {
		t.Logf("\n%s", strings.Join(blocks, "\n\n"))
	}
}
