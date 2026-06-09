# Opskit API Map

This document is the human-oriented map of Opskit's exported Go API. The
canonical symbol documentation lives in the package doc comments and should stay
accurate in editors, `go doc`, and pkg.go.dev. This file explains how the pieces
fit together and what each part is for.

Opskit is intentionally small. It gives services one shared operational contract
for component identity, status, readiness, inspection, checks, and commands. It
does not run those operations. Callers, applications, and higher-level kits
decide when to expose, execute, authorize, or schedule them.

## Design Boundaries

Opskit's API is built around three rules.

First, component status is descriptive. `Status(context.Context)` should return a
fast local snapshot. It should not ping dependencies, reload configuration,
dispatch commands, start work, or mutate lifecycle state.

Second, readiness is explicit. A component can be unhealthy, degraded, optional,
or informational without forcing every consumer to infer admission policy from a
single status field.

Third, every value returned by Opskit contracts is safe for operational surfaces.
Statuses, attributes, inspections, check errors, and command results may flow
into logs, admin endpoints, dashboards, support tooling, or test output.
Components must redact secrets before returning data.

## Core Vocabulary

### `State`

`State` is the shared high-level lifecycle vocabulary:

```go
const (
	opskit.StateUnknown
	opskit.StateInitializing
	opskit.StateReady
	opskit.StateDegraded
	opskit.StateNotReady
	opskit.StateFailed
	opskit.StateStopped
)
```

Domain packages can keep richer internal state machines, but they should map
outward-facing operational state into this vocabulary when reporting through
Opskit.

### `Attribute`

`Attribute` is a safe operational key/value pair:

```go
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func Attr(key, value string) Attribute
```

Use attributes for low-cardinality details such as component role, shard,
region, target name, backend type, mode, or policy. Do not use them for tokens,
credentials, raw connection strings, personally identifying data, or unredacted
error payloads.

Opskit does not validate attribute keys. Prefer stable safe-token keys using
ASCII letters, ASCII digits, dots, underscores, and hyphens. Presentation and
telemetry layers may apply stricter rules before turning attributes into log
fields, metrics labels, filters, or routes.

### `Duration`

`Duration` wraps `time.Duration` with JSON that humans can read:

```go
func NewDuration(time.Duration) Duration

func (d Duration) String() string
func (d Duration) TimeDuration() time.Duration
func (d Duration) MarshalJSON() ([]byte, error)
func (d *Duration) UnmarshalJSON([]byte) error
```

`Duration` marshals as strings such as `"150ms"`, `"2s"`, and `"1m30s"` instead
of raw nanoseconds.

## Components

### `Component`

`Component` is the minimum contract for anything registered with Opskit:

```go
type Component interface {
	ComponentInfo() ComponentInfo
	Status(context.Context) Status
}
```

A component can represent configuration, a dependency group, a worker runtime, a
client set, application state, build metadata, or an application-owned
operational concern.

### `ComponentInfo`

`ComponentInfo` gives a component its stable operational identity:

```go
type ComponentInfo struct {
	Name        string      `json:"name"`
	Kind        string      `json:"kind,omitempty"`
	Description string      `json:"description,omitempty"`
	Labels      []Attribute `json:"labels,omitempty"`
}

func ValidateComponentName(name string) error
func IsValidComponentName(name string) bool
```

`Name` must be unique within a registry. It must be one safe path segment using
ASCII letters, ASCII digits, dots, underscores, and hyphens. Empty names, spaces,
slashes, `.` and `..`, colons, and other path-hostile characters are rejected at
registration time. Use `ValidateComponentName` when callers need the same
sentinel errors returned by registration. Use `IsValidComponentName` when only a
boolean predicate is needed.

`Kind` should be low-cardinality, such as `config`, `worker_runtime`,
`dependencies`, `clients`, `state`, or `build`. Opskit does not validate
`Kind`; prefer stable safe tokens using ASCII letters, ASCII digits, dots,
underscores, and hyphens because presentation and telemetry layers may use
kinds in filters, labels, dashboards, or routes. `Description` is optional human
context for admin surfaces.

`Labels` are stable identity metadata for passive inventory, routing, filtering,
dashboards, and admin presentation. Labels must be safe to expose anywhere
`ComponentInfo` appears. Do not use labels for secrets, user data, request IDs,
dynamic health details, or high-cardinality values.

