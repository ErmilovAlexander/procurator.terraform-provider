terraform {
  required_providers {
    procurator = {
      source  = "local/procurator/procurator"
      version = "0.1.0"
    }
  }
}

provider "procurator" {
  endpoint = "10.10.102.23"
  username = "root"
  password = "P@ssw0rd"
  insecure = false
}

data "procurator_datastore" "ds" {
  name = "DEV-STOR-0"
}

resource "procurator_vm" "resize_test" {
  name            = "tf-create-vm-01"
  storage_id      = data.procurator_datastore.ds.id
  power_state     = "stopped"
  vcpus           = var.vcpus
  max_vcpus       = var.max_vcpus
  core_per_socket = var.core_per_socket
  memory_size_mb  = var.memory_size_mb
  machine_type    = "pc-q35-6.2"
  cpu_model       = "host-model"
  cpu_hotplug     = false
  memory_hotplug  = false
}

variable "vcpus" {
  type    = number
  default = 2
}

variable "max_vcpus" {
  type    = number
  default = 2
}

variable "core_per_socket" {
  type    = number
  default = 1
}

variable "memory_size_mb" {
  type    = number
  default = 4096
}