package opskit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCheckConstructors(t *testing.T) {
	tests := []struct {
		name      string
		result    CheckResult
		wantState State
		wantReady bool
		wantError string
	}{
		{
			name:      "ready",
			result:    ReadyCheck("ready", 150*time.Millisecond, Attr("target", "cache")),
			wantState: StateReady,
			wantReady: true,
		},
		{
			name:      "degraded",
			result:    DegradedCheck("degraded", 150*time.Millisecond, Attr("target", "cache")),
			wantState: StateDegraded,
			wantReady: true,
		},
		{
			name:      "not ready",
			result:    NotReadyCheck("not ready", 150*time.Millisecond, Attr("target", "cache")),
			wantState: StateNotReady,
			wantReady: false,
		},
		{
			name:      "failed",
			result:    FailedCheck("failed", errors.New("boom"), 150*time.Millisecond, Attr("target", "cache")),
			wantState: StateFailed,
			wantReady: false,
			wantError: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.State != tt.wantState {
				t.Fatalf("State = %q, want %q", tt.result.State, tt.wantState)
			}
			if tt.result.Ready != tt.wantReady {
				t.Fatalf("Ready = %t, want %t", tt.result.Ready, tt.wantReady)
			}
			if tt.result.Message != tt.name {
				t.Fatalf("Message = %q, want %q", tt.result.Message, tt.name)
			}
			if tt.result.Error != tt.wantError {
				t.Fatalf("Error = %q, want %q", tt.result.Error, tt.wantError)
			}
			if tt.result.CheckedAt == nil {
				t.Fatal("CheckedAt is nil")
			}
			if tt.result.CheckedAt.Location() != time.UTC {
				t.Fatalf("CheckedAt location = %q, want UTC", tt.result.CheckedAt.Location())
			}
			if tt.result.Duration.TimeDuration() != 150*time.Millisecond {
				t.Fatalf("Duration = %v, want 150ms", tt.result.Duration.TimeDuration())
			}
			if len(tt.result.Attributes) != 1 || tt.result.Attributes[0] != Attr("target", "cache") {
				t.Fatalf("Attributes = %+v, want target cache", tt.result.Attributes)
			}
		})
	}
}

func TestCheckConstructorsCloneAttributes(t *testing.T) {
	constructors := map[string]func(string, time.Duration, ...Attribute) CheckResult{
		"ready":     ReadyCheck,
		"degraded":  DegradedCheck,
		"not_ready": NotReadyCheck,
	}

	for name, constructor := range constructors {
		t.Run(name, func(t *testing.T) {
			attrs := []Attribute{
				Attr("target", "cache"),
				Attr("shard", "primary"),
			}

			result := constructor("check", 0, attrs...)
			attrs[0] = Attr("target", "mutated")

			if len(result.Attributes) != 2 {
				t.Fatalf("Attributes length = %d, want 2", len(result.Attributes))
			}
			if result.Attributes[0] != Attr("target", "cache") {
				t.Fatalf("Attributes[0] = %+v, want target cache", result.Attributes[0])
			}
			if result.Attributes[1] != Attr("shard", "primary") {
				t.Fatalf("Attributes[1] = %+v, want shard primary", result.Attributes[1])
			}
		})
	}
}

