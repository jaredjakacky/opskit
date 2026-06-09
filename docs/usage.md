# Usage Guide

This guide is the normal Opskit path: create a registry, register components,
read status and readiness, and discover optional capabilities when a caller
explicitly needs them.

The package is intentionally passive. The intended adoption flow is:

1. define components that report fast local status
2. register them with required, optional, or informational readiness policy
3. use `Status` and `Readiness` for passive read models
4. use `Snapshot` and `Inspect` for admin diagnostics
5. use `Checker`, `CheckGroup`, and `CommandHandler` only when another runtime
   explicitly executes active work

If that path is enough, Opskit stays small. If it is not, [Composition
Guide](composition.md) explains where Servekit, Workerkit, and application code
should take over.

## Standalone Usage

Opskit does not require the rest of the Kit Series. Use it standalone when an
application already has its own HTTP server, CLI, worker runtime, tests, or
admin surface, but needs one consistent operational model.

For example:

- an existing `/readyz` handler can return `Registry.Readiness`
- a CLI `status` command can print `Registry.Status`
- a CLI `inspect` command can call `Registry.Snapshot` or `Registry.Inspect`
- tests can assert one shared readiness shape
- an existing admin route can expose safe component snapshots

Standalone usage keeps the same boundary. Opskit provides passive contracts and
read models. The application still owns HTTP routing, authorization, check
scheduling, command dispatch, retries, telemetry, and lifecycle.

## The Normal Path

### `NewRegistry`

`NewRegistry()` constructs an empty registry:

```go
ops := opskit.NewRegistry()
```

The zero value is also ready to use:

```go
var ops opskit.Registry
```

Use the pointer constructor for ordinary application code. Use the zero value
when a registry is embedded in another struct or test fixture.

### `Register`

`Register` adds one component:

```go
err := ops.Register(component, opskit.Required())
```

`MustRegister` is useful during service assembly, where invalid registration is
a programmer error:

```go
ops.MustRegister(component, opskit.Required())
```

Component names must be stable, unique, and safe as one path segment. Valid
names include `config`, `worker_runtime`, `cache.primary`, and
`WorkerA`. Invalid names include empty strings, names with spaces, names with
slashes, `.`, `..`, and path-hostile punctuation.

Use `opskit.ValidateComponentName` when code needs the same sentinel errors
returned by registration. Use `opskit.IsValidComponentName` when only a boolean
predicate is needed.

Component kinds and attribute keys should also be stable, low-cardinality safe
tokens, but Opskit does not validate them. Presentation and telemetry layers may
apply their own field, label, route, or dashboard constraints.

### `Status`

`Status(ctx)` returns every registered component in registration order:

```go
status := ops.Status(ctx)
```

Status output includes:

- component identity
- registration policy
- detected capabilities
- the component's current `Status`

Status does not derive aggregate readiness or a single aggregate state. That is
intentional. Use `Readiness` when the caller needs an admission decision.

### `Readiness`

`Readiness(ctx)` returns the aggregate readiness view:

```go
readiness := ops.Readiness(ctx)
```

Required components block aggregate readiness when not ready. Optional
components appear in readiness details but do not block. Informational
components are omitted from readiness entirely.

If a component implements `ReadinessContributor`, the registry uses that
readiness result. Otherwise, the registry derives readiness from `Status.Ready`.

If no required readiness components are registered, the aggregate readiness is
not ready. That fail-closed behavior prevents a service from accidentally
becoming ready with only optional or informational components.

## Readiness Policy

Use `Required` for components that must be ready before the service receives
work:

```go
ops.MustRegister(database, opskit.Required())
```

Use `Optional` for visible but non-blocking components:

```go
ops.MustRegister(searchClient, opskit.Optional())
```

Use `Informational` for components that should appear in status and snapshots
but should not appear in readiness:

```go
ops.MustRegister(buildInfo, opskit.Informational())
```

Unknown policy values passed through `WithReadinessPolicy` are normalized to
`Required`. Bad options fail closed.

## Inspection

Implement `Inspector` when a component has safe diagnostic data beyond status:

```go
type Cache struct{}

func (Cache) Inspect(context.Context) (opskit.Inspection, error) {
	return opskit.Inspection{
		Summary: "cache online",
		Details: map[string]any{
			"mode": "write-through",
		},
		Attributes: []opskit.Attribute{
			opskit.Attr("shard", "primary"),
		},
	}, nil
}
```

Read inspection directly:

```go
inspection, err := ops.Inspect(ctx, "cache")
```

Or include it in a full component snapshot:

```go
snapshot, err := ops.Snapshot(ctx, "cache")
```

Inspection data may flow directly to logs, admin endpoints, and support tools.
Redact before returning it. `Summary` and `Details` must be JSON-marshalable:
prefer strings, numbers, booleans, null values, slices, maps with string keys,
or structs with stable JSON tags.

## Checks

Checks are active operational work. The registry can discover a component that
implements `Checker`, `CheckGroup`, or passive `CheckDescriber` metadata, but
it does not schedule or execute checks. The caller decides whether and when to
invoke the returned capability.

```go
checks, err := ops.Checks(ctx, "dependencies")
if err != nil {
	return err
}
for _, check := range checks {
	fmt.Println(check.Name)
}

checker, err := ops.Checker("dependencies")
if err != nil {
	return err
}

result := checker.Check(ctx)
```

Use `CheckGroup` when one component owns multiple named checks:

```go
group, err := ops.CheckGroup("dependencies")
if err != nil {
	return err
}

summary := group.CheckAll(ctx)
```

Use `SummarizeChecks` to keep aggregate check results consistent.

## Commands

Commands are active control-plane operations. The registry can discover
`CommandHandler` and passive `CommandDescriber` metadata, but it does not
authorize, validate, dispatch, retry, execute, or schedule commands. The caller
decides whether and when to invoke the returned handler.

```go
commands, err := ops.Commands(ctx, "cache-admin")
if err != nil {
	return err
}
for _, command := range commands {
	fmt.Println(command.Name)
}

handler, err := ops.CommandHandler("cache-admin")
if err != nil {
	return err
}

result := handler.HandleCommand(ctx, opskit.CommandRequest{
	Name: "cache/refresh",
})
```

Presentation layers must authenticate callers, authorize the command, validate
payloads, and enforce request size limits before constructing a
`CommandRequest`.

## Contexts

Registry read methods call component methods synchronously. Pass bounded
contexts on request paths, probe paths, and admin endpoints.

Nil contexts are normalized to `context.Background()` for registry methods and
function adapters. Canceled contexts are respected. `Status` and `Readiness`
return synthetic `opskit.registry` entries when evaluation is canceled before
component calls complete.

## Examples

Runnable examples live in [`examples/`](../examples), with a directory index at
[`examples/README.md`](../examples/README.md).

The most useful examples for this guide are:

- [`examples/basic`](../examples/basic)
- [`examples/readiness-policies`](../examples/readiness-policies)
- [`examples/inspection`](../examples/inspection)
- [`examples/checks`](../examples/checks)
- [`examples/commands`](../examples/commands)
