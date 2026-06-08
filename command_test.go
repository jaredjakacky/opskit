package opskit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCommandRequestJSON(t *testing.T) {
	requestedAt := time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC)
	request := CommandRequest{
		Name:        "cache/refresh",
		Payload:     json.RawMessage(`{"force":true}`),
		RequestedAt: &requestedAt,
		Attributes: []Attribute{
			Attr("requested_by", "admin"),
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal CommandRequest error = %v", err)
	}

	want := `{"name":"cache/refresh","payload":{"force":true},"requested_at":"2026-06-04T12:30:00Z","attributes":[{"key":"requested_by","value":"admin"}]}`
	if string(data) != want {
		t.Fatalf("Marshal CommandRequest = %s, want %s", data, want)
	}
}

func TestCommandRequestJSONOmitEmptyFields(t *testing.T) {
	data, err := json.Marshal(CommandRequest{Name: "cache/refresh"})
	if err != nil {
		t.Fatalf("Marshal CommandRequest error = %v", err)
	}

	want := `{"name":"cache/refresh"}`
	if string(data) != want {
		t.Fatalf("Marshal CommandRequest = %s, want %s", data, want)
	}
}

func TestCommandRequestPayloadRawMessageRoundTrip(t *testing.T) {
	input := []byte(`{"name":"cache/refresh","payload":{"force":true,"targets":["primary"]}}`)

	var request CommandRequest
	if err := json.Unmarshal(input, &request); err != nil {
		t.Fatalf("Unmarshal CommandRequest error = %v", err)
	}

	if request.Name != "cache/refresh" {
		t.Fatalf("Name = %q, want cache/refresh", request.Name)
	}
	wantPayload := `{"force":true,"targets":["primary"]}`
	if string(request.Payload) != wantPayload {
		t.Fatalf("Payload = %s, want %s", request.Payload, wantPayload)
	}
}

func TestCommandDescriptorJSON(t *testing.T) {
	descriptor := CommandDescriptor{
		Name:        "cache/refresh",
		Description: "refresh cache entries",
		PayloadKind: "cache_refresh",
		Dangerous:   true,
		Idempotent:  true,
		Attributes: []Attribute{
			Attr("scope", "cache"),
		},
	}

	data, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("Marshal CommandDescriptor error = %v", err)
	}

	want := `{"name":"cache/refresh","description":"refresh cache entries","payload_kind":"cache_refresh","dangerous":true,"idempotent":true,"attributes":[{"key":"scope","value":"cache"}]}`
	if string(data) != want {
		t.Fatalf("Marshal CommandDescriptor = %s, want %s", data, want)
	}
}

func TestCommandDescriptorJSONOmitEmptyFields(t *testing.T) {
	data, err := json.Marshal(CommandDescriptor{Name: "cache/refresh"})
	if err != nil {
		t.Fatalf("Marshal CommandDescriptor error = %v", err)
	}

	want := `{"name":"cache/refresh"}`
	if string(data) != want {
		t.Fatalf("Marshal CommandDescriptor = %s, want %s", data, want)
	}
}

func TestCloneCommandDescriptors(t *testing.T) {
	input := []CommandDescriptor{
		{
			Name: "cache/refresh",
			Attributes: []Attribute{
				Attr("scope", "cache"),
			},
		},
	}

	cloned := cloneCommandDescriptors(input)
	input[0].Name = "mutated"
	input[0].Attributes[0] = Attr("scope", "mutated")

	if cloned[0].Name != "cache/refresh" {
		t.Fatalf("cloned[0].Name = %q, want cache/refresh", cloned[0].Name)
	}
	if cloned[0].Attributes[0] != Attr("scope", "cache") {
		t.Fatalf("cloned[0].Attributes = %+v, want scope cache", cloned[0].Attributes)
	}
}

func TestCommandHandlerFunc(t *testing.T) {
	ctx := context.Background()
	request := CommandRequest{
		Name:    "cache/refresh",
		Payload: json.RawMessage(`{"force":true}`),
	}

	handler := CommandHandlerFunc(func(got context.Context, gotRequest CommandRequest) CommandResult {
		if got != ctx {
			t.Fatal("context was not passed through")
		}
		if gotRequest.Name != request.Name {
			t.Fatalf("request name = %q, want %q", gotRequest.Name, request.Name)
		}
		if string(gotRequest.Payload) != string(request.Payload) {
			t.Fatalf("request payload = %s, want %s", gotRequest.Payload, request.Payload)
		}
		return CompletedCommand("completed", nil, 0)
	})

	result := handler.HandleCommand(ctx, request)
	if result.State != StateReady {
		t.Fatalf("State = %q, want %q", result.State, StateReady)
	}
}

func TestCommandHandlerFuncNormalizesNilContext(t *testing.T) {
	var ctx context.Context

	CommandHandlerFunc(func(ctx context.Context, request CommandRequest) CommandResult {
		if ctx == nil {
			t.Fatal("context is nil, want normalized context")
		}
		return CompletedCommand("completed", nil, 0)
	}).HandleCommand(ctx, CommandRequest{Name: "cache/refresh"})
}

func TestNilCommandHandlerFunc(t *testing.T) {
	var handler CommandHandlerFunc

	result := handler.HandleCommand(context.Background(), CommandRequest{Name: "cache/refresh"})
	if result.State != StateUnknown {
		t.Fatalf("State = %q, want %q", result.State, StateUnknown)
	}
	if result.Accepted {
		t.Fatal("Accepted = true, want false")
	}
	if result.Message != "command handler function is not configured" {
		t.Fatalf("Message = %q, want command handler function is not configured", result.Message)
	}
}

