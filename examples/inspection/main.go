package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	opskit "github.com/jaredjakacky/opskit"
)

type cacheComponent struct{}

func (cacheComponent) ComponentInfo() opskit.ComponentInfo {
	return opskit.ComponentInfo{
		Name:        "cache",
		Kind:        "dependency",
		Description: "primary application cache",
	}
}

func (cacheComponent) Status(context.Context) opskit.Status {
	return opskit.DegradedStatus("cache is serving with elevated latency",
		opskit.Attr("mode", "write-through"),
	)
}

// Inspection data may be exposed through admin/debug surfaces. Do not include
// secrets or unredacted sensitive data.
func (cacheComponent) Inspect(context.Context) (opskit.Inspection, error) {
	return opskit.Inspection{
		Summary: "cache online with slow primary shard",
		Details: map[string]any{
			"entries":       4217,
			"primary_shard": "slow",
			"replicas":      []string{"cache-a", "cache-b"},
		},
		Attributes: []opskit.Attribute{
			opskit.Attr("shard", "primary"),
			opskit.Attr("safe", "redacted"),
		},
	}, nil
}

func main() {
	ctx := context.Background()
	registry := opskit.NewRegistry()
	registry.MustRegister(cacheComponent{}, opskit.Required())

	snapshot, err := registry.Snapshot(ctx, "cache")
	if err != nil {
		log.Fatal(err)
	}

	printJSON(snapshot)
}

func printJSON(value any) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}
