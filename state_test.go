package opskit

import (
	"encoding/json"
	"testing"
)

func TestStateValues(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  string
	}{
		{name: "unknown", state: StateUnknown, want: "unknown"},
		{name: "initializing", state: StateInitializing, want: "initializing"},
		{name: "ready", state: StateReady, want: "ready"},
		{name: "degraded", state: StateDegraded, want: "degraded"},
		{name: "not ready", state: StateNotReady, want: "not_ready"},
		{name: "failed", state: StateFailed, want: "failed"},
		{name: "stopped", state: StateStopped, want: "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.want {
				t.Fatalf("State value = %q, want %q", tt.state, tt.want)
			}
		})
	}
}

func TestStateJSON(t *testing.T) {
	requireJSON(t, StateNotReady, `"not_ready"`)

	var state State
	if err := json.Unmarshal([]byte(`"not_ready"`), &state); err != nil {
		t.Fatalf("Unmarshal State error = %v", err)
	}

	if state != StateNotReady {
		t.Fatalf("Unmarshal State = %q, want %q", state, StateNotReady)
	}
}
