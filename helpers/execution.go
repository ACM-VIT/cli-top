package helpers

import (
	"io"
	"os"
	"sync"

	"cli-top/internal/contract"
)

// ExecutionPolicy separates presentation and interaction from feature defaults.
// Legacy package entry points retain their historical behavior when unset.
type ExecutionPolicy struct {
	Machine        bool
	NonInteractive bool
	Input          io.Reader
	Output         io.Writer
	ErrorOutput    io.Writer
}

var execution struct {
	sync.RWMutex
	policy *ExecutionPolicy
}

func SetExecutionPolicy(p ExecutionPolicy) func() {
	execution.Lock()
	old := execution.policy
	execution.policy = &p
	execution.Unlock()
	return func() { execution.Lock(); execution.policy = old; execution.Unlock() }
}
func CurrentExecutionPolicy() *ExecutionPolicy {
	execution.RLock()
	defer execution.RUnlock()
	if execution.policy == nil {
		return nil
	}
	p := *execution.policy
	return &p
}
func CanPrompt() bool {
	if p := CurrentExecutionPolicy(); p != nil {
		return !p.Machine && !p.NonInteractive
	}
	return !ShouldMuteUI() && IsInteractiveOutput(os.Stdin)
}
func InputReader() io.Reader {
	if p := CurrentExecutionPolicy(); p != nil && p.Input != nil {
		return p.Input
	}
	return os.Stdin
}
func OutputWriter() io.Writer {
	if p := CurrentExecutionPolicy(); p != nil && p.Output != nil {
		return p.Output
	}
	return os.Stdout
}
func ErrorWriter() io.Writer {
	if p := CurrentExecutionPolicy(); p != nil && p.ErrorOutput != nil {
		return p.ErrorOutput
	}
	return os.Stderr
}
func RequireLogin(cookiesValid bool) error {
	if cookiesValid {
		return nil
	}
	e := contract.New("auth_required", "Run cli-top login to store valid VTOP credentials.")
	e.Next = []contract.Action{{Command: "cli-top", Args: []string{"login"}}}
	return e
}