Use `Attribute` fields on status, inspection, checks, commands, and future event
records for runtime or result-specific metadata.

### `ComponentFunc`

`ComponentFunc` is the lightweight adapter for simple components:

```go
type ComponentFunc struct {
	Info ComponentInfo
	Fn   func(context.Context) Status
}
```

It is useful for application-owned status sources that do not need a dedicated
type:

```go
component := opskit.ComponentFunc{
	Info: opskit.ComponentInfo{
		Name: "build",
		Kind: "metadata",
	},
	Fn: func(context.Context) opskit.Status {
		return opskit.ReadyStatus("build metadata loaded",
			opskit.Attr("version", version),
		)
	},
}
```

Nil function adapters are safe. A `ComponentFunc` without `Fn` returns an
unknown, not-ready status instead of panicking.

## Status

`Status` is the current operational summary for one component:

```go
type Status struct {
	State      State       `json:"state"`
	Ready      bool        `json:"ready"`
	Message    string      `json:"message,omitempty"`
	UpdatedAt  *time.Time  `json:"updated_at,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}
```

Constructors set `UpdatedAt` to the current UTC time and defensively copy
attributes:

```go
func ReadyStatus(message string, attrs ...Attribute) Status
func DegradedStatus(message string, attrs ...Attribute) Status
func NotReadyStatus(message string, attrs ...Attribute) Status
func FailedStatus(message string, attrs ...Attribute) Status
func UnknownStatus(message string, attrs ...Attribute) Status
```

Readiness-oriented defaults:

| Constructor | State | Ready |
| --- | --- | --- |
| `ReadyStatus` | `ready` | `true` |
| `DegradedStatus` | `degraded` | `true` |
| `NotReadyStatus` | `not_ready` | `false` |
| `FailedStatus` | `failed` | `false` |
| `UnknownStatus` | `unknown` | `false` |

`Status.Ready` is a component-level signal. Use registry readiness policy to
decide whether that signal blocks aggregate service readiness.

## Readiness

Status answers "what state is this component in?" Readiness answers "should this
service receive work?"

### `ReadinessContributor`

Components can implement readiness separately from status:

```go
type ReadinessContributor interface {
	Readiness(context.Context) Readiness
}
```

Implement this when admission policy is more nuanced than `Status.Ready`, such
as a dependency group with multiple backends, a worker runtime with drain state,
or a component that is degraded but still safe to serve.

### `Readiness`

```go
type Readiness struct {
	Ready      bool            `json:"ready"`
	Reason     string          `json:"reason,omitempty"`
	Components []ReadinessItem `json:"components,omitempty"`
}

type ReadinessItem struct {
	Name    string          `json:"name"`
	Kind    string          `json:"kind,omitempty"`
	Policy  ReadinessPolicy `json:"policy,omitempty"`
	Ready   bool            `json:"ready"`
	State   State           `json:"state"`
	Reason  string          `json:"reason,omitempty"`
	Message string          `json:"message,omitempty"`
}
```

Helpers:

```go
func ReadyReadiness(reason string, components ...ReadinessItem) Readiness
func NotReadyReadiness(reason string, components ...ReadinessItem) Readiness
func ReadinessFromItems(reason string, items ...ReadinessItem) Readiness
func ReadinessFromPolicyItems(reason string, items ...ReadinessItem) Readiness
func ReadinessFromStatus(ComponentInfo, Status) Readiness
func ReadinessItemFromStatus(ComponentInfo, Status) ReadinessItem
```

The helpers defensively copy component slices. `ReadyReadiness` and
`NotReadyReadiness` create explicit aggregate readiness results.
`ReadinessFromItems` derives aggregate readiness from child items and is the
safer helper when every child item must be ready for the aggregate to be ready.
`ReadinessFromPolicyItems` derives aggregate readiness from required child
items. Optional and informational child items are included in the result but do
not block the aggregate. Missing or unknown child item policy is treated as
`required`.
`ReadinessFromStatus` produces a single-item readiness result derived from
`Status.Ready`, `Status.State`, and `Status.Message`.

### `ReadinessPolicy`

Registration policy controls how a component participates in aggregate
readiness:

```go
const (
	opskit.ReadinessRequired
	opskit.ReadinessOptional
	opskit.ReadinessInformational
)
```

`required` components appear in readiness details and block aggregate readiness
when not ready. This is the default.

`optional` components appear in readiness details but do not block aggregate
readiness.

`informational` components remain visible through status and snapshots but are
omitted from readiness.

Unknown policy values are treated as `required`. That fail-closed behavior keeps
a bad option from silently removing a component from readiness.

## Registry

`Registry` is the passive component store and read model:

```go
type Registry struct {
	// zero value is ready to use
}

