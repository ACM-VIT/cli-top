// Package contract defines the public, versioned CLI result protocol.
package contract

import (
	"context"
	"errors"
	"io/fs"
	"net"
)

const Version = "1"

type Action struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Scope string `json:"scope,omitempty"`
	Index int    `json:"index,omitempty"`
}

type Error struct {
	Kind          string   `json:"kind"`
	Message       string   `json:"message"`
	Retryable     bool     `json:"retryable"`
	RequiredFlags []string `json:"required_flags,omitempty"`
	Choices       []Choice `json:"choices,omitempty"`
	Next          []Action `json:"next,omitempty"`
	Status        int      `json:"http_status,omitempty"`
	Cause         error    `json:"-"`
}

func (e *Error) Error() string        { return e.Message }
func (e *Error) Unwrap() error        { return e.Cause }
func New(kind, message string) *Error { return &Error{Kind: kind, Message: message} }
func Wrap(kind, message string, cause error) *Error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	if errors.Is(err, context.Canceled) {
		return Wrap("canceled", "Operation canceled.", err)
	}
	var path *fs.PathError
	if errors.As(err, &path) {
		return Wrap("io_error", "Unable to read or write a local file.", err)
	}
	var n net.Error
	if errors.As(err, &n) || errors.Is(err, context.DeadlineExceeded) {
		e = Wrap("network_error", "Unable to reach VTOP.", err)
		e.Retryable = true
		return e
	}
	return Wrap("internal_error", "The operation could not be completed.", err)
}
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch From(err).Kind {
	case "invalid_arguments", "invalid_selection":
		return 2
	case "auth_required", "auth_failed":
		return 3
	case "selection_required", "confirmation_required", "human_action_required":
		return 4
	case "network_error", "upstream_error", "parse_error":
		return 5
	case "client_decommissioned", "policy_blocked":
		return 6
	case "partial_failure":
		return 7
	case "canceled":
		return 130
	default:
		return 1
	}
}

type Warning struct {
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Component string `json:"component,omitempty"`
}
type Context struct {
	Source          string  `json:"source,omitempty"`
	Semester        *Choice `json:"semester,omitempty"`
	ClassGroup      *Choice `json:"class_group,omitempty"`
	SelectionSource string  `json:"selection_source,omitempty"`
	Timezone        string  `json:"timezone,omitempty"`
}
type Result struct {
	SchemaVersion string    `json:"schema_version"`
	Command       string    `json:"command"`
	Success       bool      `json:"success"`
	Data          any       `json:"data"`
	Context       Context   `json:"context"`
	Warnings      []Warning `json:"warnings"`
	Error         *Error    `json:"error"`
}

func Reply(command string, data any, ctx Context, warnings []Warning, err error) Result {
	if warnings == nil {
		warnings = []Warning{}
	}
	return Result{SchemaVersion: Version, Command: command, Success: err == nil, Data: data, Context: ctx, Warnings: warnings, Error: From(err)}
}
