# Dry-Run And Timeout Policy

This document describes the current dry-run and timeout contract in `podbridge5`.
The goal is to make it explicit which parts are already fixed as policy and which parts remain out of scope.

## Dry-run policy

The currently documented dry-run policy in `podbridge5` is limited to the `buildah.AddAndCopyOptions` path.

- default: `DefaultAddAndCopyDryRun = false`
- meaning: the default path is a mutating builder flow
- override: callers can opt in with `WithDryRun(true)` or an explicit helper path

The default is fixed to false because the main job of the current public surface is real image assembly and script injection.
In other words, dry-run is opt-in and mutation is the default.

In the current scope, dry-run only means:

- expressing no-op intent for add/copy builder mutations

Currently out of scope:

- full image build dry-run
- push dry-run
- export dry-run
- container lifecycle dry-run
- volume write dry-run

So the current dry-run contract is not a library-wide execution policy.
It is a narrow contract around the add/copy option surface.

## Timeout policy

The currently fixed shared timeout policy in `podbridge5` is the healthcheck contract.

Defaults:

- `DefaultHealthcheckInterval = 30s`
- `DefaultHealthcheckRetries = 3`
- `DefaultHealthcheckTimeout = 5s`
- `DefaultHealthcheckStartPeriod = 0s`
- minimum timeout: `MinHealthcheckTimeout = 1s`

The intent is straightforward.

- a healthcheck timeout shorter than 1 second is not allowed
- the interval can be disabled with `disable` or `0`
- the start period must not be negative

`ParseHealthcheckConfig(...)` uses shared helpers so the same rules are applied consistently.

## Currently out of scope

This document does not define timeouts for:

- build timeout
- push timeout
- export timeout
- shared container start/stop timeout policy
- a library-wide standard for context cancellation

So the current timeout policy is fixed only for the healthcheck contract.
The rest should be addressed in the remaining Sprint 2 work together with the public API boundary.
