# Opskit Examples

This page is the directory index for Opskit's runnable examples.

These examples are part of the public documentation, not just smoke-test
programs. Use this page when you want the short version: what examples exist,
what each one demonstrates, and what to run next.

Read the examples as a progression from the passive registry path outward into
readiness policy, inspection, checks, and commands.

## Read Order

1. [basic](basic)
2. [readiness-policies](readiness-policies)
3. [inspection](inspection)
4. [checks](checks)
5. [commands](commands)

## What Each Example Shows

- [basic](basic)
  The core registry story: two components, required and optional readiness
  policy, and aggregate readiness JSON.
- [readiness-policies](readiness-policies)
  Required, optional, and informational components side by side so the readiness
  and status read models are easy to compare.
- [inspection](inspection)
  A component that implements `Inspector` and returns safe diagnostic data in a
  component snapshot.
- [checks](checks)
  Passive `Checker` and `CheckGroup` capability discovery, plus explicit check
  execution outside the registry's passive status/readiness path.
- [commands](commands)
  Passive `CommandHandler` discovery and one explicit operational command
  invocation with an opaque JSON payload.

## Planned Examples

Coming after v0.1, once sibling kits expose stable Opskit adapters:

- `servekit-readiness`: present registry readiness over HTTP.
- `servekit-admin`: present status, snapshots, inspection, checks, and commands
  through authenticated admin routes.
- `workerkit-checks`: let Workerkit execute discovered `Checker` and
  `CheckGroup` capabilities under runtime policy.
- `kit-series-composition`: compose Servekit, Workerkit, Configkit, Dependkit,
  Clientkit, Statekit, and application-owned components into one registry
  without pairwise imports.
- `production-composition`: show a production-shaped assembly after the
  Servekit and Workerkit examples are real.

## Run Them

Run examples from the repository root:

```bash
go run ./examples/<name>

# for example
go run ./examples/basic
go run ./examples/readiness-policies
go run ./examples/inspection
go run ./examples/checks
go run ./examples/commands
```

Build all runnable examples with:

```bash
go build ./examples/...
```