func TestAcceptedCommand(t *testing.T) {
	attrs := []Attribute{Attr("command", "refresh")}

	result := AcceptedCommand("accepted", attrs...)
	attrs[0] = Attr("command", "mutated")

	if result.State != StateInitializing {
		t.Fatalf("State = %q, want %q", result.State, StateInitializing)
	}
	if !result.Accepted {
		t.Fatal("Accepted = false, want true")
	}
	if result.Message != "accepted" {
		t.Fatalf("Message = %q, want accepted", result.Message)
	}
	if result.StartedAt == nil {
		t.Fatal("StartedAt is nil")
	}
	if result.StartedAt.Location() != time.UTC {
		t.Fatalf("StartedAt location = %q, want UTC", result.StartedAt.Location())
	}
	if len(result.Attributes) != 1 || result.Attributes[0] != Attr("command", "refresh") {
		t.Fatalf("Attributes = %+v, want command refresh", result.Attributes)
	}
}

func TestCompletedCommand(t *testing.T) {
	attrs := []Attribute{Attr("command", "refresh")}

	result := CompletedCommand("completed", map[string]any{"refreshed": true}, 150*time.Millisecond, attrs...)
	attrs[0] = Attr("command", "mutated")

	if result.State != StateReady {
		t.Fatalf("State = %q, want %q", result.State, StateReady)
	}
	if !result.Accepted {
		t.Fatal("Accepted = false, want true")
	}
	if result.Message != "completed" {
		t.Fatalf("Message = %q, want completed", result.Message)
	}
	if result.StartedAt == nil {
		t.Fatal("StartedAt is nil")
	}
	if result.FinishedAt == nil {
		t.Fatal("FinishedAt is nil")
	}
	if result.Duration.TimeDuration() != 150*time.Millisecond {
		t.Fatalf("Duration = %v, want 150ms", result.Duration.TimeDuration())
	}
	if !result.FinishedAt.After(*result.StartedAt) && !result.FinishedAt.Equal(*result.StartedAt) {
		t.Fatalf("FinishedAt = %v, want >= StartedAt %v", result.FinishedAt, result.StartedAt)
	}
	if result.Result == nil {
		t.Fatal("Result is nil, want command result payload")
	}
	if len(result.Attributes) != 1 || result.Attributes[0] != Attr("command", "refresh") {
		t.Fatalf("Attributes = %+v, want command refresh", result.Attributes)
	}
}

func TestRejectedCommand(t *testing.T) {
	attrs := []Attribute{Attr("command", "refresh")}

	result := RejectedCommand("rejected", attrs...)
	attrs[0] = Attr("command", "mutated")

	if result.State != StateNotReady {
		t.Fatalf("State = %q, want %q", result.State, StateNotReady)
	}
	if result.Accepted {
		t.Fatal("Accepted = true, want false")
	}
	if result.Message != "rejected" {
		t.Fatalf("Message = %q, want rejected", result.Message)
	}
	if result.FinishedAt == nil {
		t.Fatal("FinishedAt is nil")
	}
	if len(result.Attributes) != 1 || result.Attributes[0] != Attr("command", "refresh") {
		t.Fatalf("Attributes = %+v, want command refresh", result.Attributes)
	}
}

func TestFailedCommand(t *testing.T) {
	attrs := []Attribute{Attr("command", "refresh")}

	result := FailedCommand("failed", errors.New("boom"), 150*time.Millisecond, attrs...)
	attrs[0] = Attr("command", "mutated")

	if result.State != StateFailed {
		t.Fatalf("State = %q, want %q", result.State, StateFailed)
	}
	if !result.Accepted {
		t.Fatal("Accepted = false, want true")
	}
	if result.Message != "failed" {
		t.Fatalf("Message = %q, want failed", result.Message)
	}
	if result.Error != "boom" {
		t.Fatalf("Error = %q, want boom", result.Error)
	}
	if result.StartedAt == nil {
		t.Fatal("StartedAt is nil")
	}
	if result.FinishedAt == nil {
		t.Fatal("FinishedAt is nil")
	}
	if result.Duration.TimeDuration() != 150*time.Millisecond {
		t.Fatalf("Duration = %v, want 150ms", result.Duration.TimeDuration())
	}
	if len(result.Attributes) != 1 || result.Attributes[0] != Attr("command", "refresh") {
		t.Fatalf("Attributes = %+v, want command refresh", result.Attributes)
	}
}

func TestFailedCommandWithNilError(t *testing.T) {
	result := FailedCommand("failed", nil, 0)

	if result.State != StateFailed {
		t.Fatalf("State = %q, want %q", result.State, StateFailed)
	}
	if result.Error != "" {
		t.Fatalf("Error = %q, want empty", result.Error)
	}
}

func TestCommandResultJSONOmitEmptyFields(t *testing.T) {
	data, err := json.Marshal(CommandResult{
		State:    StateNotReady,
		Accepted: false,
	})
	if err != nil {
		t.Fatalf("Marshal CommandResult error = %v", err)
	}

	want := `{"state":"not_ready","accepted":false}`
	if string(data) != want {
		t.Fatalf("Marshal CommandResult = %s, want %s", data, want)
	}
}

func TestCommandResultJSONIncludesResult(t *testing.T) {
	result := CommandResult{
		State:      StateReady,
		Accepted:   true,
		Message:    "completed",
		Result:     map[string]any{"refreshed": true},
		Attributes: []Attribute{Attr("command", "refresh")},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal CommandResult error = %v", err)
	}

	want := `{"state":"ready","accepted":true,"message":"completed","result":{"refreshed":true},"attributes":[{"key":"command","value":"refresh"}]}`
	if string(data) != want {
		t.Fatalf("Marshal CommandResult = %s, want %s", data, want)
	}
}
