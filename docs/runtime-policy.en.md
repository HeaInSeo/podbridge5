# Runtime Initialization Policy

This document describes how `podbridge5` resolves the Podman runtime connection and store initialization path.
The goal for `Sprint 2` is to make this policy explicit and keep code and docs aligned.

## Connection precedence

On Linux, `podbridge5` resolves the Podman connection URI in this order.

1. `CONTAINER_HOST`
2. `XDG_RUNTIME_DIR`
3. rootless default `/run/user/<uid>/podman/podman.sock`
4. root default `/run/podman/podman.sock`

If `CONTAINER_HOST` is set, it always wins over automatic socket discovery.
This keeps host-side tests and remote clean-VM tests aligned with the same rule.

## Init / Shutdown contract

The public initialization entrypoints are:

- `Init()`
- `InitWithContext(ctx)`
- `Shutdown()`
- `ReexecIfNeeded()`

The current contract is:

- `Init()` / `InitWithContext(ctx)` initialize both the Podman connection and the store.
- If initialization fails, the runtime is not locked into a successful state.
- This means callers can fix the environment and retry `Init()`.
- `Shutdown()` is only valid after a successful initialization.
- After `Shutdown()`, a fresh `Init()` is allowed.

## Classifiable errors

Runtime initialization errors are kept classifiable with `errors.Is`.

- `ErrRuntimeConnectionUnavailable`
- `ErrRuntimeStoreUnavailable`
- `ErrRuntimeNotInitialized`

The intent is straightforward.

- distinguish Podman socket / connection failures
- distinguish store initialization failures
- distinguish calls made before runtime initialization

## Relation to tests

The local preflight checks should follow the same rules as the code.

- `make runtime-host-check`
- `make test-runtime`
- `make test-runtime-integration`

`runtime-host-check` uses the same precedence.

1. `CONTAINER_HOST`
2. `XDG_RUNTIME_DIR`
3. default Podman socket

That keeps the host preflight and the Go runtime resolution path in sync.

## Out of scope

This document does not cover:

- detailed build/push/export option policies
- healthcheck timeout parsing policy
- the clean VM lifecycle itself

Those are documented separately.
