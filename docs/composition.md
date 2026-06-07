# Composition Guide

Opskit is the operational contract layer, not the application host.

Use this guide to understand where Opskit should sit once the rest of a service
has configuration, workers, dependencies, clients, state, HTTP presentation, and
background execution.

## The Boundary

Opskit owns shared passive contracts:

- component identity
- status
- readiness
- inspection
- checks and check groups
- commands
- events and observers
- registry read models

Opskit does not own runtime behavior:

- HTTP routing
- check scheduling
- command dispatch
- retries
- lifecycle
- authorization
- telemetry exporting
- dependency behavior
- configuration loading
- application policy

That split is the design. Opskit gives independent packages one operational
language without making them import each other.

## Service Assembly

A service should register operational components at the assembly boundary:

```go
ops := opskit.NewRegistry()

ops.MustRegister(configComponent, opskit.Required())
ops.MustRegister(workerRuntimeComponent, opskit.Required())
ops.MustRegister(dependencyComponent, opskit.Required())
ops.MustRegister(clientComponent, opskit.Optional())
ops.MustRegister(buildInfoComponent, opskit.Informational())
```

The registry then becomes the shared read model for HTTP presentation, worker
execution, tests, CLIs, logs, and diagnostics.

## Kit Series Shape

The intended long-term Kit Series composition looks like this:

```go
ops := opskit.NewRegistry()

ops.MustRegister(configkitops.Component(configManager), opskit.Required())
ops.MustRegister(workerkitops.Component(runtime), opskit.Required())
ops.MustRegister(dependkitops.Component(dependencies), opskit.Required())
ops.MustRegister(clientkitops.Component(clients), opskit.Optional())
ops.MustRegister(statekitops.Component(stateManager), opskit.Required())
ops.MustRegister(buildInfoComponent, opskit.Informational())
```

Each domain kit owns its own behavior and maps outward into Opskit contracts.
Opskit remains the common vocabulary.

> **Planned integration: Kit Series adapters**
>
> The adapter names above are illustrative. Servekit, Workerkit, Configkit,
> Dependkit, Clientkit, and Statekit have not all been updated to expose stable
> Opskit adapters yet. Do not copy those package names as working code until the
> sibling repositories publish real adapters.

## Servekit Presentation

Servekit is the natural place to present Opskit state over HTTP:

```go
server := servekit.New(
	servekit.WithOpsReadiness(ops),
	servekit.WithOpsAdmin(ops, servekit.WithAuthGate(requireAdmin)),
)
```

That is the right boundary because Servekit owns routing, middleware, probes,
auth gates, response encoding, and HTTP lifecycle.

> **Planned integration: Servekit presentation**
>
> Servekit has not yet been updated with stable Opskit readiness or admin
> presentation options. This section documents the intended integration shape,
> not runnable code.

## Workerkit Execution

Workerkit is the natural place to execute Opskit checks and commands:

```go
runtime.Register(opskitworker.CheckGroupWorker(ops))
runtime.Register(opskitworker.CommandWorker(ops))
```

That is the right boundary because Workerkit owns lifecycle, scheduling,
timeouts, retries, concurrency, command admission, and shutdown policy.

> **Planned integration: Workerkit execution**
>
> Workerkit has not yet been updated with stable Opskit check or command worker
> adapters. This section documents the intended integration shape, not runnable
> code.

## Application-Owned Composition

You do not need sibling kits to use Opskit directly.

Applications can define their own components now:

```go
ops.MustRegister(opskit.ComponentFunc{
	Info: opskit.ComponentInfo{
		Name: "build",
		Kind: "metadata",
	},
	Fn: func(context.Context) opskit.Status {
		return opskit.ReadyStatus("build metadata loaded",
			opskit.Attr("version", version),
		)
	},
}, opskit.Informational())
```

Applications can also implement richer component types that support inspection,
checks, or commands. That keeps operational data coherent before the rest of the
Kit Series is integrated.
The application remains responsible for presenting registry data and invoking
active capabilities.

## Rules Of Thumb

Register components where the service is assembled.

Keep component names stable. Operational consumers may put names in paths, logs,
alerts, dashboards, and tests.

Use required readiness sparingly but deliberately. If a component blocks serving
traffic, make it required. If it is useful but non-critical, make it optional. If
it should never influence readiness, make it informational.

Keep status passive. Use checks and commands for active work.

Put auth and exposure policy outside Opskit. Opskit values are safe shapes, but
they are not permission checks.
