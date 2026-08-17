# podbridge5 Constitution

<!--
  ②-form (D-12), authority revision AR-2026-08-17.1: this file does NOT own
  cross-repo invariants. It consumes the task Authority Snapshot and indexes
  only THIS repo's own enforced constraints. SoT for those is the rules
  themselves (Makefile gates / CI), not this prose.
-->

## Cross-repo authority — revision-pinned repository mirror

Cross-repo platform meaning is selected by the external Authority Router. For
`AR-2026-08-17.1` the scoped authority chain is:

- platform invariants: `Platform Spec Wiki — CURRENT / 1. constitution`
- platform structure / responsibility / call direction:
  `Platform Spec Wiki — CURRENT / 2. architecture`
- repository-portable mirror: `HeaInSeo/NodeVault` —
  `docs/PLATFORM_MASTER_DESIGN.md` at the same authority revision

podbridge5 does **not** treat NodeVault §4 as an independent platform canonical.
A task may consume that repository mirror only when its `Authority Snapshot`
declares `AR-2026-08-17.1`. Missing/mismatched/conflicting snapshots must stop
with `AUTHORITY_CONFLICT`; do not choose a source by timestamp, filename, or
search rank.

Note: podbridge5 is the **in-process image builder** (buildah wrapper) linked
into NodeVault. The **image-build invariant** — rootless build and no privileged
fallback — is cross-repo and is owned by the current platform authority chain.
Even though this repo is where that build happens, the invariant is not restated
or independently owned here; this repo implements/enforces its local side.

## Process discipline (repo-operational — owned by this repo)

- **Deterministic gates are the guarantee.** Merge is decided by the **required**
  checks (golangci-lint incl. gosec, govulncheck, race tests, `go vet` across
  build tags, CodeQL). LLM/agent review is **advisory**: a passing review never
  merges alone, a failing **required** gate is never overridden. `actionlint`
  and `module-drift` also run in CI but are **not required checks** (advisory —
  see the local-rules index below); they do not decide merges.
- **Agent operating rules — see `AGENTS.md`** (repo root). That file governs how
  agents work in this repo; it is authoritative for repo-local agent operations
  and is not duplicated here.
- **Spec-anchored change**; **test-first** (behavioral changes ship with tests
  that fail before / pass after; CI runs the `-race` variant).
- **Runtime-sensitive paths** (buildah/podman/storage adapters) build only under
  explicit tags (`runtime`, `integration`) and their tests run on a real
  host/VM, not the default unit lane. Compile-safety for those files is still
  gated (`go vet` with the runtime+integration tags) so production removals never
  silently break the tagged suites.
- **Branch protection**: `main` lands via PR with required checks; no direct
  pushes.

## Repo-local enforced constraints (derived index — NOT canonical)

> Derived index of THIS repo's own gates. Not canonical — SoT is the gate itself
> (`Makefile` target / CI job / `.golangci.yml`). Where a gate has no `make`
> target, its SoT is the CI workflow, noted inline.

- **golangci-lint** (IMPLEMENTED — `make lint`, CI `lint` job): lint gate. Its
  enabled set (`.golangci.yml`) includes **gosec** (G101–G602), so static
  **security analysis is enforced through this same gate** — there is no separate
  `make lint-security-check` target in this repo.
- **govulncheck** (IMPLEMENTED — CI `vuln-scan` job via `cmd/govulncheckgate`):
  vulnerability scan, blocking. No `make vuln-check` target; SoT is the CI job.
- **race tests** (IMPLEMENTED — `make test` / `make test-unit` with `-race`; CI
  `test-unit` job): concurrency safety on the unit lane.
- **go vet across build tags** (IMPLEMENTED — CI `build` job for base tags;
  `make check-runtime-build` runs the `runtime`+`integration` tag variant):
  compile/vet gate, incl. the runtime-tagged files the unit lane never compiles.
- **actionlint** (PROPOSED — runs in CI `actionlint` job but is NOT a required check, so it does not block merge): GitHub Actions workflow lint.
- **module-drift** (PROPOSED — runs in CI `module-drift` job but is NOT a required check, so it does not block merge): `go.mod`/`go.sum` tidiness.
- **CodeQL** (IMPLEMENTED — `.github/workflows/codeql.yml`): static analysis
  (Go + Actions).

> **Coverage is measured, NOT gated.** `make test-unit` writes a coverage
> profile and `make coverage-merge` reports combined statement coverage, but no
> deterministic **threshold** gate fails a build on coverage. It is therefore
> not listed as an enforced constraint above.

## §1.10 — "do not record what you did not observe"

**Authority: CURRENT platform invariant under `AR-2026-08-17.1`. Enforcement in
this repo: PROPOSED where no deterministic local gate exists.** podbridge5 has
no deterministic rule that generally enforces this invariant today. The
platform invariant's authority status and this repo's local enforcement status
are separate axes.

**Version**: 2.0.0 | **Ratified**: 2026-08-02 | **Last Amended**: 2026-08-17
