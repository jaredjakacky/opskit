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

	registry.MustRegister(opskit.ComponentFunc{
		Info: opskit.ComponentInfo{
			Name:        "config",
			Kind:        "config",
			Description: "application configuration",
		},
		Fn: func(context.Context) opskit.Status {
			return opskit.ReadyStatus("configuration loaded",
				opskit.Attr("source", "file"),
			)
		},
	}, opskit.Required())

	registry.MustRegister(opskit.ComponentFunc{
		Info: opskit.ComponentInfo{
			Name:        "search",
			Kind:        "client",
			Description: "optional search enrichment client",
		},
		Fn: func(context.Context) opskit.Status {
			return opskit.NotReadyStatus("search API unavailable")
		},
	}, opskit.Optional())

	printJSON(registry.Readiness(ctx))
}

func printJSON(value any) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}
