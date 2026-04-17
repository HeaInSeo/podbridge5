# Runtime Validation

This document describes the runtime validation path for `podbridge5`.
The main README stays focused on project description, while the VM-based validation flow is documented here.

## Validation paths

Validation is split into three layers.

### 1. Local unit path

This is the fast path on the current host.

- `make test-unit`
- `make runtime-env-check`
- `make test-runtime`
- `make test-runtime-integration`
- `make runtime-env-check`

This path is useful for code changes and lightweight validation.
`test-unit` is the fast path without runtime-tagged tests, while `test-runtime` exercises the current host with Podman/buildah available.
Since the repository still depends on `buildah`, `containers/storage`, and `containers/image`, the runtime path may still require host packages to be installed.

### 2. Local runtime / integration path

This is the host-side path that exercises real Podman/buildah behavior on the current machine.

- `make test-runtime`
- `make test-runtime-integration`

`test-runtime` runs the `runtime`-tagged tests. Before execution, `runtime-host-check` validates the current host Podman socket state.
`test-runtime-integration` runs the `runtime + integration` path under `unshare`. Before execution, `runtime-integration-host-check` verifies that `unshare` is available.

### 3. Remote clean-VM path

Runtime-dependent validation runs on an ephemeral Multipass VM created on `100.123.80.48`.

- target machine: `100.123.80.48`
- user: `seoy`
- default VM name: `podbridge5-dev`
- recommended OS: Ubuntu 24.04

Validated components include:

- `buildah`
- `fuse-overlayfs`
- `pkg-config` / `gpgme`
- `btrfs` headers
- system `podman` socket
- storage initialization
- image build / push flows

## Why use a clean VM

The key property is reproducibility.
If a failure only exists because of environment residue, it should disappear after deleting the VM and recreating it.
If it still reproduces on a fresh VM, the problem is more likely in code or runtime initialization.

## Makefile automation

The Makefile handles the remote runtime VM lifecycle automatically.

`make vm-test-runtime` does the following:

1. creates a fresh test VM on the remote machine
2. installs required packages and prepares the system `podman` socket
3. archives the current local `podbridge5` worktree and uploads it to the remote host
4. syncs that archive into the fresh VM with `multipass transfer`
5. runs `go test ./...` inside the VM
6. deletes the VM after the test finishes

`make vm-test-runtime-integration` follows the same lifecycle and runs the `runtime + integration` tag path.

## Log collection

Remote VM test output is shown in the console and also stored in local log files.

- `artifacts/vm-test-runtime.log`
- `artifacts/vm-test-runtime-integration.log`

The logs include the VM lifecycle, remote preparation steps, worktree sync, and `go test` stdout/stderr.

When the local runtime path fails, the preflight messages should be the first place to look.

- `runtime-env-check`: package prerequisites such as buildah, fuse-overlayfs, gpgme, and btrfs headers
- `runtime-host-check`: Podman socket or `XDG_RUNTIME_DIR` issues
- `runtime-integration-host-check`: `unshare` availability issues

## Common targets

- `make test-unit`
- `make runtime-env-check`
- `make test-runtime`
- `make test-runtime-integration`
- `make vm-test-runtime REMOTE_PASS=...`
- `make vm-test-runtime-integration REMOTE_PASS=...`
- `make vm-create-runtime REMOTE_PASS=...`
- `make vm-prepare-runtime REMOTE_PASS=...`
- `make vm-sync-runtime REMOTE_PASS=...`
- `make vm-delete-runtime REMOTE_PASS=...`

By default, the current local worktree (`$(CURDIR)`) is synced into the VM.
You can override it explicitly with `PODBRIDGE5_LOCAL_REPO=/path/to/repo`.
