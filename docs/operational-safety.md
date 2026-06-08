# Operational Safety

Opskit values are designed for operational surfaces: logs, readiness responses,
admin endpoints, test output, support tooling, dashboards, and diagnostics.

That makes the API convenient, but it also means every component must treat
returned data as potentially visible.

## The Rule

Anything returned through Opskit must be safe to expose.

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

If `Inspect` returns an error while building a component snapshot, Opskit copies
that error text into `inspection_error`. Return only safe, redacted inspection
errors.

If a component `Status`, `Readiness`, or `Inspect` method panics during a
registry read model, Opskit recovers and emits only a generic panic message. It
does not expose the recovered panic value.

## Check Errors

`FailedCheck` copies `err.Error()` into the public `error` field:

```go
return opskit.FailedCheck("cache ping failed", safeErr, elapsed)
```

Only pass errors that are already safe. If the underlying error may include
secrets or private data, wrap or replace it before returning:

```go
if err != nil {
	return opskit.FailedCheck("cache ping failed", errors.New("timeout"), elapsed)
}
```

## Command Results And Errors

`FailedCommand` copies `err.Error()` into the public `error` field.
`CompletedCommand` stores `Result` as public operational output.

Return only safe command errors and result values. Do not include raw payloads,
tokens, user data, credentials, or internal authorization details.

## Command Payloads

`CommandRequest.Payload` is raw JSON. Opskit does not authenticate, authorize,
validate, or limit it.

Any presentation layer that accepts user-supplied command payloads must handle:

- authentication
- authorization
- request size limits
- JSON schema or semantic validation
- audit logging where appropriate
- timeout and cancellation policy

Command handlers should return only safe `CommandResult.Result` values.

## HTTP Exposure

> **Planned integration: Servekit admin exposure**
>
> Servekit does not yet expose stable Opskit admin routes. When it does, those
> routes must be authenticated and should default to conservative exposure.

Until then, applications that expose Opskit data over HTTP should:

- require authentication for admin endpoints
- authorize commands separately from status reads
- use bounded contexts
- set response size limits where inspection can be large
- decide whether check and command errors are visible to every admin caller
- keep public readiness probes narrower than full admin snapshots

## Worker Execution

> **Planned integration: Workerkit execution**
>
> Workerkit does not yet expose stable Opskit check or command execution
> adapters. When it does, execution policy should live there, not in Opskit.

Checks and commands should run under explicit timeout, retry, concurrency,
admission, and shutdown policy. Opskit only defines the contracts and result
shapes.

## Safe Defaults Checklist

Before exposing a component through Opskit, check:

- component name is stable and path-safe
- status is cheap and local
- attributes are low-cardinality and non-secret
- inspection is redacted
- inspection errors are safe strings
- failed check and command errors are safe strings
- command payloads are validated outside Opskit
- command results contain only operational output
- HTTP admin presentation is authenticated
- active checks and commands run under bounded contexts
