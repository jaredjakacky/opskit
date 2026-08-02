# Examples Guide

The examples are designed to show Opskit's boundary clearly: Opskit defines
passive read models and contracts for both descriptive state and active
capabilities, while applications and sibling kits decide when to present,
execute, authorize, or schedule operational work.

## Reading Order

Start with [`examples/basic`](../examples/basic). It registers two components and
prints aggregate readiness.

Then read [`examples/readiness-policies`](../examples/readiness-policies). It
shows why registration policy matters: required components block readiness,
optional components appear without blocking, and informational components stay
out of readiness entirely.

[`examples/inspection`](../examples/inspection) adds the `Inspector` capability
for safe diagnostic data.

[`examples/checks`](../examples/checks) shows passive check metadata and the
active check contracts. Opskit can discover `CheckDescriber`, `Checker`, and
`CheckGroup` implementations, but it does not run them on an interval or decide
retry policy.

[`examples/commands`](../examples/commands) shows passive command metadata and
command request and result shapes. Opskit can discover `CommandDescriber` and
`CommandHandler` implementations, but authentication, authorization,
transport validation, dispatch, concurrency, and retries belong outside Opskit.
The handler remains responsible for command-specific decoding and semantic
validation.

## Cross-Kit Examples

Servekit, Workerkit, Configkit, and Dependkit now contain runnable integration
examples for the presentation, execution, and domain behavior they own. See the
[examples directory index](../examples/README.md#cross-kit-examples) for direct
links. Keeping those programs in sibling repositories avoids adding runtime and
presentation dependencies to Opskit itself.
