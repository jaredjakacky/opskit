# Examples Guide

The examples are designed to show Opskit's boundary clearly: Opskit defines
passive contracts and read models, while applications and sibling kits decide
when to present, execute, authorize, or schedule operational work.

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
validation, dispatch, concurrency, and retries belong outside Opskit.

## Planned Examples

The remaining integration scenarios depend on sibling Kit Series packages being
updated around Opskit. They are listed as planned examples in
[`examples/README.md`](../examples/README.md) rather than fake code or empty
placeholder directories, so the repository stays buildable and honest.