func TestFailedCheckWithNilError(t *testing.T) {
	result := FailedCheck("failed", nil, 0)

	if result.State != StateFailed {
		t.Fatalf("State = %q, want %q", result.State, StateFailed)
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
	if result.Error != "" {
		t.Fatalf("Error = %q, want empty", result.Error)
	}
}

func TestFailedCheckClonesAttributes(t *testing.T) {
	attrs := []Attribute{
		Attr("target", "cache"),
	}

	result := FailedCheck("failed", errors.New("boom"), 0, attrs...)
	attrs[0] = Attr("target", "mutated")

	if len(result.Attributes) != 1 {
		t.Fatalf("Attributes length = %d, want 1", len(result.Attributes))
	}
	if result.Attributes[0] != Attr("target", "cache") {
		t.Fatalf("Attributes[0] = %+v, want target cache", result.Attributes[0])
	}
}

func TestCheckFunc(t *testing.T) {
	ctx := context.Background()
	checker := CheckFunc(func(got context.Context) CheckResult {
		if got != ctx {
			t.Fatal("context was not passed through")
		}
		return ReadyCheck("ready", 0)
	})

	result := checker.Check(ctx)
	if result.State != StateReady {
		t.Fatalf("State = %q, want %q", result.State, StateReady)
	}
}

func TestCheckFuncNormalizesNilContext(t *testing.T) {
	var ctx context.Context

	CheckFunc(func(ctx context.Context) CheckResult {
		if ctx == nil {
			t.Fatal("context is nil, want normalized context")
		}
		return ReadyCheck("ready", 0)
	}).Check(ctx)
}

func TestNilCheckFunc(t *testing.T) {
	var checker CheckFunc

	result := checker.Check(context.Background())
	if result.State != StateUnknown {
		t.Fatalf("State = %q, want %q", result.State, StateUnknown)
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
	if result.Message != "check function is not configured" {
		t.Fatalf("Message = %q, want check function is not configured", result.Message)
	}
}

func TestCheckGroupFunc(t *testing.T) {
	ctx := context.Background()
	group := CheckGroupFunc(func(got context.Context) CheckSummary {
		if got != ctx {
			t.Fatal("context was not passed through")
		}
		return CheckSummary{State: StateReady, Ready: true}
	})

	summary := group.CheckAll(ctx)
	if summary.State != StateReady {
		t.Fatalf("State = %q, want %q", summary.State, StateReady)
	}
}

func TestCheckGroupFuncNormalizesNilContext(t *testing.T) {
	var ctx context.Context

	CheckGroupFunc(func(ctx context.Context) CheckSummary {
		if ctx == nil {
			t.Fatal("context is nil, want normalized context")
		}
		return CheckSummary{State: StateReady, Ready: true}
	}).CheckAll(ctx)
}

func TestNilCheckGroupFunc(t *testing.T) {
	var group CheckGroupFunc

	summary := group.CheckAll(context.Background())
	if summary.State != StateUnknown {
		t.Fatalf("State = %q, want %q", summary.State, StateUnknown)
	}
	if summary.Ready {
		t.Fatal("Ready = true, want false")
	}
	if summary.Message != "check group function is not configured" {
		t.Fatalf("Message = %q, want check group function is not configured", summary.Message)
	}
}

func TestSummarizeChecksWithNoResults(t *testing.T) {
	summary := SummarizeChecks("", time.Now().UTC(), nil)

	if summary.State != StateUnknown {
		t.Fatalf("State = %q, want %q", summary.State, StateUnknown)
	}
	if summary.Ready {
		t.Fatal("Ready = true, want false")
	}
	if summary.Message != "no checks ran" {
		t.Fatalf("Message = %q, want no checks ran", summary.Message)
	}
	if summary.CheckedAt == nil {
		t.Fatal("CheckedAt is nil")
	}
	if summary.Results != nil {
		t.Fatalf("Results = %+v, want nil", summary.Results)
	}
}

func TestSummarizeChecksAllReady(t *testing.T) {
	results := []NamedCheck{
		{Name: "cache", Kind: "dependency", Result: ReadyCheck("ready", 0)},
	}

	summary := SummarizeChecks("", time.Now().UTC(), results)
	results[0].Name = "mutated"

	if summary.State != StateReady {
		t.Fatalf("State = %q, want %q", summary.State, StateReady)
	}
	if !summary.Ready {
		t.Fatal("Ready = false, want true")
	}
	if summary.Message != "all checks ready" {
		t.Fatalf("Message = %q, want all checks ready", summary.Message)
	}
	if len(summary.Results) != 1 || summary.Results[0].Name != "cache" {
		t.Fatalf("Results = %+v, want cloned cache result", summary.Results)
	}
}

func TestSummarizeChecksDegraded(t *testing.T) {
	results := []NamedCheck{
		{Name: "cache", Result: DegradedCheck("slow", 0)},
	}

	summary := SummarizeChecks("", time.Now().UTC(), results)
	if summary.State != StateDegraded {
		t.Fatalf("State = %q, want %q", summary.State, StateDegraded)
	}
	if !summary.Ready {
		t.Fatal("Ready = false, want true")
	}
	if summary.Message != "one or more checks degraded" {
		t.Fatalf("Message = %q, want one or more checks degraded", summary.Message)
	}
}

func TestSummarizeChecksNotReady(t *testing.T) {
	results := []NamedCheck{
		{Name: "cache", Result: NotReadyCheck("down", 0)},
		{Name: "database", Result: DegradedCheck("slow", 0)},
	}

	summary := SummarizeChecks("", time.Now().UTC(), results)
	if summary.State != StateNotReady {
		t.Fatalf("State = %q, want %q", summary.State, StateNotReady)
	}
	if summary.Ready {
		t.Fatal("Ready = true, want false")
	}
	if summary.Message != "one or more checks are not ready" {
		t.Fatalf("Message = %q, want one or more checks are not ready", summary.Message)
	}
}

func TestSummarizeChecksFailed(t *testing.T) {
	results := []NamedCheck{
		{Name: "cache", Result: NotReadyCheck("down", 0)},
		{Name: "database", Result: FailedCheck("failed", errors.New("boom"), 0)},
		{Name: "search", Result: DegradedCheck("slow", 0)},
	}

	summary := SummarizeChecks("", time.Now().UTC(), results)
	if summary.State != StateFailed {
		t.Fatalf("State = %q, want %q", summary.State, StateFailed)
	}
	if summary.Ready {
		t.Fatal("Ready = true, want false")
	}
	if summary.Message != "one or more checks failed" {
		t.Fatalf("Message = %q, want one or more checks failed", summary.Message)
	}
}

func TestSummarizeChecksPreservesMessage(t *testing.T) {
	summary := SummarizeChecks("custom message", time.Now().UTC(), []NamedCheck{
		{Name: "cache", Result: ReadyCheck("ready", 0)},
	})

	if summary.Message != "custom message" {
		t.Fatalf("Message = %q, want custom message", summary.Message)
	}
}

func TestCloneNamedChecks(t *testing.T) {
	results := []NamedCheck{
		{Name: "cache", Kind: "dependency", Result: ReadyCheck("ready", 0)},
	}

	cloned := cloneNamedChecks(results)
	results[0].Name = "mutated"

	if len(cloned) != 1 {
		t.Fatalf("cloned length = %d, want 1", len(cloned))
	}
	if cloned[0].Name != "cache" {
		t.Fatalf("cloned[0].Name = %q, want cache", cloned[0].Name)
	}
	if got := cloneNamedChecks(nil); got != nil {
		t.Fatalf("cloneNamedChecks(nil) = %+v, want nil", got)
	}
	if got := cloneNamedChecks([]NamedCheck{}); got != nil {
		t.Fatalf("cloneNamedChecks(empty) = %+v, want nil", got)
	}
}

func TestCheckResultJSONOmitEmptyFields(t *testing.T) {
	data, err := json.Marshal(CheckResult{
		State: StateReady,
		Ready: true,
	})
	if err != nil {
		t.Fatalf("Marshal CheckResult error = %v", err)
	}

	want := `{"state":"ready","ready":true}`
	if string(data) != want {
		t.Fatalf("Marshal CheckResult = %s, want %s", data, want)
	}
}

func TestCheckSummaryJSONOmitEmptyFields(t *testing.T) {
	data, err := json.Marshal(CheckSummary{
		State: StateReady,
		Ready: true,
	})
	if err != nil {
		t.Fatalf("Marshal CheckSummary error = %v", err)
	}

	want := `{"state":"ready","ready":true}`
	if string(data) != want {
		t.Fatalf("Marshal CheckSummary = %s, want %s", data, want)
	}
}

func TestNamedCheckJSON(t *testing.T) {
	data, err := json.Marshal(NamedCheck{
		Name: "cache",
		Kind: "dependency",
		Result: CheckResult{
			State: StateReady,
			Ready: true,
		},
	})
	if err != nil {
		t.Fatalf("Marshal NamedCheck error = %v", err)
	}

	want := `{"name":"cache","kind":"dependency","result":{"state":"ready","ready":true}}`
	if string(data) != want {
		t.Fatalf("Marshal NamedCheck = %s, want %s", data, want)
	}
}
