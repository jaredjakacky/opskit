package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	opskit "github.com/jaredjakacky/opskit"
)

type dependencyChecks struct{}

func (dependencyChecks) ComponentInfo() opskit.ComponentInfo {
	return opskit.ComponentInfo{
		Name: "dependencies",
		Kind: "dependency_group",
	}
}

func (dependencyChecks) Status(context.Context) opskit.Status {
	return opskit.DegradedStatus("one dependency check is failing")
}

func (dependencyChecks) Checks(context.Context) []opskit.CheckDescriptor {
	return []opskit.CheckDescriptor{
		{
			Name:        "database",
			Kind:        "dependency",
			Description: "ping database",
			Attributes: []opskit.Attribute{
				opskit.Attr("target", "database"),
			},
		},
		{
			Name:        "cache",
			Kind:        "dependency",
			Description: "ping primary cache",
			Attributes: []opskit.Attribute{
				opskit.Attr("target", "cache"),
			},
		},
	}
}

func (dependencyChecks) Check(context.Context) opskit.CheckResult {
	return opskit.FailedCheck(
		"primary cache ping failed",
		errors.New("timeout after 50ms"),
		50*time.Millisecond,
		opskit.Attr("target", "cache"),
	)
}

func (dependencyChecks) CheckAll(ctx context.Context) opskit.CheckSummary {
	// Pretend the group started before the delegated check so the summary duration
	// covers all named checks in this deterministic example.
	started := time.Now().Add(-64 * time.Millisecond)
	results := []opskit.NamedCheck{
		{
			Name:   "database",
			Kind:   "dependency",
			Result: opskit.ReadyCheck("database reachable", 12*time.Millisecond),
		},
		{
			Name:   "cache",
			Kind:   "dependency",
			Result: dependencyChecks{}.Check(ctx),
		},
	}

	return opskit.SummarizeChecks("", started, results)
}

func main() {
	ctx := context.Background()
	registry := opskit.NewRegistry()
	registry.MustRegister(dependencyChecks{}, opskit.Required())

	checker, err := registry.Checker("dependencies")
	if err != nil {
		log.Fatal(err)
	}

	group, err := registry.CheckGroup("dependencies")
	if err != nil {
		log.Fatal(err)
	}

	checks, err := registry.Checks(ctx, "dependencies")
	if err != nil {
		log.Fatal(err)
	}

	// The registry discovers the capability. The caller decides when to execute it.
	fmt.Println("described checks")
	printJSON(checks)

	fmt.Println("single check")
	printJSON(checker.Check(ctx))

	fmt.Println("check group")
	printJSON(group.CheckAll(ctx))
}

func printJSON(value any) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}
