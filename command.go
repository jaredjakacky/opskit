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

// NewCommandRequest returns a command request with the current UTC timestamp.
//
// NewCommandRequest does not validate the command name or payload. Callers that
// need a custom RequestedAt value can construct CommandRequest directly.
func NewCommandRequest(name string, payload json.RawMessage, attrs ...Attribute) CommandRequest {
	return CommandRequest{
		Name:        name,
		Payload:     cloneRawMessage(payload),
		RequestedAt: nowUTC(),
		Attributes:  cloneAttributes(attrs),
	}
}

// CommandDescriptor describes one supported operational command.
//
// Descriptors are passive metadata for presentation, documentation, and
// execution layers. They do not validate, authorize, route, schedule, or
// execute commands.
type CommandDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// PayloadKind is a human- and tool-readable payload category. It is not a
	// schema, validator, or authorization rule.
	PayloadKind string `json:"payload_kind,omitempty"`
	// Dangerous is an advisory hint for presentation and execution layers.
	// Opskit does not enforce safety policy from this value.
	Dangerous bool `json:"dangerous,omitempty"`
	// Idempotent is an advisory hint for presentation and execution layers.
	// Opskit does not enforce retry, scheduling, or execution policy from this
	// value.
	Idempotent bool        `json:"idempotent,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
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

// CommandHandler handles an active operational command.
//
// HandleCommand is an execution hook. Opskit does not dispatch it, wrap it with
// timeout or panic recovery, retry it, authorize it, audit it, limit its
// concurrency, validate its payload, or export telemetry for it. Callers that
// invoke HandleCommand own those policies.
type CommandHandler interface {
	HandleCommand(context.Context, CommandRequest) CommandResult
}

// CommandDescriber reports the operational commands a component supports.
type CommandDescriber interface {
	Commands(context.Context) []CommandDescriptor
}

func cloneCommandDescriptors(commands []CommandDescriptor) []CommandDescriptor {
	if len(commands) == 0 {
		return nil
	}

	cloned := make([]CommandDescriptor, len(commands))
	for i, command := range commands {
		command.Attributes = cloneAttributes(command.Attributes)
		cloned[i] = command
	}
	return cloned
}

func cloneRawMessage(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return nil
	}

	cloned := make(json.RawMessage, len(payload))
	copy(cloned, payload)
	return cloned
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
