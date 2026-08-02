# Composition Guide

Opskit is the operational contract layer, not the application host.

Use this guide to place Opskit between domain-owned operational state,
Servekit presentation, and Workerkit execution without turning registration
into implicit execution.

## The Boundary

Opskit owns shared contracts and data shapes for:

- component identity
- status
- readiness
- inspection
- active checks and check groups
- active command handlers
- passive descriptor and capability discovery
- registry read models

Opskit does not own runtime behavior:

- HTTP routing or response encoding
- check scheduling or invocation
- command dispatch
- retries or concurrency limits
- lifecycle
- authorization
- telemetry exporting
- dependency behavior
- configuration loading
- application policy

The package and registry are passive. `Checker`, `CheckGroup`, and
`CommandHandler` describe active capability hooks, but Opskit never invokes
those hooks. Applications explicitly select capabilities and give them to an
execution layer.

## Current Kit Series Shape

The current sibling packages integrate through root Opskit contracts rather
than illustrative pairwise adapter packages:

| Package | Current Opskit relationship |
| --- | --- |
| Servekit | `WithOps` reads registry readiness; `WithOpsAdmin` presents read-only inventory and component snapshots |
| Workerkit | `Runtime` implements `Component`, `ReadinessContributor`, and `Inspector`; `NewCheckLoop`, `NewCheckGroupLoop`, and `CommandFromOpskit` execute explicitly supplied active capabilities |
| Configkit | `Manager[T]` implements `Component`, `ReadinessContributor`, and `Inspector`; `ReloadCommand` exposes a separate command component |
| Dependkit | `Registry` implements `Component`, `ReadinessContributor`, `Inspector`, and `CheckGroup`; `CheckCommands` exposes separate command handlers |

Clientkit and Statekit should follow the same import rule when their public
Opskit integrations stabilize: implement Opskit contracts in the domain kit,
without making Opskit import the domain kit or making domain kits import one
another.

## Service Assembly

Register components where the application assembles the service:

```go
ops := opskit.NewRegistry()

ops.MustRegister(configManager, opskit.Required())
ops.MustRegister(dependencies, opskit.Required())
ops.MustRegister(runtime, opskit.Required())
ops.MustRegister(configReload, opskit.Informational())
ops.MustRegister(buildInfo, opskit.Informational())
```

The registry is then the shared descriptive model for HTTP presentation, tests,
CLIs, logs, and diagnostics. Registering `dependencies` or `configReload` does
not cause their checks or commands to run.

## Servekit Presentation

Servekit presents the shared registry over HTTP:

```go
server := servekit.New(
	servekit.WithOps(
		ops,
		servekit.WithOpsAdmin(),
		servekit.WithOpsAdminAuthGate(requireAdmin),
		servekit.WithOpsTimeout(2*time.Second),
	),
)
```

`WithOps` adds registry readiness to Servekit's built-in `GET /readyz`
decision. Servekit evaluates its own lifecycle gate first, then Opskit
readiness, then any lightweight `WithReadinessChecks` predicates.

`WithOpsAdmin` opts into two generic read-only routes:

- `GET /admin/components` returns `Registry.Entries()` inventory without
  evaluating component state.
- `GET /admin/components/{name}` returns `Registry.Snapshot(...)`, including
  status, readiness, and safe inspection when supported.

These routes do not run checks, dispatch commands, or expose an active command
endpoint. Admin routes are not enabled by `WithOps` alone and are not
authenticated unless the application supplies `WithOpsAdminAuthGate` or
equivalent network-level protection.

## Workerkit Execution

Workerkit executes selected Opskit capabilities under runtime policy. It does
not scan an Opskit registry and automatically execute every discovered hook.

Use `NewCheckLoop` or `NewCheckGroupLoop` for periodic checks:

```go
err := runtime.Register(workerkit.WorkerSpec{
	Name: "dependencies",
	Worker: workerkit.NewCheckGroupLoop(
		dependencies,
		workerkit.WithCheckInterval(30*time.Second),
		workerkit.WithCheckTimeout(5*time.Second),
	),
}, workerkit.WithWorkerReadinessContribution(false))
```

Here the Dependkit registry is both the active `CheckGroup` and the cached
dependency-readiness component. Making the Workerkit loop non-contributing
avoids counting the same dependency gate twice when both Dependkit and the
Workerkit runtime are registered with Opskit. The application may choose a
different readiness authority, but it should choose one deliberately.

Use `CommandFromOpskit` to bind one descriptor and handler into Workerkit:

```go
descriptors := configReload.Commands(ctx)
if len(descriptors) != 1 {
	return fmt.Errorf("reload command descriptors = %d, want 1", len(descriptors))
}

err := runtime.Register(workerkit.WorkerSpec{
	Name:   "config",
	Worker: configWorker,
}, workerkit.WithCommandSpec(
	workerkit.CommandFromOpskit(descriptors[0], configReload),
))
```

Workerkit then owns dispatch, admission, timeout, retry, concurrency, panic
recovery, observation, and lifecycle policy. The Configkit handler still owns
reload semantics and command-specific payload validation.

## Configkit State And Reload

`Manager[T]` directly exposes cached configuration lifecycle state through
Opskit:

```go
manager := configkit.NewManager[AppConfig](
	configkit.WithIdentity("config"),
)
ops.MustRegister(manager, opskit.Required())
```

Its status and readiness methods do not read the source or execute the config
pipeline. Reload remains an explicit active command:

```go
reload := configkit.ReloadCommand(manager, source, pipeline)
ops.MustRegister(reload, opskit.Informational())
```

Registering the reload component enables passive inventory and descriptor
discovery only. Workerkit or another explicit execution layer must invoke it.

## Dependkit State And Checks

`*dependkit.Registry` directly exposes cached dependency state and implements
`opskit.CheckGroup`:

```go
dependencies := dependkit.NewRegistry()
ops.MustRegister(dependencies, opskit.Required())
```

Its `Status` and `Readiness` methods read the latest local registry snapshot;
they do not run dependency checks. Pass the registry explicitly to
`workerkit.NewCheckGroupLoop` when Workerkit should keep that state fresh.

Manual check commands are separate:

```go
checkCommands := dependkit.CheckCommands(dependencies)
ops.MustRegister(checkCommands, opskit.Informational())
```

Bind those descriptors and the handler through `workerkit.CommandFromOpskit`
when operators need Workerkit-owned check-now dispatch.

## Application-Owned Composition

Applications can implement Opskit contracts without using sibling kits:

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

Applications remain responsible for choosing what to register, what blocks
readiness, which active capabilities may execute, and which execution and
presentation policies apply.

## Rules Of Thumb

Register components at the application composition root.

Keep status, readiness, inspection, and descriptor methods fast, local,
side-effect-free, and safe for concurrent calls. Expensive work belongs in
`Check`, `CheckAll`, `HandleCommand`, or domain-owned execution paths.

Treat capability discovery as metadata, not authorization or permission to
execute. Explicitly bind selected capabilities to Workerkit.

Keep component names stable. Operational consumers may put names in paths,
logs, alerts, dashboards, and tests. Use `opskit.ValidateComponentName` or
`opskit.IsValidComponentName` when validating before registration.

Keep component kinds, labels, and attribute keys stable and low-cardinality.
Use labels for identity-level metadata and attributes for runtime or
result-specific metadata.

Use required readiness deliberately. If a component blocks serving traffic,
make it required. If it is useful but non-critical, make it optional. If it
should never influence readiness, make it informational.

Put HTTP auth and exposure policy in Servekit or the application edge. Put
active execution policy in Workerkit or another explicit application runtime.
