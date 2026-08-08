# Opskit

[![Release](https://img.shields.io/github/v/release/jaredjakacky/opskit?sort=semver)](https://github.com/jaredjakacky/opskit/releases)
[![CI](https://github.com/jaredjakacky/opskit/actions/workflows/ci.yaml/badge.svg)](https://github.com/jaredjakacky/opskit/actions/workflows/ci.yaml)
[![Go Support](https://img.shields.io/badge/go%20support-1.25.x%20%7C%201.26.x-00ADD8)](https://github.com/jaredjakacky/opskit/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/jaredjakacky/opskit)](https://github.com/jaredjakacky/opskit/blob/main/LICENSE)

## Overview

Opskit is a small Go package for giving service components one shared operational language.

It defines the contracts and data shapes that production services keep rebuilding around status, readiness, inspection, checks, commands, and safe operational metadata. Components expose what they know. A registry collects them. Presentation and execution layers decide what to do with that state.

Opskit is especially useful for services that want a coherent operations surface without turning the application into a framework.

Opskit is not a runtime. It does not serve HTTP, start goroutines, run health checks, dispatch commands, schedule work, load configuration, build clients, export telemetry, authorize admin calls, or own application lifecycle. It gives those systems a common contract.

In the Kit Series, Opskit is the operational spine:

- [Servekit](https://github.com/jaredjakacky/servekit) presents service operations over HTTP.
- [Workerkit](https://github.com/jaredjakacky/workerkit) executes background work, checks, and commands.
- [Configkit](https://github.com/jaredjakacky/configkit) exposes configuration lifecycle state.
- [Dependkit](https://github.com/jaredjakacky/dependkit) exposes dependency health state.
- Clientkit exposes outbound client state.
- Statekit exposes application state lifecycle and inspection.
- Opskit gives all of them one vocabulary without forcing them to import each other.

## Why Opskit exists

Production services usually have several operational domains:

- configuration
- workers
- clients
- dependencies
- state
- build metadata
- internal admin actions
- service readiness
- diagnostic inspection

Each domain can be implemented cleanly on its own. The problem appears at the service boundary.

Operators do not want five unrelated status formats. Tests do not want five different readiness models. HTTP admin endpoints should not need a custom adapter for every package. Worker runtimes should not need to know the internal details of every component that can be checked or commanded.

Without a shared operational contract, integration drifts into pairwise glue:

```text
config -> servekit
config -> workerkit
dependencies -> servekit
dependencies -> workerkit
clients -> servekit
clients -> workerkit
state -> servekit
state -> workerkit
```

That glue is not domain logic. It is operational plumbing.

Opskit pulls the common language into one small package. A component can say:

- who it is
- what state it is in
- whether it is ready
- what safe details can be inspected
- whether it supports active checks
- whether it supports grouped checks
- whether it supports operational commands
- what safe attributes it exposes

The application still owns policy. Opskit only gives the policy a stable shape.

## Who uses Opskit directly?

You can use Opskit without any other Kit Series package.

Most applications in the Kit Series will eventually use Opskit indirectly
through sibling packages, but Opskit is also useful on its own.

Use Opskit directly when your application has custom operational components that should participate in the same service-level status, readiness, inspection, check, or command model as the rest of the system.

Opskit is most useful at the service assembly boundary: the place where application code wires together configuration, workers, clients, dependencies, state, build metadata, and operational presentation.

For example, an application can register its own components next to Kit Series components:

```go
ops := opskit.NewRegistry()

ops.MustRegister(configComponent, opskit.Required())
ops.MustRegister(workerRuntimeComponent, opskit.Required())
ops.MustRegister(clientComponent, opskit.Required())
ops.MustRegister(dependencyComponent, opskit.Required())
ops.MustRegister(stateComponent, opskit.Required())
ops.MustRegister(buildInfoComponent, opskit.Informational())
ops.MustRegister(myCustomComponent, opskit.Optional())
```

Then presentation layers, tests, CLIs, and application code can consume the same
read models without knowing each component's implementation details. Execution
layers receive explicitly selected active capabilities rather than treating
registration as permission to execute them.

## Using Opskit Standalone

Standalone Opskit is useful when your application already has its own HTTP
server, CLI, worker runtime, platform integration, or admin surface, but you
want one consistent model for operational state.

Common standalone uses include:

- backing an existing `/readyz` endpoint with `Registry.Readiness`
- powering CLI commands such as `status` or `inspect`
- giving tests one readiness model instead of ad hoc booleans
- exposing safe admin snapshots through an existing admin surface
- standardizing component identity, status, readiness, inspection, checks,
  commands, and attributes without adopting the rest of the Kit Series

Opskit still stays passive in standalone usage. It does not serve HTTP,
schedule checks, dispatch commands, authorize callers, export telemetry, retry
work, or manage lifecycle. Your application decides where registry data is
presented and when active capabilities are invoked.

## What Opskit is not

Opskit is not the service runtime. It does not own HTTP routing, worker loops,
configuration loading, dependency execution, outbound client construction, state
management, telemetry exporting, dashboards, dependency injection, scheduling,
authorization, or application lifecycle.

Opskit does not execute checks or commands. `Checker`, `CheckGroup`, and
`CommandHandler` are contracts for active capabilities; Opskit's role is limited
to defining and discovering them. A runtime such as Workerkit decides when and
how to execute explicitly supplied capabilities.

## Good fit / not a fit

Opskit is a good fit when:

- you want one common operational model across several service components
- you want status, readiness, and inspection to use ordinary Go values
- you want components to compose without pairwise package adapters
- you want readiness aggregation without coupling every component to HTTP
- you want active checks and commands to be discoverable without making Opskit execute them
- you want Servekit, Workerkit, Configkit, Dependkit, Clientkit, Statekit, and application-owned components to meet at one small boundary
- you want a shared operational vocabulary without adopting a full framework

Opskit is probably not a fit when:

- one boolean readiness flag is enough
- your service already has a settled operational contract
- you want a package that runs background checks for you
- you want built-in HTTP routes, dashboards, metrics exporters, or alerting
- you want lifecycle, scheduling, command authorization, retries, or admin presentation in the same package
- you want dependency injection or application hosting

## Installation

```bash
go get github.com/jaredjakacky/opskit
```

```go
import opskit "github.com/jaredjakacky/opskit"
```

## Quick Start

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

	ops.MustRegister(opskit.ComponentFunc{
		Info: opskit.ComponentInfo{
			Name:        "search",
			Kind:        "client",
			Description: "optional search enrichment client",
		},
		Fn: func(context.Context) opskit.Status {
			return opskit.NotReadyStatus("search API unavailable")
		},
	}, opskit.Optional())

	readiness := ops.Readiness(ctx)

	out, err := json.MarshalIndent(readiness, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(out))
}
```

The aggregate readiness remains ready because `config` is required and ready, while `search` is optional and non-blocking:

```json
{
  "ready": true,
  "reason": "all required readiness components ready",
  "components": [
    {
      "component": {
        "name": "config",
        "kind": "config",
        "description": "application configuration"
      },
      "registration": {
        "readiness_policy": "required"
      },
      "readiness": {
        "ready": true,
        "reason": "component ready",
        "items": [
          {
            "name": "config",
            "kind": "config",
            "impact": "blocking",
            "ready": true,
            "state": "ready",
            "message": "configuration loaded"
          }
        ]
      }
    },
    {
      "component": {
        "name": "search",
        "kind": "client",
        "description": "optional search enrichment client"
      },
      "registration": {
        "readiness_policy": "optional"
      },
      "readiness": {
        "ready": false,
        "reason": "component not ready",
        "items": [
          {
            "name": "search",
            "kind": "client",
            "impact": "blocking",
            "ready": false,
            "state": "not_ready",
            "message": "search API unavailable"
          }
        ]
      }
    }
  ]
}
```

The two levels are intentional: `registration.readiness_policy` says whether
the registered parent blocks service readiness, while child `impact` belongs to
the component's own readiness domain.

That one registry already gives you:

- stable component identity
- component status snapshots
- aggregate readiness
- required, optional, and informational readiness policy
- capability discovery
- safe inspection hooks
- contracts and passive discovery for active checks, check groups, and commands
- one read model for HTTP, workers, tests, CLIs, logs, and diagnostics

In practice, you get a shared operations surface without building a new adapter contract for every package combination.

## The Core Model

Opskit is deliberately built around passive read models, explicit capability
contracts, and an explicit registry.

### Component

`Component` is the minimum contract for registered operational state:

```go
type Component interface {
	ComponentInfo() ComponentInfo
	Status(context.Context) Status
}
```

`ComponentInfo` gives the component a stable operational identity. Names must be unique within a registry and safe for use in path-oriented operations surfaces. Kinds, labels, and attribute keys should be stable, low-cardinality safe tokens, but Opskit leaves stricter validation to presentation and telemetry layers. Labels are stable identity metadata for passive inventory and filtering; attributes on status, inspection, checks, and commands carry runtime or result-specific metadata.

`Status` reports the current component state. It should normally be a fast cached or local snapshot. Expensive work such as dependency pings, reloads, active checks, command dispatch, remote calls, or state transitions belongs in explicit check, command, worker, or application execution paths.

### Readiness

Status answers: what state is this component in?

Readiness answers: should this component allow the service to receive work?

By default, readiness can be derived from `Status.Ready`. Components with richer admission rules can implement `ReadinessContributor`:

```go
type ReadinessContributor interface {
	Readiness(context.Context) Readiness
}
```

The registry also has registration-level readiness policy:

```go
ops.MustRegister(configManager, opskit.Required())
ops.MustRegister(searchClient, opskit.Optional())
ops.MustRegister(buildInfo, opskit.Informational())
```

Required components block aggregate readiness when not ready. Optional components appear in readiness details but do not block. Informational components are omitted from readiness and remain visible through status and admin snapshots.

### Inspection

`Inspector` exposes safe diagnostic data beyond basic status:

```go
type Inspector interface {
	Inspect(context.Context) (Inspection, error)
}
```

Inspection data is intended for admin endpoints, diagnostics, support workflows, logs, and tests. Components are responsible for redacting inspection data before returning it. `Summary` and `Details` must be JSON-marshalable.

### Checks and check groups

`Checker` and `CheckGroup` describe active operational checks:

```go
type Checker interface {
	Check(context.Context) CheckResult
}

type CheckDescriber interface {
	Checks(context.Context) []CheckDescriptor
}

type CheckGroup interface {
	CheckAll(context.Context) CheckSummary
}
```

`CheckDescriber` lets admin surfaces, CLIs, worker runtimes, and docs
generators list supported checks without running them.

Opskit only defines the contracts and result shapes. It does not run checks on an interval, retry them, cache them, or decide when they should affect readiness.

A package such as Workerkit can execute checks under lifecycle, timeout, retry, jitter, and concurrency policy. A package such as Dependkit can expose dependency health through the same shape without making Opskit own dependency behavior.

### Commands

`CommandHandler` describes an operational command handler:

```go
type CommandHandler interface {
	HandleCommand(context.Context, CommandRequest) CommandResult
}

type CommandDescriber interface {
	Commands(context.Context) []CommandDescriptor
}
```

The handler method is intentionally named `HandleCommand` rather than `Command`
because it is the active operation on a handler, distinct from command metadata.

`CommandDescriber` lets admin surfaces, CLIs, worker runtimes, and docs
generators list supported commands without invoking them.

Command payloads are opaque JSON. Presentation layers own transport validation,
request limits, authentication, authorization, and selection of an allowed
command. The handler owns command-specific decoding and semantic validation.

Opskit does not dispatch commands. It does not authorize callers, apply concurrency limits, retry commands, or expose HTTP routes. Those responsibilities belong to execution and presentation layers such as Workerkit and Servekit.

### Registry

`Registry` stores components and provides common read methods:

```go
ops := opskit.NewRegistry()

ops.MustRegister(component, opskit.Required())

status := ops.Status(ctx)
readiness := ops.Readiness(ctx)
snapshot, err := ops.Snapshot(ctx, "component-name")
inspection, err := ops.Inspect(ctx, "component-name")
```

The registry is passive. It calls component read methods synchronously when asked. Probe and admin paths should pass bounded contexts. Registry cannot interrupt a component that ignores cancellation, but it rejects the component's result after the call returns if the context has expired.

The zero value is ready to use.

## Kit Series Composition

Opskit is not the application host. It is the shared operations contract used at service assembly boundaries.

The intended Kit Series shape is simple: domain kits report operational state
through Opskit contracts, Servekit presents that state, and Workerkit executes
active checks or commands under runtime policy.

The current Kit Series integrations preserve this boundary directly:

- Opskit records component state and discovers supported active capabilities.
- Servekit uses `WithOps` to present readiness, inventory, and read-only
  component snapshots.
- Workerkit executes explicitly supplied capabilities through `NewCheckLoop`,
  `NewCheckGroupLoop`, and `CommandFromOpskit`.
- Domain kits decide what their state means.
- Applications explicitly wire capabilities and decide policy.

Configkit managers, Dependkit registries, and Workerkit runtimes implement their
Opskit component contracts directly. Configkit and Dependkit expose command
handlers separately so registration remains distinct from execution. See the
[Composition Guide](docs/composition.md) for the current APIs and ownership
model.

## Why This Works

Opskit rests on three choices:

1. It keeps operational state as ordinary Go interfaces and structs.
2. It separates passive description from active execution.
3. It lets packages compose through shared contracts instead of importing each other.

That is why the package can stay small without becoming merely a bag of unrelated types.

Opskit gives the Kit Series a common operational language while preserving each kit's boundary:

- Servekit owns HTTP service bootstrap and presentation.
- Workerkit owns background execution, lifecycle, and command dispatch.
- Configkit owns typed configuration lifecycle.
- Dependkit owns generic dependency health.
- Clientkit owns outbound client behavior.
- Statekit owns state lifecycle and inspection.
- Applications own business policy.

## Advanced Capabilities

Opskit has a small core path, but it is not limited to status and readiness. Advanced hooks include:

- component capability discovery
- safe operational attributes
- custom readiness contributors
- safe inspection data
- active checker and check group contracts with passive capability discovery
- active command handler contracts with passive descriptor discovery
- JSON-friendly duration values
- registry-level required, optional, and informational readiness policy
- named component snapshots for admin presentation

These are contracts, not runtime behavior. Execution policy belongs outside Opskit.

## Documentation

- [Getting Started](docs/getting-started.md): first registry, first component, first readiness result
- [Usage Guide](docs/usage.md): components, status, readiness policy, inspection, and registry usage
- [Design Guide](docs/design.md): package boundaries, public type relationships, and registry flows
- [Composition Guide](docs/composition.md): how Opskit fits with Servekit, Workerkit, Configkit, Dependkit, Clientkit, and Statekit
- [Operational Safety](docs/operational-safety.md): safe attributes, inspection redaction, command payloads, and admin exposure
- [API Map](docs/api.md): human-friendly map of the exported surface
- [Examples Guide](docs/examples.md): how the runnable examples build from the core registry outward
- [Examples Directory](examples/README.md): quick index of runnable example programs

## Examples

Runnable standalone programs live in [`examples/`](examples), which build from
the smallest registry path outward. Current cross-kit examples are linked from
[`examples/README.md`](examples/README.md) and remain in the sibling repositories
that own their runtime and presentation behavior.

Recommended reading order:

1. [`examples/basic`](examples/basic)
2. [`examples/readiness-policies`](examples/readiness-policies)
3. [`examples/inspection`](examples/inspection)
4. [`examples/checks`](examples/checks)
5. [`examples/commands`](examples/commands)

For production-shaped composition, continue with the linked Servekit,
Workerkit, Configkit, and Dependkit examples after the standalone sequence.

## API Reference

The canonical symbol-level API documentation should live in Go doc comments so it stays accurate in editors and Go tooling. The repository-level companion is [docs/api.md](docs/api.md), which groups the exported surface into a human-oriented map.

## Maintenance

Opskit is a small open source library maintained on a best-effort basis.

The active development line lives on `main`, and that is the only line actively maintained unless explicitly noted otherwise. The minimum supported Go version is declared in [`go.mod`](go.mod), and the Go versions currently verified in CI are listed in [`.github/workflows/ci.yaml`](.github/workflows/ci.yaml).

Compatibility-impacting changes should be called out explicitly in release notes or release descriptions. Long-lived maintenance branches and backports are not planned unless explicitly noted.

## Development

This repository uses `make` for local verification:

```bash
make verify
make test-race
make govulncheck
```

`make verify` checks formatting, runs `go vet`, runs tests, and verifies that `go.mod` and `go.sum` are tidy.

CI runs verification and race tests on the supported Go versions.

## Releasing

Releases start from the repository's **Actions → Release → Run workflow**
screen. Select `main` and enter the new semantic version, such as `v0.4.0`.
The workflow validates that version and runs `make verify`, `make test-race`,
and `make govulncheck` against the exact selected commit on every supported Go
version. Only after all checks pass does it create the version tag and GitHub
Release.

Do not create or push `v*` tags manually; doing so would publish a Go module
version without the release workflow's pre-publication checks.

## Issues and Scope

Bug reports, documentation fixes, small API ergonomics improvements, and compatibility issues are welcome.

Opskit is intentionally scoped as a passive operational spine and contract
package. Large runtime features are out of scope, including HTTP routing,
command dispatch, check scheduling, retries, lifecycle management, service
discovery, dependency injection, telemetry exporting, authorization,
dashboards, and application policy.

For security issues, please follow [`SECURITY.md`](SECURITY.md) instead of opening a public issue.

## License

[MIT](LICENSE)
