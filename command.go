package opskit

import (
	"context"
	"encoding/json"
	"time"
)

// CommandRequest describes an operational command invocation.
//
// Commands are control-plane operations, not business request handlers.
// Examples: config/reload, cache/refresh, index/rebuild, dependency/check.
type CommandRequest struct {
	Name string `json:"name"`
	// Payload is command-specific data supplied by the caller. Command
	// handlers are responsible for validating and interpreting it. Presentation
	// layers that accept payloads from users must provide authentication,
	// authorization, and request size limits before constructing CommandRequest.
	Payload     json.RawMessage `json:"payload,omitempty"`
	RequestedAt *time.Time      `json:"requested_at,omitempty"`
	Attributes  []Attribute     `json:"attributes,omitempty"`
}

// CommandResult describes the outcome of an operational command invocation.
type CommandResult struct {
	State State `json:"state"`
	// Accepted means the command was admitted for execution, not necessarily
	// completed.
	Accepted bool   `json:"accepted"`
	Message  string `json:"message,omitempty"`
	// Error is exposed through operational surfaces. Command handlers must not
	// include secrets, credentials, tokens, raw connection strings, or
	// unredacted user data.
	Error      string     `json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Duration   Duration   `json:"duration,omitempty"`
	// Result is command-specific output. Command handlers must only return
	// values that are safe for operational surfaces.
	Result     any         `json:"result,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}

// CommandHandler handles an operational command.
type CommandHandler interface {
	HandleCommand(context.Context, CommandRequest) CommandResult
}

// CommandHandlerFunc adapts a function into a CommandHandler.
type CommandHandlerFunc func(context.Context, CommandRequest) CommandResult

// HandleCommand invokes the function-backed command handler when the caller
// explicitly calls it.
func (fn CommandHandlerFunc) HandleCommand(ctx context.Context, request CommandRequest) CommandResult {
	ctx = normalizeContext(ctx)

	if fn == nil {
		return CommandResult{
			State:    StateUnknown,
			Accepted: false,
			Message:  "command handler function is not configured",
		}
	}

	return fn(ctx, request)
}

// AcceptedCommand returns a command result for accepted asynchronous work.
func AcceptedCommand(message string, attrs ...Attribute) CommandResult {
	return CommandResult{
		State:      StateInitializing,
		Accepted:   true,
		Message:    message,
		StartedAt:  nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}

// CompletedCommand returns a successful completed command result.
//
// The result value may be exposed through operational surfaces, so callers
// should pass only safe, redacted output.
func CompletedCommand(message string, result any, duration time.Duration, attrs ...Attribute) CommandResult {
	now := nowUTC()

	return CommandResult{
		State:      StateReady,
		Accepted:   true,
		Message:    message,
		StartedAt:  timeUTCAt(now.Add(-duration)),
		FinishedAt: now,
		Duration:   NewDuration(duration),
		Result:     result,
		Attributes: cloneAttributes(attrs),
	}
}

// RejectedCommand returns a command result for work that was not accepted.
func RejectedCommand(message string, attrs ...Attribute) CommandResult {
	return CommandResult{
		State:      StateNotReady,
		Accepted:   false,
		Message:    message,
		FinishedAt: nowUTC(),
		Attributes: cloneAttributes(attrs),
	}
}

// FailedCommand returns a failed command result.
//
// The error text may be exposed through operational surfaces, so callers should
// pass only safe, redacted errors.
func FailedCommand(message string, err error, duration time.Duration, attrs ...Attribute) CommandResult {
	now := nowUTC()

	result := CommandResult{
		State:      StateFailed,
		Accepted:   true,
		Message:    message,
		StartedAt:  timeUTCAt(now.Add(-duration)),
		FinishedAt: now,
		Duration:   NewDuration(duration),
		Attributes: cloneAttributes(attrs),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result
}
