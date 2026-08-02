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
  Passive `CheckDescriber` metadata plus discovery of active `Checker` and
  `CheckGroup` capabilities. The trusted example caller invokes them explicitly
  outside the registry's passive status/readiness path.
- [commands](commands)
  Passive `CommandDescriber` metadata plus discovery of an active
  `CommandHandler`. The trusted example caller performs one low-level invocation
  with an opaque JSON payload.

## Cross-Kit Examples

The sibling kits now contain the runnable examples for the behavior they own:

- [Servekit operations](https://github.com/jaredjakacky/servekit/tree/main/examples/operations)
  presents Opskit readiness, inventory, and read-only component snapshots over
  HTTP with an explicit admin auth gate.
- [Workerkit Opskit checks](https://github.com/jaredjakacky/workerkit/tree/main/examples/opskit-checks)
  executes explicitly supplied `Checker` and `CheckGroup` capabilities under
  Workerkit lifecycle and scheduling policy.
- [Workerkit Opskit command](https://github.com/jaredjakacky/workerkit/tree/main/examples/opskit-command)
  binds one descriptor and `CommandHandler` through `CommandFromOpskit`.
- [Configkit production composition](https://github.com/jaredjakacky/configkit/tree/main/examples/10-production-composition)
  combines Configkit state, an Opskit reload command, Workerkit dispatch, and
  Servekit presentation.
- [Dependkit production composition](https://github.com/jaredjakacky/dependkit/tree/main/examples/10-production-composition)
  combines cached dependency readiness, Workerkit-owned periodic checks and
  commands, and Servekit presentation.

These examples remain in the sibling repositories so Opskit does not acquire
runtime or presentation dependencies merely for demonstrations.

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