func NewRegistry() *Registry
```

The zero value works:

```go
var registry opskit.Registry
```

Registration:

```go
func (r *Registry) Register(component Component, opts ...RegisterOption) error
func (r *Registry) MustRegister(component Component, opts ...RegisterOption)

func Required() RegisterOption
func Optional() RegisterOption
func Informational() RegisterOption
func WithReadinessPolicy(policy ReadinessPolicy) RegisterOption
```

Lookup:

```go
func (r *Registry) Component(name string) (Component, bool)
func (r *Registry) Components() []Component
func (r *Registry) Entries() []ComponentEntry
```

`Components` returns components in registration order and returns a copy of the
registry slice.

`Entries` returns passive inventory data in registration order. It includes
component identity, registration policy, and capabilities without calling
component `Status`, `Readiness`, `Inspect`, `Checks`, or `Commands` methods.

Read models:

```go
func (r *Registry) Status(context.Context) SystemStatus
func (r *Registry) Readiness(context.Context) Readiness
func (r *Registry) Snapshot(context.Context, name string) (ComponentSnapshot, error)
```

`Status` returns all registered components, including informational components.
It does not synthesize a single aggregate state. Use `Readiness` when the caller
needs an admission decision.

Registry read models recover panics from component `Status`, `Readiness`,
`Inspect`, `Checks`, and `Commands` methods. Recovered panics are represented
with generic unknown or not-ready operational data, or with `ErrComponentPanicked`
for strict single-component metadata reads. Panic values are not exposed because
they may contain unsafe details.

The status read model is:

```go
type SystemStatus struct {
	Components []ComponentStatus `json:"components,omitempty"`
}

type ComponentStatus struct {
	Component    ComponentInfo         `json:"component"`
	Registration ComponentRegistration `json:"registration"`
	Capabilities ComponentCapabilities `json:"capabilities"`
	Status       Status                `json:"status"`
}

type ComponentEntry struct {
	Component    ComponentInfo         `json:"component"`
	Registration ComponentRegistration `json:"registration"`
	Capabilities ComponentCapabilities `json:"capabilities"`
}

type ComponentRegistration struct {
	ReadinessPolicy ReadinessPolicy `json:"readiness_policy"`
}
```

`ComponentRegistration` is included in status and snapshot output so consumers
can distinguish component health from readiness policy. A not-ready optional
component should look different from a not-ready required component.

`Readiness` includes required and optional components. Informational components
are omitted. If no required readiness components are registered, aggregate
readiness is not ready with reason `"no required readiness components
registered"`, even when optional components are ready. This prevents a service
from accidentally becoming ready with no required readiness contract.

Registry-level readiness policy and child item policy are separate layers.
Registration policy controls whether the registered component blocks service
readiness. `ReadinessItem.Policy` is for contributor-owned child aggregation,
such as dependency groups where some children are required and others are
optional. Contributors that need child policy should return readiness built with
`ReadinessFromPolicyItems`.

`Snapshot` returns the combined view of one component:

```go
type ComponentSnapshot struct {
	Component    ComponentInfo         `json:"component"`
	Registration ComponentRegistration `json:"registration"`
	Capabilities ComponentCapabilities `json:"capabilities"`
	Status       Status                `json:"status"`
	Readiness    *Readiness            `json:"readiness,omitempty"`
	Inspection   *Inspection           `json:"inspection,omitempty"`
	InspectionError string             `json:"inspection_error,omitempty"`
}
```

Snapshots include readiness for required and optional components. Informational
components do not receive readiness snapshots, even if they implement
`ReadinessContributor`.
If inspection fails while building a snapshot, the snapshot still includes
status, registration, capabilities, and readiness; `inspection_error` contains
the inspection failure and `inspection` is omitted.

