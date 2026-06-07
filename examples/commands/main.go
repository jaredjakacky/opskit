package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	opskit "github.com/jaredjakacky/opskit"
)

type cacheAdmin struct{}

func (cacheAdmin) ComponentInfo() opskit.ComponentInfo {
	return opskit.ComponentInfo{
		Name: "cache-admin",
		Kind: "command_handler",
	}
}

func (cacheAdmin) Status(context.Context) opskit.Status {
	return opskit.ReadyStatus("cache command handler ready")
}

func (cacheAdmin) HandleCommand(_ context.Context, request opskit.CommandRequest) opskit.CommandResult {
	if request.Name != "cache/refresh" {
		return opskit.RejectedCommand("unsupported command")
	}

	var payload struct {
		Force bool `json:"force"`
	}
	if len(request.Payload) > 0 {
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			return opskit.RejectedCommand("invalid command payload")
		}
	}

	return opskit.CompletedCommand(
		"cache refresh completed",
		map[string]any{"refreshed": true, "force": payload.Force},
		18*time.Millisecond,
		opskit.Attr("command", "cache/refresh"),
	)
}

func main() {
	ctx := context.Background()
	registry := opskit.NewRegistry()
	registry.MustRegister(cacheAdmin{}, opskit.Informational())

	handler, err := registry.CommandHandler("cache-admin")
	if err != nil {
		log.Fatal(err)
	}

	request := opskit.CommandRequest{
		Name:    "cache/refresh",
		Payload: json.RawMessage(`{"force":true}`),
	}

	printJSON(handler.HandleCommand(ctx, request))
}

func printJSON(value any) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}
