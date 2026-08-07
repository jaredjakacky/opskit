# Operational Safety

Opskit values are designed for operational surfaces: logs, readiness responses,
admin endpoints, test output, support tooling, dashboards, and diagnostics.

That makes the API convenient, but it also means every component must treat
returned data as potentially visible.

## The Rule

Any operational data value returned through Opskit must be safe to expose.

This rule applies to operational data values. Ordinary Go `error` returns such
as the error from direct `Registry.Inspect` are private diagnostic/control-flow
channels and may contain arbitrary text. Do not copy them into public data or
logs without an application-owned presentation policy.

Do not return:

- secrets
- credentials
- API tokens
- session IDs
- raw connection strings
- raw request bodies
- private user data
- unredacted database errors
- authorization details that reveal policy internals

Return:

- stable component names
- stable low-cardinality kinds
- redacted messages
- safe attributes
- summarized state
- bounded diagnostic details
- public command outcomes

## Attributes

Attributes are intentionally simple:

```go
opskit.Attr("shard", "primary")
opskit.Attr("mode", "write-through")
opskit.Attr("backend", "redis")
```

Good attributes help operators filter and understand state without exposing
private data.

Attribute keys should be stable, low-cardinality safe tokens. Prefer ASCII
letters, ASCII digits, dots, underscores, and hyphens. Opskit does not validate
attribute keys because presentation, telemetry, and routing layers may have
different field-name or label-name rules.

Avoid attributes like:

```go
opskit.Attr("password", password)
opskit.Attr("dsn", rawDSN)
opskit.Attr("authorization", header)
```

## Status Messages

Status messages should explain the operational state without dumping internals:

```go
opskit.NotReadyStatus("configuration has not loaded")
opskit.DegradedStatus("cache is serving with elevated latency")
opskit.FailedStatus("dependency health check failed")
```

Avoid raw error strings when they might include hostnames, usernames, queries,
request payloads, or credentials.

## Inspection

`Inspection` is the most flexible Opskit shape and therefore the easiest to
misuse.

Good inspection data:

```go
opskit.Inspection{
	Summary: "cache online",
	Details: map[string]any{
		"mode":    "write-through",
		"entries": 4217,
	},
	Attributes: []opskit.Attribute{
		opskit.Attr("shard", "primary"),
	},
}
```

Unsafe inspection data:

```go
opskit.Inspection{
	Details: map[string]any{
		"dsn":      rawDatabaseURL,
		"api_key":  apiKey,
		"last_sql": queryWithUserData,
	},
}
```

Presentation layers may pass inspection through directly. Redact before
returning.

`Summary` and `Details` must also be JSON-marshalable. Prefer strings, numbers,
booleans, null values, slices, maps with string keys, or structs with stable JSON
tags. Do not return functions, channels, cyclic values, non-finite floats, or
values that require unavailable custom encoders.

If `Inspect` returns an error while building a component snapshot, Opskit omits
the inspection and adds a generic `inspection_failure`. Arbitrary inspector
error text is available to a direct `Registry.Inspect` caller but is never
copied into the public snapshot.

If a component `Status`, `Readiness`, `Inspect`, `Checks`, or `Commands` method
panics during a registry read model, Opskit recovers and emits only a generic
panic message or sentinel error. It does not expose the recovered panic value.

## Check Failures

`FailedCheck` fails closed and does not accept or publish an `error`:

```go
return opskit.FailedCheck("cache ping failed", elapsed)
```

When public detail is useful, provide it explicitly with
`FailedCheckWithFailure`. Keep the underlying error in the owning kit or
application for internal logging and control flow:

```go
if err != nil {
	return opskit.FailedCheckWithFailure(
		"cache ping failed",
		opskit.Failure{Code: "timeout", Message: "cache did not respond before the deadline"},
		elapsed,
	)
}
```

## Command Results And Failures

`FailedCommand` and `RejectedCommand` omit detailed failure text. Their
`WithFailure` variants accept an explicit public `Failure`.
`CompletedCommand` stores `Result` as public operational output.

Return only safe command failure detail and result values. Do not include raw
payloads, tokens, user data, credentials, or internal authorization details.

## Command Payloads

`CommandRequest.Payload` is raw JSON. Opskit does not authenticate, authorize,
validate, or limit it.

Any presentation layer that accepts user-supplied command payloads must handle:

- authentication
- authorization
- request size limits
- valid JSON and transport-level validation
- selection of an allowed command
- audit logging where appropriate
- timeout and cancellation policy

Command handlers remain responsible for command-specific decoding and semantic
validation. They should return only safe `CommandResult.Result` values.

## HTTP Exposure

Servekit exposes the current generic Opskit presentation path through
`servekit.WithOps(...)`. `WithOpsAdmin()` opts into two read-only routes:

- `GET /admin/components`
- `GET /admin/components/{name}`

The routes present registry inventory and component snapshots. They do not run
checks, dispatch commands, or execute other active capabilities. Admin routes
are disabled unless `WithOpsAdmin()` is supplied and are unauthenticated unless
the application adds `WithOpsAdminAuthGate(...)` or equivalent network-level
protection.

Applications that expose Opskit data over HTTP should:

- require authentication for admin endpoints
- authorize commands separately from status reads
- use bounded contexts
- set response size limits where inspection can be large
- decide whether explicit check and command failure details are visible to every admin caller
- keep public readiness probes narrower than full admin snapshots

## Worker Execution

Workerkit exposes the current execution path for active Opskit capabilities:

- `workerkit.NewCheckLoop(...)` executes one `opskit.Checker` periodically.
- `workerkit.NewCheckGroupLoop(...)` executes one `opskit.CheckGroup`
  periodically.
- `workerkit.CommandFromOpskit(...)` adapts one command descriptor and handler
  into Workerkit dispatch.

Applications must select and bind those capabilities explicitly. Registration
or discovery in Opskit is not authorization or permission to execute them.
Workerkit owns timeout, retry, concurrency, admission, panic recovery,
observation, lifecycle, and shutdown policy; domain handlers own operation
semantics and command-specific payload validation.

## Safe Defaults Checklist

Before exposing a component through Opskit, check:

- component name is stable and path-safe
- status is cheap and local
- attributes are low-cardinality and non-secret
- inspection is redacted
- explicit inspection, check, and command messages are public-safe
- internal error causes remain outside Opskit results
- command payloads are validated outside Opskit
- command results contain only operational output
- HTTP admin presentation is authenticated
- active checks and commands run under bounded contexts