If inspection panics while building a snapshot, `inspection_error` contains a
generic panic message and the panic value is not exposed.

`Registry` methods normalize nil contexts to `context.Background()`. Methods
that evaluate components synchronously respect canceled contexts. For request
paths, probes, and admin endpoints, pass bounded contexts.

### Registry Errors

Registration and capability accessors return sentinel errors:

```go
var (
	ErrNilComponent
	ErrEmptyComponentName
	ErrInvalidComponentName
	ErrDuplicateComponent
	ErrComponentNotFound
	ErrComponentPanicked
	ErrInspectionUnsupported
	ErrCheckerUnsupported
	ErrCheckDescriberUnsupported
	ErrCheckGroupUnsupported
	ErrCommandHandlerUnsupported
	ErrCommandDescriberUnsupported
)
```

`Status` and `Readiness` do not return errors. If evaluation is canceled before
component calls begin, they return a synthetic `opskit.registry` item describing
the cancellation.

## Capabilities

Capabilities are optional interfaces a registered component may implement.

```go
type ComponentCapabilities struct {
	ReadinessContributor bool `json:"readiness_contributor,omitempty"`
	Inspector            bool `json:"inspector,omitempty"`
	Checker              bool `json:"checker,omitempty"`
	CheckDescriber       bool `json:"check_describer,omitempty"`
	CheckGroup           bool `json:"check_group,omitempty"`
	CommandHandler       bool `json:"command_handler,omitempty"`
	CommandDescriber     bool `json:"command_describer,omitempty"`
}
```

The registry reports capabilities in `ComponentStatus` and `ComponentSnapshot`.
It also exposes typed accessors:

```go
func (r *Registry) Inspect(ctx context.Context, name string) (Inspection, error)
func (r *Registry) Checker(name string) (Checker, error)
func (r *Registry) CheckDescriber(name string) (CheckDescriber, error)
func (r *Registry) Checks(ctx context.Context, name string) ([]CheckDescriptor, error)
func (r *Registry) CheckGroup(name string) (CheckGroup, error)
func (r *Registry) CommandHandler(name string) (CommandHandler, error)
func (r *Registry) CommandDescriber(name string) (CommandDescriber, error)
func (r *Registry) Commands(ctx context.Context, name string) ([]CommandDescriptor, error)
```

These accessors discover capability support. The accessors themselves do not
schedule, authorize, dispatch, or execute operations. Callers decide whether and
when to invoke returned capabilities.

## Inspection

`Inspector` exposes safe diagnostic state:

```go
type Inspector interface {
	Inspect(context.Context) (Inspection, error)
}

type Inspection struct {
	Summary    any         `json:"summary,omitempty"`
	Details    any         `json:"details,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}
