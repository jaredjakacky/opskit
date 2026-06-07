package opskit

import (
	"testing"
	"time"
)

func TestStatusConstructors(t *testing.T) {
	tests := []struct {
		name      string
		status    Status
		wantState State
		wantReady bool
	}{
		{
			name:      "ready",
			status:    ReadyStatus("ready"),
			wantState: StateReady,
			wantReady: true,
		},
		{
			name:      "degraded",
			status:    DegradedStatus("degraded"),
			wantState: StateDegraded,
			wantReady: true,
		},
		{
			name:      "not ready",
			status:    NotReadyStatus("not ready"),
			wantState: StateNotReady,
			wantReady: false,
		},
		{
			name:      "failed",
			status:    FailedStatus("failed"),
			wantState: StateFailed,
			wantReady: false,
		},
		{
			name:      "unknown",
			status:    UnknownStatus("unknown"),
			wantState: StateUnknown,
			wantReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.State != tt.wantState {
				t.Fatalf("State = %q, want %q", tt.status.State, tt.wantState)
			}

			if tt.status.Ready != tt.wantReady {
				t.Fatalf("Ready = %t, want %t", tt.status.Ready, tt.wantReady)
			}

			if tt.status.Message != tt.name {
				t.Fatalf("Message = %q, want %q", tt.status.Message, tt.name)
			}

			if tt.status.UpdatedAt == nil {
				t.Fatal("UpdatedAt is nil")
			}

			if tt.status.UpdatedAt.Location() != time.UTC {
				t.Fatalf("UpdatedAt location = %q, want UTC", tt.status.UpdatedAt.Location())
			}

			if len(tt.status.Attributes) != 0 {
				t.Fatalf("Attributes length = %d, want 0", len(tt.status.Attributes))
			}
		})
	}
}

func TestStatusConstructorsCloneAttributes(t *testing.T) {
	constructors := map[string]func(string, ...Attribute) Status{
		"ready":     ReadyStatus,
		"degraded":  DegradedStatus,
		"not_ready": NotReadyStatus,
		"failed":    FailedStatus,
		"unknown":   UnknownStatus,
	}

	for name, constructor := range constructors {
		t.Run(name, func(t *testing.T) {
			attrs := []Attribute{
				Attr("component", "cache"),
				Attr("shard", "primary"),
			}

			status := constructor("status", attrs...)
			attrs[0] = Attr("component", "mutated")

			if len(status.Attributes) != 2 {
				t.Fatalf("Attributes length = %d, want 2", len(status.Attributes))
			}

			if status.Attributes[0] != Attr("component", "cache") {
				t.Fatalf("Attributes[0] = %+v, want original attribute", status.Attributes[0])
			}

			if status.Attributes[1] != Attr("shard", "primary") {
				t.Fatalf("Attributes[1] = %+v, want original attribute", status.Attributes[1])
			}
		})
	}
}

func TestStatusJSONOmitEmptyFields(t *testing.T) {
	requireJSON(t, Status{
		State: StateReady,
		Ready: true,
	}, `{"state":"ready","ready":true}`)
}

func TestComponentStatusJSONIncludesOperationalMetadata(t *testing.T) {
	updatedAt := time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC)

	component := ComponentStatus{
		Component: ComponentInfo{
			Name: "cache",
			Kind: "dependency",
		},
		Registration: ComponentRegistration{
			ReadinessPolicy: ReadinessOptional,
		},
		Capabilities: ComponentCapabilities{
			Checker:        true,
			CommandHandler: true,
		},
		Status: Status{
			State:     StateDegraded,
			Ready:     true,
			Message:   "cache slow",
			UpdatedAt: &updatedAt,
			Attributes: []Attribute{
				Attr("shard", "primary"),
			},
		},
	}

	requireJSON(t, component, `{"component":{"name":"cache","kind":"dependency"},"registration":{"readiness_policy":"optional"},"capabilities":{"checker":true,"command_handler":true},"status":{"state":"degraded","ready":true,"message":"cache slow","updated_at":"2026-06-04T12:30:00Z","attributes":[{"key":"shard","value":"primary"}]}}`)
}

func TestSystemStatusJSONOmitEmptyComponents(t *testing.T) {
	requireJSON(t, SystemStatus{}, `{}`)
}
