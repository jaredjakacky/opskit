package opskit

import (
	"context"
	"time"
)

// CheckResult describes the result of one active operational check.
//
// A check is an active operation: ping a dependency, verify a client target,
// validate a local resource, or probe some component-owned condition.
type CheckResult struct {
	State   State  `json:"state"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
	// Error is exposed through operational surfaces. Callers must not include
	// secrets, credentials, tokens, raw connection strings, or unredacted user
	// data.
	Error      string      `json:"error,omitempty"`
	CheckedAt  *time.Time  `json:"checked_at,omitempty"`
	Duration   Duration    `json:"duration,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}

// Checker performs one operational check.
type Checker interface {
	Check(context.Context) CheckResult
}

// CheckSummary describes the aggregate result of a group of operational checks.
type CheckSummary struct {
	State     State        `json:"state"`
	Ready     bool         `json:"ready"`
	Message   string       `json:"message,omitempty"`
	CheckedAt *time.Time   `json:"checked_at,omitempty"`
	Duration  Duration     `json:"duration,omitempty"`
	Results   []NamedCheck `json:"results,omitempty"`
}

// NamedCheck is one named check result inside a CheckSummary.
type NamedCheck struct {
	Name   string      `json:"name"`
	Kind   string      `json:"kind,omitempty"`
	Result CheckResult `json:"result"`
}

// CheckGroup performs a group of operational checks.
type CheckGroup interface {
	CheckAll(context.Context) CheckSummary
}

// ReadyCheck returns a ready check result with the current UTC timestamp.
func ReadyCheck(message string, duration time.Duration, attrs ...Attribute) CheckResult {
	return CheckResult{
		State:      StateReady,
		Ready:      true,
		Message:    message,
		CheckedAt:  nowUTC(),
		Duration:   NewDuration(duration),
		Attributes: cloneAttributes(attrs),
	}
}

// DegradedCheck returns a degraded check result with the current UTC timestamp.
//
// Degraded checks are still ready by default. Domain kits can use NotReadyCheck
// or FailedCheck when degraded state should block readiness.
func DegradedCheck(message string, duration time.Duration, attrs ...Attribute) CheckResult {
	return CheckResult{
		State:      StateDegraded,
		Ready:      true,
		Message:    message,
		CheckedAt:  nowUTC(),
		Duration:   NewDuration(duration),
		Attributes: cloneAttributes(attrs),
	}
}

// NotReadyCheck returns a not-ready check result with the current UTC timestamp.
func NotReadyCheck(message string, duration time.Duration, attrs ...Attribute) CheckResult {
	return CheckResult{
		State:      StateNotReady,
		Ready:      false,
		Message:    message,
		CheckedAt:  nowUTC(),
		Duration:   NewDuration(duration),
		Attributes: cloneAttributes(attrs),
	}
}

// FailedCheck returns a failed check result with the current UTC timestamp.
//
// The error text may be exposed through operational surfaces, so callers should
// pass only safe, redacted errors.
func FailedCheck(message string, err error, duration time.Duration, attrs ...Attribute) CheckResult {
	result := CheckResult{
		State:      StateFailed,
		Ready:      false,
		Message:    message,
		CheckedAt:  nowUTC(),
		Duration:   NewDuration(duration),
		Attributes: cloneAttributes(attrs),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result
}

// CheckFunc adapts a function into a Checker.
type CheckFunc func(context.Context) CheckResult

// Check invokes the function-backed check when the caller explicitly calls it.
func (fn CheckFunc) Check(ctx context.Context) CheckResult {
	ctx = normalizeContext(ctx)

	if fn == nil {
		return CheckResult{
			State:   StateUnknown,
			Ready:   false,
			Message: "check function is not configured",
		}
	}

	return fn(ctx)
}

// CheckGroupFunc adapts a function into a CheckGroup.
type CheckGroupFunc func(context.Context) CheckSummary

// CheckAll invokes the function-backed check group when the caller explicitly
// calls it.
func (fn CheckGroupFunc) CheckAll(ctx context.Context) CheckSummary {
	ctx = normalizeContext(ctx)

	if fn == nil {
		return CheckSummary{
			State:   StateUnknown,
			Ready:   false,
			Message: "check group function is not configured",
		}
	}

	return fn(ctx)
}

// SummarizeChecks builds a CheckSummary from named check results.
//
// The summary is ready only when every result is ready. The summary state is a
// coarse check summary: unknown when no checks ran, failed when any check
// failed, not_ready when any other check is not ready, degraded when all checks
// are ready but at least one is degraded, and ready otherwise.
func SummarizeChecks(message string, startedAt time.Time, results []NamedCheck) CheckSummary {
	finishedAt := nowUTC()

	summary := CheckSummary{
		State:     StateReady,
		Ready:     true,
		Message:   message,
		CheckedAt: finishedAt,
		Duration:  NewDuration(finishedAt.Sub(startedAt)),
		Results:   cloneNamedChecks(results),
	}

	if len(results) == 0 {
		summary.State = StateUnknown
		summary.Ready = false
		if summary.Message == "" {
			summary.Message = "no checks ran"
		}
		return summary
	}

	failed := false
	notReady := false
	degraded := false
	for _, result := range results {
		if result.Result.State == StateFailed {
			failed = true
		}
		if !result.Result.Ready {
			notReady = true
		}
		if result.Result.State == StateDegraded {
			degraded = true
		}
	}

	if failed {
		summary.Ready = false
		summary.State = StateFailed
		if summary.Message == "" {
			summary.Message = "one or more checks failed"
		}
		return summary
	}

	if notReady {
		summary.Ready = false
		summary.State = StateNotReady
		if summary.Message == "" {
			summary.Message = "one or more checks are not ready"
		}
		return summary
	}

	if summary.Ready && degraded {
		summary.State = StateDegraded
	}

	if summary.Ready && summary.Message == "" && summary.State == StateDegraded {
		summary.Message = "one or more checks degraded"
	}

	if summary.Ready && summary.Message == "" {
		summary.Message = "all checks ready"
	}

	return summary
}

func cloneNamedChecks(results []NamedCheck) []NamedCheck {
	if len(results) == 0 {
		return nil
	}

	cloned := make([]NamedCheck, len(results))
	copy(cloned, results)
	return cloned
}
