package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	opskit "github.com/jaredjakacky/opskit"
)

func main() {
	ctx := context.Background()
	registry := opskit.NewRegistry()

	registry.MustRegister(statusComponent(
		"database",
		"dependency",
		opskit.ReadyStatus("database reachable"),
	), opskit.Required())

	registry.MustRegister(statusComponent(
		"search",
		"client",
		opskit.NotReadyStatus("search enrichment unavailable"),
	), opskit.Optional())

	registry.MustRegister(statusComponent(
		"build",
		"metadata",
		opskit.NotReadyStatus("build metadata missing"),
	), opskit.Informational())

	fmt.Println("readiness")
	printJSON(registry.Readiness(ctx))

	fmt.Println("status")
	printJSON(registry.Status(ctx))
}

func statusComponent(name, kind string, status opskit.Status) opskit.Component {
	return opskit.ComponentFunc{
		Info: opskit.ComponentInfo{Name: name, Kind: kind},
		Fn: func(context.Context) opskit.Status {
			return status
		},
	}
}

func printJSON(value any) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}
