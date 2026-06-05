#!/usr/bin/env bash
# Sets up a persistent dev VM on the remote host for manual buildah/podman investigation.
# For automated test runs use 'make vm-test-runtime' instead — it manages its own ephemeral VM.
set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-100.123.80.48}"
REMOTE_USER="${REMOTE_USER:-seoy}"
REMOTE_PORT="${REMOTE_PORT:-22}"
VM_NAME="${PODBRIDGE5_VM_NAME:-podbridge5-dev}"
VM_CPUS="${PODBRIDGE5_VM_CPUS:-2}"
VM_MEMORY="${PODBRIDGE5_VM_MEMORY:-4G}"
VM_DISK="${PODBRIDGE5_VM_DISK:-20G}"

SSH_OPTS=(-F /dev/null -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p "${REMOTE_PORT}")
[[ -n "${REMOTE_PASS:-}" ]] && SSH_OPTS+=(sshpass -p "${REMOTE_PASS}")

remote() {
  ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" "$@"
}

echo "==> checking ${VM_NAME} on ${REMOTE_USER}@${REMOTE_HOST}"

if remote "multipass info '${VM_NAME}'" >/dev/null 2>&1; then
  echo "    VM already exists — skipping creation"
else
  echo "==> launching ${VM_NAME} (cpus=${VM_CPUS} mem=${VM_MEMORY} disk=${VM_DISK})"
  remote "multipass launch 24.04 \
    --name '${VM_NAME}' \
    --cpus '${VM_CPUS}' \
    --memory '${VM_MEMORY}' \
    --disk '${VM_DISK}'"
fi

echo "==> installing packages"
remote "multipass exec '${VM_NAME}' -- sudo apt-get update -q"
remote "multipass exec '${VM_NAME}' -- sudo apt-get install -y -q \
  buildah podman fuse-overlayfs slirp4netns uidmap"

echo "==> verifying rootless setup"
remote "multipass exec '${VM_NAME}' -- bash -c '
  echo \"--- subuid ---\" && cat /etc/subuid
  echo \"--- subgid ---\" && cat /etc/subgid
  echo \"--- buildah version ---\" && buildah version
  echo \"--- rootless smoke ---\" && buildah images || true
'"

echo ""
echo "VM ${VM_NAME} is ready."
echo "  ssh in:  ssh ${REMOTE_USER}@${REMOTE_HOST} then: multipass shell ${VM_NAME}"
echo "  delete:  ssh ${REMOTE_USER}@${REMOTE_HOST} then: multipass delete --purge ${VM_NAME}"
