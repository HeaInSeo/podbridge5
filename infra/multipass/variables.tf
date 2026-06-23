variable "remote_host" {
  description = "Host that runs Multipass."
  type        = string
  default     = "100.123.80.48"

  validation {
    condition     = can(regex("^[a-zA-Z0-9][a-zA-Z0-9.-]*$", var.remote_host))
    error_message = "remote_host must be a hostname or IP address without shell metacharacters."
  }
}

variable "remote_user" {
  description = "SSH user on remote_host."
  type        = string
  default     = "seoy"

  validation {
    condition     = can(regex("^[a-z_][a-z0-9_-]*$", var.remote_user))
    error_message = "remote_user must be a valid Linux user name."
  }
}

variable "ssh_config" {
  description = "Optional SSH config file. /dev/null disables user-level SSH configuration."
  type        = string
  default     = "/dev/null"

  validation {
    condition     = can(regex("^/[a-zA-Z0-9_./-]+$", var.ssh_config))
    error_message = "ssh_config must be an absolute path without shell metacharacters."
  }
}

variable "vm_name" {
  description = "Name of the persistent Multipass validation VM."
  type        = string
  default     = "podbridge5-dev"

  validation {
    condition     = can(regex("^[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$", var.vm_name))
    error_message = "vm_name must contain only alphanumeric characters or '-', and must not start or end with '-'."
  }
}

variable "image" {
  description = "Multipass image used by the validation VM."
  type        = string
  default     = "24.04"

  validation {
    condition     = can(regex("^[a-zA-Z0-9._/-]+$", var.image))
    error_message = "image must not contain shell metacharacters."
  }
}

variable "cpus" {
  description = "vCPU count for the validation VM."
  type        = number
  default     = 2

  validation {
    condition     = var.cpus >= 1 && floor(var.cpus) == var.cpus
    error_message = "cpus must be an integer >= 1."
  }
}

variable "memory" {
  description = "Memory for the validation VM, for example 4G."
  type        = string
  default     = "4G"

  validation {
    condition     = can(regex("^[1-9][0-9]*[MG]$", var.memory))
    error_message = "memory must look like 4G or 4096M."
  }
}

variable "disk" {
  description = "Disk capacity for the validation VM, for example 20G."
  type        = string
  default     = "20G"

  validation {
    condition     = can(regex("^[1-9][0-9]*[MG]$", var.disk))
    error_message = "disk must look like 20G or 20480M."
  }
}