```

Use `Summary` for compact operator-facing state and `Details` for richer
diagnostics. Both are intentionally `any` so domain kits can expose useful
structured values without forcing a central schema.

Inspection is not a secret vault. Presentation layers may pass inspection data
directly to admin endpoints, logs, or support tooling. Components are
responsible for returning only safe, redacted data.

`Summary` and `Details` must also be JSON-marshalable. Prefer strings, numbers,
booleans, null values, slices, maps with string keys, or structs with stable JSON
tags. Do not return functions, channels, cyclic values, non-finite floats, or
values that require unavailable custom encoders.

## Checks

Checks are active operational probes. Opskit defines the contract and data
shape; something else decides when to run them.

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

Function adapters:

```go
type CheckFunc func(context.Context) CheckResult
type CheckGroupFunc func(context.Context) CheckSummary
```

Nil `CheckFunc` and `CheckGroupFunc` values return unknown, not-ready results
instead of panicking. Nil contexts are normalized.

`CheckDescriber` is passive metadata. It helps admin surfaces, CLIs, worker
runtimes, and docs generators list supported checks without running them. The
descriptors are advisory; callers still own scheduling, execution, retry,
caching, timeout, concurrency, and readiness policy.

If `Registry.Checks` recovers a `CheckDescriber` panic, it returns
`ErrComponentPanicked` without exposing the panic value.

### `CheckDescriptor`

```go
type CheckDescriptor struct {
	Name        string      `json:"name"`
	Kind        string      `json:"kind,omitempty"`
	Description string      `json:"description,omitempty"`
	Attributes  []Attribute `json:"attributes,omitempty"`
}
```

`Kind` should be a low-cardinality category such as `dependency`, `filesystem`,
`queue`, or `client`. Attributes are operational metadata and must be safe to
expose.

### `CheckResult`

```go
type CheckResult struct {
	State      State       `json:"state"`
	Ready      bool        `json:"ready"`
	Message    string      `json:"message,omitempty"`
	Error      string      `json:"error,omitempty"`
	CheckedAt  *time.Time  `json:"checked_at,omitempty"`
	Duration   Duration    `json:"duration,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}
```

Constructors:

```go
func ReadyCheck(message string, duration time.Duration, attrs ...Attribute) CheckResult
func DegradedCheck(message string, duration time.Duration, attrs ...Attribute) CheckResult
func NotReadyCheck(message string, duration time.Duration, attrs ...Attribute) CheckResult
func FailedCheck(message string, err error, duration time.Duration, attrs ...Attribute) CheckResult
```

Defaults:

| Constructor | State | Ready |
| --- | --- | --- |
| `ReadyCheck` | `ready` | `true` |
| `DegradedCheck` | `degraded` | `true` |
| `NotReadyCheck` | `not_ready` | `false` |
| `FailedCheck` | `failed` | `false` |

Check constructors set `CheckedAt` to current UTC time, store the supplied
duration, and clone attributes. `FailedCheck` copies `err.Error()` into the
public `Error` field when `err` is non-nil, so callers must pass safe errors.

### `CheckSummary`

```go
type NamedCheck struct {
	Name   string      `json:"name"`
	Kind   string      `json:"kind,omitempty"`
	Result CheckResult `json:"result"`
}

type CheckSummary struct {
	State     State        `json:"state"`
	Ready     bool         `json:"ready"`
	Message   string       `json:"message,omitempty"`
	CheckedAt *time.Time   `json:"checked_at,omitempty"`
	Duration  Duration     `json:"duration,omitempty"`
	Results   []NamedCheck `json:"results,omitempty"`
}

func SummarizeChecks(message string, startedAt time.Time, results []NamedCheck) CheckSummary
```

`SummarizeChecks` gives check groups a consistent aggregate result. The first
matching row wins:

| Results | Summary state | Ready | Default message |
| --- | --- | --- | --- |
| none | `unknown` | `false` | `no checks ran` |
| any failed | `failed` | `false` | `one or more checks failed` |
| any not-ready, none failed | `not_ready` | `false` | `one or more checks are not ready` |
| any degraded, all ready | `degraded` | `true` | `one or more checks degraded` |
| all ready | `ready` | `true` | `all checks ready` |

If the caller passes a non-empty message, that message is preserved.

## Commands

Commands are control-plane operations. Examples include `config/reload`,
`cache/refresh`, `index/rebuild`, and `dependency/check`.

Opskit defines the request and result shape, but it does not authorize,
validate, route, queue, retry, or execute commands.

```go
type CommandHandler interface {
	HandleCommand(context.Context, CommandRequest) CommandResult
}

type CommandDescriber interface {
	Commands(context.Context) []CommandDescriptor
}

type CommandHandlerFunc func(context.Context, CommandRequest) CommandResult
```

The handler method is intentionally named `HandleCommand` rather than `Command`
because it is the active operation on a handler, distinct from command metadata.

Nil `CommandHandlerFunc` values return an unknown, rejected result instead of
panicking. Nil contexts are normalized.

`CommandDescriber` is passive metadata. It helps admin surfaces, CLIs, worker
runtimes, and docs generators list supported commands without invoking them.
The descriptors are advisory; callers still own authorization, validation,
routing, scheduling, concurrency, and execution.

If `Registry.Commands` recovers a `CommandDescriber` panic, it returns
`ErrComponentPanicked` without exposing the panic value.

### `CommandDescriptor`

```go
type CommandDescriptor struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	PayloadKind string      `json:"payload_kind,omitempty"`
	Dangerous   bool        `json:"dangerous,omitempty"`
	Idempotent  bool        `json:"idempotent,omitempty"`
	Attributes  []Attribute `json:"attributes,omitempty"`
}
```

`PayloadKind` is a human- and tool-readable payload category, not a schema.
`Dangerous` and `Idempotent` are advisory hints for presentation and execution
layers. Because false values are omitted from JSON, false means "not marked,"
not an Opskit safety or execution guarantee. Opskit does not enforce either
flag.

### `CommandRequest`

```go
type CommandRequest struct {
	Name        string          `json:"name"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	RequestedAt *time.Time      `json:"requested_at,omitempty"`
	Attributes  []Attribute     `json:"attributes,omitempty"`
}

