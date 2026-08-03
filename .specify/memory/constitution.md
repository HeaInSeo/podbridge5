# podbridge5 Constitution

<!--
  ②-form (D-12): this file does NOT own cross-repo invariants. It references the
  platform canonical constitution and indexes only THIS repo's own enforced
  constraints. SoT for those is the rules themselves (Makefile gates / CI), not
  this prose.
-->

## Cross-repo invariants live in the platform canonical (NodeVault §4)

Cross-repo invariants — reproducibility, `casHash`, `stableRef`, the artifact
dual-axis (`lifecycle_phase` / `integrity_health`), the sori boundary, and the
image-build / ResolveRecipe rules — are owned solely by the platform canonical:
**`github.com/HeaInSeo/NodeVault` — `docs/PLATFORM_MASTER_DESIGN.md` §4**
(immutable architecture decisions). This document does not restate or fork
them; on any conflict, §4 wins.

Note: podbridge5 is the **in-process image builder** (buildah wrapper) linked
into NodeVault. The **image-build invariant** — rootless build, no privileged
fallback (§4.8) — is therefore **cross-repo** and owned by the platform canonical. Even
though this repo is where that build actually happens, the invariant is not
restated or forked here; NodeVault §4.8 is canonical and wins on any conflict.

## Process discipline (repo-operational — owned by this repo)

- **Deterministic gates are the guarantee.** Merge is decided by the **required**
  checks (golangci-lint incl. gosec, govulncheck, race tests, `go vet` across
  build tags, CodeQL). LLM/agent review is **advisory**: a passing review never
  merges alone, a failing **required** gate is never overridden. `actionlint`
  and `module-drift` also run in CI but are **not required checks** (advisory —
  see the local-rules index below); they do not decide merges.
- **Agent operating rules — see `AGENTS.md`** (repo root). That file governs how
  agents work in this repo; it is authoritative and is not duplicated here.
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

**Status: PROPOSED (not enforced in this repo).** §1.10 is a cross-repo
rule (not yet part of NodeVault §4); podbridge5 has **no deterministic rule** enforcing
it today. Marked PROPOSED, not IMPLEMENTED, until such a gate exists.

**Version**: 1.0.0 | **Ratified**: 2026-08-02 | **Last Amended**: 2026-08-02
