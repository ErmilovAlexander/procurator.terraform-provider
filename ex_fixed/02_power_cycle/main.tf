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

resource "procurator_vm" "power_test" {
  name             = "tf-create-vm-01"
  storage_id       = data.procurator_datastore.ds.id
  power_state      = var.power_state
  power_force      = var.power_force
  vcpus            = 2
  max_vcpus        = 2
  core_per_socket  = 2
  memory_size_mb   = 4096
  cpu_model        = "host-model"
  cpu_hotplug      = false
  memory_hotplug   = false
  machine_type     = "pc-q35-6.2"

  disk_devices {
    bus            = "virtio"
    target         = "vda"
    size           = 30
    create         = false
    boot_order     = 1
    storage_id     = data.procurator_datastore.ds.id
    provision_type = "thin"
    read_only      = false
    disk_mode      = "dependent"
    device_type    = "disk"
  }

  network_devices {
    network    = "VLAN106"
    model      = "virtio"
    target     = "eth0"
    boot_order = 0
    vlan       = 0
  }

  boot_options {
    firmware      = "efi"
    boot_delay_ms = 1000
    boot_menu     = true
  }

  guest_tools {
    enabled           = true
    synchronized_time = true
  }
}

variable "power_state" {
  type    = string
  default = "running"
}

variable "power_force" {
  type    = bool
  default = false
}

output "vm_id" {
  value = procurator_vm.power_test.id
}

output "vm_power_state" {
  value = procurator_vm.power_test.power_state
}
