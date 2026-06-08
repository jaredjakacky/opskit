package opskit

import (
	"context"
	"encoding/json"
	"testing"
)

func TestComponentInfoJSONOmitEmptyFields(t *testing.T) {
	info := ComponentInfo{Name: "cache"}

	requireJSON(t, info, `{"name":"cache"}`)
}

func TestComponentInfoJSONIncludesAllFields(t *testing.T) {
	info := ComponentInfo{
		Name:        "cache",
		Kind:        "dependency",
		Description: "primary cache",
	}

	requireJSON(t, info, `{"name":"cache","kind":"dependency","description":"primary cache"}`)
}

func TestReadinessPolicyValues(t *testing.T) {
	tests := []struct {
		name   string
		policy ReadinessPolicy
		want   string
	}{
		{name: "required", policy: ReadinessRequired, want: "required"},
		{name: "optional", policy: ReadinessOptional, want: "optional"},
		{name: "informational", policy: ReadinessInformational, want: "informational"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.policy) != tt.want {
				t.Fatalf("ReadinessPolicy = %q, want %q", tt.policy, tt.want)
			}
		})
	}
}

func TestComponentRegistrationJSON(t *testing.T) {
	registration := ComponentRegistration{
		ReadinessPolicy: ReadinessOptional,
	}

	requireJSON(t, registration, `{"readiness_policy":"optional"}`)
}

func TestComponentCapabilitiesJSONOmitEmptyFields(t *testing.T) {
	requireJSON(t, ComponentCapabilities{}, `{}`)
}

func TestComponentCapabilitiesJSONIncludesSupportedCapabilities(t *testing.T) {
	capabilities := ComponentCapabilities{
		ReadinessContributor: true,
		Inspector:            true,
		Checker:              true,
		CheckDescriber:       true,
		CheckGroup:           true,
		CommandHandler:       true,
		CommandDescriber:     true,
	}

	requireJSON(t, capabilities, `{"readiness_contributor":true,"inspector":true,"checker":true,"check_describer":true,"check_group":true,"command_handler":true,"command_describer":true}`)
}

func TestComponentEntryJSON(t *testing.T) {
	entry := ComponentEntry{
		Component: ComponentInfo{
			Name: "cache",
			Kind: "dependency",
		},
		Registration: ComponentRegistration{
			ReadinessPolicy: ReadinessOptional,
		},
		Capabilities: ComponentCapabilities{
			Checker: true,
		},
	}

	requireJSON(t, entry, `{"component":{"name":"cache","kind":"dependency"},"registration":{"readiness_policy":"optional"},"capabilities":{"checker":true}}`)
}

func TestComponentSnapshotJSONOmitsPointerViews(t *testing.T) {
	snapshot := ComponentSnapshot{
		Component: ComponentInfo{
			Name: "cache",
		},
		Registration: ComponentRegistration{
			ReadinessPolicy: ReadinessRequired,
		},
		Status: ReadyStatus("ready"),
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal ComponentSnapshot error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal ComponentSnapshot error = %v", err)
	}

	if _, ok := got["component"]; !ok {
		t.Fatal("component field missing")
	}
	if _, ok := got["registration"]; !ok {
		t.Fatal("registration field missing")
	}
	if _, ok := got["status"]; !ok {
		t.Fatal("status field missing")
	}
	capabilities, ok := got["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities field = %T, want object", got["capabilities"])
	}
	if len(capabilities) != 0 {
		t.Fatalf("capabilities = %+v, want empty object", capabilities)
	}
	if _, ok := got["readiness"]; ok {
		t.Fatal("readiness field present, want omitted")
	}
	if _, ok := got["inspection"]; ok {
		t.Fatal("inspection field present, want omitted")
	}
	if _, ok := got["inspection_error"]; ok {
		t.Fatal("inspection_error field present, want omitted")
	}
}

func TestComponentSnapshotJSONIncludesOptionalViews(t *testing.T) {
	readiness := ReadyReadiness("ready")
	inspection := Inspection{Summary: "ok"}
	snapshot := ComponentSnapshot{
		Component: ComponentInfo{
			Name: "cache",
			Kind: "dependency",
		},
		Registration: ComponentRegistration{
			ReadinessPolicy: ReadinessOptional,
		},
		Capabilities: ComponentCapabilities{
			ReadinessContributor: true,
			Inspector:            true,
		},
		Status:     ReadyStatus("ready"),
		Readiness:  &readiness,
		Inspection: &inspection,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal ComponentSnapshot error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal ComponentSnapshot error = %v", err)
	}
	if _, ok := got["capabilities"]; !ok {
		t.Fatal("capabilities field missing")
	}
	if _, ok := got["readiness"]; !ok {
		t.Fatal("readiness field missing")
	}
	if _, ok := got["inspection"]; !ok {
		t.Fatal("inspection field missing")
	}
}

func TestComponentSnapshotJSONIncludesInspectionError(t *testing.T) {
	snapshot := ComponentSnapshot{
		Component: ComponentInfo{
			Name: "cache",
		},
		Registration: ComponentRegistration{
			ReadinessPolicy: ReadinessRequired,
		},
		Status:          ReadyStatus("ready"),
		InspectionError: "inspection failed",
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal ComponentSnapshot error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal ComponentSnapshot error = %v", err)
	}
	if got["inspection_error"] != "inspection failed" {
		t.Fatalf("inspection_error = %q, want inspection failed", got["inspection_error"])
	}
}

func TestComponentFuncComponentInfo(t *testing.T) {
	component := ComponentFunc{
		Info: ComponentInfo{
			Name: "cache",
			Kind: "dependency",
		},
	}

	info := component.ComponentInfo()
	if info.Name != "cache" {
		t.Fatalf("Name = %q, want cache", info.Name)
	}
	if info.Kind != "dependency" {
		t.Fatalf("Kind = %q, want dependency", info.Kind)
	}
}

func TestComponentFuncStatus(t *testing.T) {
	ctx := context.Background()
	component := ComponentFunc{
		Info: ComponentInfo{Name: "cache"},
		Fn: func(got context.Context) Status {
			if got != ctx {
				t.Fatal("context was not passed through")
			}
			return ReadyStatus("ready")
		},
	}

	status := component.Status(ctx)
	if status.State != StateReady {
		t.Fatalf("State = %q, want %q", status.State, StateReady)
	}
	if !status.Ready {
		t.Fatal("Ready = false, want true")
	}
}

func TestComponentFuncStatusNormalizesNilContext(t *testing.T) {
	var ctx context.Context
	component := ComponentFunc{
		Info: ComponentInfo{Name: "cache"},
		Fn: func(ctx context.Context) Status {
			if ctx == nil {
				t.Fatal("context is nil, want normalized context")
			}
			return ReadyStatus("ready")
		},
	}

	component.Status(ctx)
}

func TestComponentFuncStatusWithNilFunc(t *testing.T) {
	component := ComponentFunc{
		Info: ComponentInfo{Name: "cache"},
	}

	status := component.Status(context.Background())
	if status.State != StateUnknown {
		t.Fatalf("State = %q, want %q", status.State, StateUnknown)
	}
	if status.Ready {
		t.Fatal("Ready = true, want false")
	}
	if status.Message != "component status function is not configured" {
		t.Fatalf("Message = %q, want component status function is not configured", status.Message)
	}
}
