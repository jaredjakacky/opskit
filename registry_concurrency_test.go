package opskit

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestRegistryConcurrentRegisterAndReadModels(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	const components = 64
	const readers = 8

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < components; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			component := testComponent{
				info: ComponentInfo{
					Name: fmt.Sprintf("component-%02d", i),
					Kind: "test",
				},
				status: ReadyStatus("ready"),
			}
			if err := registry.Register(component); err != nil {
				t.Errorf("Register(%s) error = %v", component.info.Name, err)
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < components; j++ {
				_ = registry.Components()
				_ = registry.Status(ctx)
				_ = registry.Readiness(ctx)
			}
		}()
	}

	close(start)
	wg.Wait()

	registered := registry.Components()
	if len(registered) != components {
		t.Fatalf("Components length = %d, want %d", len(registered), components)
	}

	status := registry.Status(ctx)
	if len(status.Components) != components {
		t.Fatalf("Status.Components length = %d, want %d", len(status.Components), components)
	}

	readiness := registry.Readiness(ctx)
	if !readiness.Ready {
		t.Fatalf("Readiness.Ready = false, reason = %q", readiness.Reason)
	}
	if len(readiness.Components) != components {
		t.Fatalf("Readiness.Components length = %d, want %d", len(readiness.Components), components)
	}

	seen := make(map[string]bool, components)
	for _, component := range status.Components {
		if component.Component.Name == "" {
			t.Fatal("Status component name is empty")
		}
		if seen[component.Component.Name] {
			t.Fatalf("duplicate status component name %q", component.Component.Name)
		}
		seen[component.Component.Name] = true
	}
}

func TestRegistryConcurrentSnapshotAndCapabilityAccessors(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	component := &testOperationalComponent{
		info:       ComponentInfo{Name: "operational", Kind: "test"},
		status:     ReadyStatus("ready"),
		readiness:  ReadyReadiness("ready"),
		inspection: Inspection{Summary: "ok"},
	}
	if err := registry.Register(component); err != nil {
		t.Fatalf("Register(operational) error = %v", err)
	}

	const readers = 16
	const iterations = 64

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < iterations; j++ {
				if _, err := registry.Snapshot(ctx, "operational"); err != nil {
					t.Errorf("Snapshot error = %v", err)
				}
				if _, err := registry.Inspect(ctx, "operational"); err != nil {
					t.Errorf("Inspect error = %v", err)
				}
				if _, err := registry.Checker("operational"); err != nil {
					t.Errorf("Checker error = %v", err)
				}
				if _, err := registry.CheckDescriber("operational"); err != nil {
					t.Errorf("CheckDescriber error = %v", err)
				}
				if _, err := registry.Checks(ctx, "operational"); err != nil {
					t.Errorf("Checks error = %v", err)
				}
				if _, err := registry.CheckGroup("operational"); err != nil {
					t.Errorf("CheckGroup error = %v", err)
				}
				if _, err := registry.CommandHandler("operational"); err != nil {
					t.Errorf("CommandHandler error = %v", err)
				}
				if _, err := registry.CommandDescriber("operational"); err != nil {
					t.Errorf("CommandDescriber error = %v", err)
				}
				if _, err := registry.Commands(ctx, "operational"); err != nil {
					t.Errorf("Commands error = %v", err)
				}
			}
		}()
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			component := testComponent{
				info: ComponentInfo{
					Name: fmt.Sprintf("plain-%02d", i),
					Kind: "test",
				},
				status: ReadyStatus("ready"),
			}
			if err := registry.Register(component); err != nil {
				t.Errorf("Register(%s) error = %v", component.info.Name, err)
			}
		}(i)
	}

	close(start)
	wg.Wait()
}
