# Runtime Validation

This document describes the runtime validation path for `podbridge5`.
The main README stays focused on project description, while the VM-based validation flow is documented here.

## Validation paths

Validation is split into two paths.

### 1. Local base path

This is the fast path on the current host.

- `make test`
- `make test-runtime`
- `make test-runtime-integration`
- `make runtime-env-check`

This path is useful for code changes and lightweight validation.
Since the repository still depends on `buildah`, `containers/storage`, and `containers/image`, a full test run may still require host packages to be installed.

### 2. Remote clean-VM path

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

`make vm-test-runtime-integration` follows the same lifecycle and runs the integration-tag path.

## Log collection

Remote VM test output is shown in the console and also stored in local log files.

- `artifacts/vm-test-runtime.log`
- `artifacts/vm-test-runtime-integration.log`

The logs include the VM lifecycle, remote preparation steps, worktree sync, and `go test` stdout/stderr.

## Common targets

- `make vm-test-runtime REMOTE_PASS=...`
- `make vm-test-runtime-integration REMOTE_PASS=...`
- `make vm-create-runtime REMOTE_PASS=...`
- `make vm-prepare-runtime REMOTE_PASS=...`
- `make vm-sync-runtime REMOTE_PASS=...`
- `make vm-delete-runtime REMOTE_PASS=...`

By default, the current local worktree (`$(CURDIR)`) is synced into the VM.
You can override it explicitly with `PODBRIDGE5_LOCAL_REPO=/path/to/repo`.