func NewCommandRequest(name string, payload json.RawMessage, attrs ...Attribute) CommandRequest
```

`Payload` is command-specific raw JSON. Presentation layers that accept payloads
from users must perform authentication, authorization, validation, and size
limits before constructing a `CommandRequest`.

`NewCommandRequest` sets `RequestedAt` to the current UTC time and defensively
copies payload bytes and attributes. It does not validate command names or
payloads. Callers that need a custom `RequestedAt` can construct
`CommandRequest` directly.

### `CommandResult`

```go
type CommandResult struct {
	State      State       `json:"state"`
	Accepted   bool        `json:"accepted"`
	Message    string      `json:"message,omitempty"`
	Error      string      `json:"error,omitempty"`
	StartedAt  *time.Time  `json:"started_at,omitempty"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
	Duration   Duration    `json:"duration,omitempty"`
	Result     any         `json:"result,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}
```

Constructors:

```go
func AcceptedCommand(message string, attrs ...Attribute) CommandResult
func CompletedCommand(message string, result any, duration time.Duration, attrs ...Attribute) CommandResult
func RejectedCommand(message string, attrs ...Attribute) CommandResult
func FailedCommand(message string, err error, duration time.Duration, attrs ...Attribute) CommandResult
```

Defaults:

| Constructor | State | Accepted | Meaning |
| --- | --- | --- | --- |
| `AcceptedCommand` | `initializing` | `true` | admitted for asynchronous work |
| `CompletedCommand` | `ready` | `true` | admitted and completed successfully |
| `RejectedCommand` | `not_ready` | `false` | not admitted |
| `FailedCommand` | `failed` | `true` | admitted but failed |

`Accepted` means the command was admitted, not necessarily completed. Command
results are operational output and must contain only safe values.

## Common Patterns

### Register Required And Optional Components

```go
registry := opskit.NewRegistry()

registry.MustRegister(configComponent, opskit.Required())
registry.MustRegister(cacheComponent, opskit.Optional())
registry.MustRegister(buildComponent, opskit.Informational())
```

Required components decide aggregate readiness. Optional components remain
visible in readiness details without blocking startup. Informational components
stay visible through status and snapshots without participating in readiness.

### Separate Status From Active Checks

```go
type Cache struct {
	lastStatus opskit.Status
}

func (c *Cache) ComponentInfo() opskit.ComponentInfo {
	return opskit.ComponentInfo{Name: "cache", Kind: "dependency"}
}

func (c *Cache) Status(context.Context) opskit.Status {
	return c.lastStatus
}

func (c *Cache) Check(ctx context.Context) opskit.CheckResult {
	started := time.Now()
	if err := c.ping(ctx); err != nil {
		return opskit.FailedCheck("cache ping failed", errors.New("timeout"), time.Since(started))
	}
	return opskit.ReadyCheck("cache reachable", time.Since(started))
}
```

`Status` is cheap and descriptive. `Check` is explicit active work.

### Expose A Safe Snapshot

```go
snapshot, err := registry.Snapshot(ctx, "cache")
if err != nil {
	return err
}

encoded, err := json.MarshalIndent(snapshot, "", "  ")
if err != nil {
	return err
}
```

Snapshots are useful for admin endpoints because they include identity,
registration, capabilities, status, readiness when applicable, and inspection
when supported.

## Compatibility Notes

The exported API is deliberately made of simple interfaces, structs, constants,
constructors, and sentinel errors. That gives downstream kits and applications a
stable contract without pulling in a runtime dependency or framework lifecycle.

When extending the API, prefer additions over semantic changes. In particular,
avoid changing JSON field names, readiness policy behavior, state meanings,
registration validation, or constructor defaults without a major-version reason.
