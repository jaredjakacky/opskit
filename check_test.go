package opskit

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestCheckConstructors(t *testing.T) {
	tests := []struct {
		name        string
		result      CheckResult
		wantState   State
		wantReady   bool
		wantFailure *Failure
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
			name:        "failed",
			result:      FailedCheckWithFailure("failed", Failure{Code: "unavailable", Message: "cache unavailable"}, 150*time.Millisecond, Attr("target", "cache")),
			wantState:   StateFailed,
			wantReady:   false,
			wantFailure: &Failure{Code: "unavailable", Message: "cache unavailable"},
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
			if !equalFailure(tt.result.Failure, tt.wantFailure) {
				t.Fatalf("Failure = %+v, want %+v", tt.result.Failure, tt.wantFailure)
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

func TestFailedCheckOmitsFailureByDefault(t *testing.T) {
	result := FailedCheck("failed", 0)

	if result.State != StateFailed {
		t.Fatalf("State = %q, want %q", result.State, StateFailed)
	}
	if result.Ready {
		t.Fatal("Ready = true, want false")
	}
	if result.Failure != nil {
		t.Fatalf("Failure = %+v, want nil", result.Failure)
	}
}

func TestFailedCheckWithZeroFailureOmitsFailure(t *testing.T) {
	result := FailedCheckWithFailure("failed", Failure{}, 0)
	if result.Failure != nil {
		t.Fatalf("Failure = %+v, want nil", result.Failure)
	}
}

func TestFailedCheckClonesAttributes(t *testing.T) {
	attrs := []Attribute{
		Attr("target", "cache"),
	}

	result := FailedCheck("failed", 0, attrs...)
	attrs[0] = Attr("target", "mutated")

	if len(result.Attributes) != 1 {
		t.Fatalf("Attributes length = %d, want 1", len(result.Attributes))
	}
	if result.Attributes[0] != Attr("target", "cache") {
		t.Fatalf("Attributes[0] = %+v, want target cache", result.Attributes[0])
	}
}

func TestCheckDescriptorJSON(t *testing.T) {
	descriptor := CheckDescriptor{
		Name:        "cache",
		Kind:        "dependency",
		Description: "ping primary cache",
		Attributes: []Attribute{
			Attr("target", "cache"),
		},
	}

	data, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("Marshal CheckDescriptor error = %v", err)
	}

	want := `{"name":"cache","kind":"dependency","description":"ping primary cache","attributes":[{"key":"target","value":"cache"}]}`
	if string(data) != want {
		t.Fatalf("Marshal CheckDescriptor = %s, want %s", data, want)
	}
}

func TestCheckDescriptorJSONOmitEmptyFields(t *testing.T) {
	data, err := json.Marshal(CheckDescriptor{Name: "cache"})
	if err != nil {
		t.Fatalf("Marshal CheckDescriptor error = %v", err)
	}

	want := `{"name":"cache"}`
	if string(data) != want {
		t.Fatalf("Marshal CheckDescriptor = %s, want %s", data, want)
	}
}

func TestCloneCheckDescriptors(t *testing.T) {
	input := []CheckDescriptor{
		{
			Name: "cache",
			Attributes: []Attribute{
				Attr("target", "cache"),
			},
		},
	}

	cloned := cloneCheckDescriptors(input)
	input[0].Name = "mutated"
	input[0].Attributes[0] = Attr("target", "mutated")

	if cloned[0].Name != "cache" {
		t.Fatalf("cloned[0].Name = %q, want cache", cloned[0].Name)
	}
	if cloned[0].Attributes[0] != Attr("target", "cache") {
		t.Fatalf("cloned[0].Attributes = %+v, want target cache", cloned[0].Attributes)
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

func TestSummarizeChecksDetachesNestedCheckResult(t *testing.T) {
	wantCheckedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	checkedAt := wantCheckedAt
	failure := Failure{}
	attributes := []Attribute{Attr("target", "cache")}
	results := []NamedCheck{
		{
			Name: "cache",
			Result: CheckResult{
				State:      StateFailed,
				Ready:      false,
				Failure:    &failure,
				CheckedAt:  &checkedAt,
				Attributes: attributes,
			},
		},
	}

	summary := SummarizeChecks("", time.Now().UTC(), results)
	checkedAt = checkedAt.Add(time.Hour)
	failure.Message = "mutated"
	attributes[0] = Attr("target", "mutated")

	got := summary.Results[0].Result
	if got.CheckedAt == nil || !got.CheckedAt.Equal(wantCheckedAt) {
		t.Fatalf("CheckedAt = %v, want %v", got.CheckedAt, wantCheckedAt)
	}
	if got.Failure == nil || *got.Failure != (Failure{}) {
		t.Fatalf("Failure = %+v, want non-nil zero failure", got.Failure)
	}
	if got.Attributes[0] != Attr("target", "cache") {
		t.Fatalf("Attributes = %+v, want original attributes", got.Attributes)
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
		{Name: "database", Result: FailedCheck("failed", 0)},
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

func equalFailure(got, want *Failure) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
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
		{
			Name: "cache",
			Kind: "dependency",
			Result: FailedCheckWithFailure(
				"ready",
				Failure{Code: "failed", Message: "safe failure"},
				0,
				Attr("target", "cache"),
			),
		},
	}

	cloned := cloneNamedChecks(results)
	results[0].Name = "mutated"
	results[0].Result.Failure.Message = "mutated"
	results[0].Result.Attributes[0] = Attr("target", "mutated")

	if len(cloned) != 1 {
		t.Fatalf("cloned length = %d, want 1", len(cloned))
	}
	if cloned[0].Name != "cache" {
		t.Fatalf("cloned[0].Name = %q, want cache", cloned[0].Name)
	}
	if cloned[0].Result.Attributes[0] != Attr("target", "cache") {
		t.Fatalf("cloned[0].Result.Attributes = %+v, want target cache", cloned[0].Result.Attributes)
	}
	if cloned[0].Result.Failure == nil || cloned[0].Result.Failure.Message != "safe failure" {
		t.Fatalf("cloned[0].Result.Failure = %+v, want safe failure", cloned[0].Result.Failure)
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

func TestCheckResultJSONIncludesFailure(t *testing.T) {
	data, err := json.Marshal(CheckResult{
		State:   StateFailed,
		Ready:   false,
		Failure: &Failure{Code: "timeout", Message: "dependency timed out"},
	})
	if err != nil {
		t.Fatalf("Marshal CheckResult error = %v", err)
	}

	want := `{"state":"failed","ready":false,"failure":{"code":"timeout","message":"dependency timed out"}}`
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
