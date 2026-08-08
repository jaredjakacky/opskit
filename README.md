# Opskit

[![Release](https://img.shields.io/github/v/release/jaredjakacky/opskit?sort=semver)](https://github.com/jaredjakacky/opskit/releases)
[![CI](https://github.com/jaredjakacky/opskit/actions/workflows/ci.yaml/badge.svg)](https://github.com/jaredjakacky/opskit/actions/workflows/ci.yaml)
[![Go Support](https://img.shields.io/badge/go%20support-1.25.x%20%7C%201.26.x-00ADD8)](https://github.com/jaredjakacky/opskit/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/jaredjakacky/opskit)](https://github.com/jaredjakacky/opskit/blob/main/LICENSE)

## What Opskit Is

Opskit is a small Go package that gives service components one shared
operational language for identity, status, readiness, inspection, checks,
commands, and safe metadata. It works on its own or as the contract layer that
lets Kit Series packages compose without importing one another.

Components expose what they know. A registry collects their passive state and
discovers optional capabilities. Applications, presentation layers, and worker
runtimes decide what to expose or execute.

| Opskit owns | Applications and higher-level packages own |
| --- | --- |
| Operational contracts and JSON-friendly data shapes | HTTP routes, response encoding, dashboards, and telemetry backends |
| Component registration and passive read models | Lifecycle, scheduling, retries, concurrency, and background work |
| Readiness aggregation and capability discovery | Check invocation, command dispatch, authentication, and authorization |

Opskit is not a runtime. It does not serve HTTP, start goroutines, schedule
checks, dispatch commands, or own application policy. In particular, the
registry never invokes `Check`, `CheckAll`, or `HandleCommand`.

## Installation

```bash
go get github.com/jaredjakacky/opskit
```

```go
import opskit "github.com/jaredjakacky/opskit"
```

## Standalone Quick Start

```go
package main

import (
	"context"
	"fmt"

	opskit "github.com/jaredjakacky/opskit"
)

func main() {
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

	readiness := ops.Readiness(context.Background())
	fmt.Printf("ready=%t reason=%q\n", readiness.Ready, readiness.Reason)
}
```

The registry derives readiness from the component's fast local status snapshot.
See [Getting Started](docs/getting-started.md) and
[`examples/basic`](examples/basic) for the complete JSON result and a runnable
version.

## Kit Series Composition

At the application composition root, register domain components in one shared
registry and give that registry to the layers that need its read models:

```go
ops := opskit.NewRegistry()

ops.MustRegister(configManager, opskit.Required())
ops.MustRegister(dependencies, opskit.Required())
ops.MustRegister(workerRuntime, opskit.Required())
ops.MustRegister(buildInfo, opskit.Informational())

server := servekit.New(
	servekit.WithOps(
		ops,
		servekit.WithOpsAdmin(),
		servekit.WithOpsAdminAuthGate(requireAdmin),
	),
)
```

Domain kits report their own operational state. Servekit presents registry
readiness and protected snapshots. Workerkit executes only capabilities the
application explicitly supplies to it. Registration and capability discovery
are not authorization or permission to execute active work.

See the [Composition Guide](docs/composition.md) for the current ownership model
and cross-kit APIs.

## Core Contracts

| Contract | Purpose | Behavior |
| --- | --- | --- |
| `Component` | Stable identity and current `Status` | Passive and required for registration |
| `ReadinessContributor` | Component-owned admission decision and optional child details | Passive; otherwise derived from `Status.Ready` |
| `Inspector` | Richer safe diagnostic state | Passive; intended for protected operational surfaces |
| `CheckDescriber` | Metadata describing supported checks | Passive; does not run a check |
| `Checker` / `CheckGroup` | Active check capability and result shapes | Invoked explicitly by Workerkit, a CLI, a test, or application code |
| `CommandDescriber` | Metadata describing supported commands | Passive; does not invoke a command |
| `CommandHandler` | Active command capability and result shapes | Invoked by an authenticated, authorized execution path |
| `Registry` | Registration, aggregation, snapshots, and capability discovery | Calls descriptive methods synchronously; never invokes active capabilities |

The [API Map](docs/api.md) documents every public type, helper, constructor,
accessor, and sentinel error.

## Readiness Policy

Status answers what state a component is in. Readiness answers whether the
service should receive work. Registration policy determines how each parent
component participates in the service-level decision:

| Policy | Included in readiness | Blocks service readiness |
| --- | --- | --- |
| `Required()` | Yes | Yes |
| `Optional()` | Yes | No |
| `Informational()` | No | No |

If no required components are registered, aggregate readiness fails closed.

Registry output preserves two separate policy levels. The registered parent's
`ReadinessPolicy` decides whether that entire component blocks service
readiness. A contributor's child `ReadinessItem.Impact` explains or derives
readiness inside that component's own domain; it never overrides the parent's
registration policy.

See the [Usage Guide](docs/usage.md#readiness-policy) and
[Design Guide](docs/design.md#readiness-flow) for aggregation details.

## Operational Safety

Opskit values may flow to admin routes, logs, dashboards, diagnostics, support
tools, telemetry adapters, and tests. Every operational value returned through
Opskit must already be safe to expose. Redact secrets, user data, raw connection
strings, command payloads, and arbitrary error text before returning public
statuses, inspections, failures, results, labels, or attributes.

Ordinary Go `error` returns, such as the error from direct `Registry.Inspect`,
are private diagnostic and control-flow channels. They are not automatically
safe to copy into operational data or logs.

Read [Operational Safety](docs/operational-safety.md) before exposing registry
data through an HTTP route, CLI, support tool, or other operational surface.

## Learn More

- [Getting Started](docs/getting-started.md): build a first registry and inspect its readiness result
- [Usage Guide](docs/usage.md): use registration, status, readiness, inspection, checks, commands, and contexts
- [Design Guide](docs/design.md): understand package boundaries, type relationships, and registry flows
- [Composition Guide](docs/composition.md): compose Servekit, Workerkit, domain kits, and application policy
- [Operational Safety](docs/operational-safety.md): protect attributes, inspections, failures, payloads, and admin exposure
- [API Map](docs/api.md): browse the complete exported surface by responsibility
- [Examples Guide](docs/examples.md): follow the runnable examples from the core registry outward

Start with [`examples/basic`](examples/basic). The
[examples index](examples/README.md) covers the full standalone sequence and
links to production-shaped examples in sibling repositories.

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

`make verify` checks formatting, enforces that Opskit's root module has no
external module dependencies, runs `go vet`, runs tests, and verifies that every
checked-in Go module is tidy.

CI runs verification and race tests on the supported Go versions.

## Releasing

Releases start from the repository's **Actions → Release → Run workflow**
screen. Select `main` and enter the new semantic version, such as `v0.4.0`.
The workflow validates that version against the module path and runs
`make verify`, `make test-race`, and `make govulncheck` against the exact
selected commit on every supported Go version. Only after all checks pass does
it create the version tag and GitHub Release.

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
