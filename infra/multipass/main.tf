# This VM only owns pd5's runtime-validation host. It is intentionally kept
# outside infra-lab because it is not a Kubernetes cluster or shared platform
# resource.
resource "null_resource" "podbridge5_dev" {
  triggers = {
    remote_host = var.remote_host
    remote_user = var.remote_user
    ssh_config  = var.ssh_config
    vm_name     = var.vm_name
    image       = var.image
    cpus        = tostring(var.cpus)
    memory      = var.memory
    disk        = var.disk
  }

  provisioner "local-exec" {
    command = <<-EOT
      set -euo pipefail
      ssh -F '${var.ssh_config}' -o BatchMode=yes '${var.remote_user}@${var.remote_host}' \
        'multipass info "${var.vm_name}" >/dev/null 2>&1 || multipass launch "${var.image}" --name "${var.vm_name}" --cpus "${var.cpus}" --memory "${var.memory}" --disk "${var.disk}"'
    EOT
  }

  provisioner "local-exec" {
    when = destroy

    command = <<-EOT
      ssh -F '${self.triggers.ssh_config}' -o BatchMode=yes '${self.triggers.remote_user}@${self.triggers.remote_host}' \
        'multipass delete -p "${self.triggers.vm_name}" >/dev/null 2>&1 || true'
    EOT
  }
}

output "vm_name" {
  description = "Name of the persistent pd5 runtime-validation VM."
  value       = null_resource.podbridge5_dev.triggers.vm_name
}
