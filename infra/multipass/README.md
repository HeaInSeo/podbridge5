# pd5 runtime validation VM

This HCL owns only the persistent `podbridge5-dev` Multipass VM used for pd5
runtime validation. It deliberately does not live in `infra-lab`: the VM is a
project-specific Buildah test host, not a Kubernetes cluster resource.

The VM is created on the configured remote Multipass host through SSH. Its
default target is `seoy@100.123.80.48`.

```bash
cd infra/multipass
tofu init
tofu apply
```

The HCL only creates and deletes the VM. Prepare, synchronize, and test the
current pd5 worktree using the existing Make targets; unlike `vm-test-runtime`,
these commands leave the VM running:

```bash
make vm-prepare-runtime REMOTE_USER=seoy
make vm-sync-runtime REMOTE_USER=seoy
make vm-run-runtime REMOTE_USER=seoy
```

Destroy the persistent VM only when it is no longer needed:

```bash
tofu destroy
```

Do not use `make vm-test-runtime` with this HCL-managed VM: that compatibility
target creates and then deletes `podbridge5-dev` as part of its cleanup flow.
