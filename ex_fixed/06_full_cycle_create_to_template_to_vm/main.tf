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

resource "procurator_vm" "source" {
  name             = "tf-full-cycle-source"
  storage_id       = data.procurator_datastore.ds.id
  power_state      = "stopped"
  vcpus            = 2
  max_vcpus        = 2
  core_per_socket  = 1
  memory_size_mb   = 2048
  cpu_model        = "host-model"
  cpu_hotplug      = false
  memory_hotplug   = false
  machine_type     = "pc-q35-6.2"

  disk_devices {
    bus            = "virtio"
    target         = "vda"
    size           = 20
    create         = true
    boot_order     = 1
    storage_id     = data.procurator_datastore.ds.id
    provision_type = "thin"
    read_only      = false
    disk_mode      = "persistent"
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
}

resource "procurator_vm_convert_to_template" "convert" {
  depends_on = [procurator_vm.source]
  vm_id      = procurator_vm.source.id
}

data "procurator_template" "converted" {
  depends_on = [procurator_vm_convert_to_template.convert]
  name       = procurator_vm.source.name
}

resource "procurator_vm" "deployed" {
  name            = "tf-full-cycle-deployed"
  template_id     = data.procurator_template.converted.id
  storage_id      = data.procurator_datastore.ds.id
  power_state     = "stopped"
  vcpus           = 2
  max_vcpus       = 2
  core_per_socket = 1
  memory_size_mb  = 2048
}

output "source_vm_id" {
  value = procurator_vm.source.id
}

output "template_id" {
  value = data.procurator_template.converted.id
}

output "deployed_vm_id" {
  value = procurator_vm.deployed.id
}