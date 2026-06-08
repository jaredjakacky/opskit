# Getting Started

Use this guide for the shortest path from `go get` to a useful Opskit registry.
It stays on the normal Opskit path:

1. install the package
2. create a registry
3. register one component
4. read status and readiness
5. move active work into checks or commands instead of `Status`

## Install

```bash
go get github.com/jaredjakacky/opskit
```

Opskit's minimum supported Go version is declared in [`go.mod`](../go.mod). The
Go versions currently verified in CI are listed in
[`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml).

## Build a first registry

This example keeps the component simple so the registry contract stays easy to
see.

```go
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
	ops := opskit.NewRegistry()

	ops.MustRegister(opskit.ComponentFunc{
		Info: opskit.ComponentInfo{
			Name: "config",
			Kind: "config",
		},
		Fn: func(context.Context) opskit.Status {
			return opskit.ReadyStatus("configuration loaded",
				opskit.Attr("source", "file"),
			)
		},
	}, opskit.Required())

	readiness := ops.Readiness(ctx)

	out, err := json.MarshalIndent(readiness, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(out))
}
```

The component returns a fast local status snapshot. The registry turns that into
aggregate readiness because the component is registered as required.

## Run it

From your own module, run your main package, for example:

```bash
go run ./cmd/your-service
```

If you are just exploring locally in this repository, run the existing example
instead:

```bash
go run ./examples/basic
```

## Expected output

The exact timestamps are omitted from readiness output, so the result stays
small:

```json
{
  "ready": true,
  "reason": "all readiness components ready",
  "components": [
    {
      "name": "config",
      "kind": "config",
      "policy": "required",
      "ready": true,
      "state": "ready",
      "message": "configuration loaded"
    }
  ]
}
```

## What You Get From `NewRegistry`

A fresh registry gives you:

- registration of named operational components
- required, optional, and informational readiness policy
- aggregate status and readiness read models
- capability discovery for inspection, checks, check groups, and commands
- component snapshots for admin presentation
- a zero-value-safe registry for embedded structs and tests

The registry is passive. It does not serve HTTP, run checks, dispatch commands,
schedule background work, authorize admin calls, or export telemetry.

## The First Rule

Keep `Status(context.Context)` cheap.

Good status work:

- return cached readiness or lifecycle state
- include safe attributes
- report the last known local state
- explain why the component is degraded or not ready

Work that belongs somewhere else:

- dependency pings
- retries
- configuration reloads
- command dispatch
- background scheduling
- HTTP admin presentation
- authorization

Use `Checker`, `CheckGroup`, `CommandHandler`, Workerkit, Servekit, or
application-owned code for those responsibilities.

## Next Steps

- Read [Usage Guide](usage.md) for registry usage, readiness policy, inspection,
  checks, and commands.
- Read [Operational Safety](operational-safety.md) before exposing inspection,
  check errors, or command payloads.
- Read [Composition Guide](composition.md) for how Opskit is intended to fit
  with Servekit, Workerkit, and the rest of the Kit Series.
- Read [API Map](api.md) for the complete exported surface.
- Run the examples listed in [Opskit Examples](../examples/README.md).
