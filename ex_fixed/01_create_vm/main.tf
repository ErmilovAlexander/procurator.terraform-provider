terraform {
  required_providers {
    procurator = {
      source  = "local/procurator/procurator"
      version = "0.1.0"
    }
  }
}

provider "procurator" {
  endpoint = "10.10.102.22:3641"
  username = "root"
  password = "P@ssw0rd"
  insecure = false
}

data "procurator_datastore" "ds" {
  name = "DatastoreHDD_ISCSI"
}

resource "procurator_vm" "test_vm" {
  count = 1 
  name             = "tf-create-vm-0${count.index + 1}"
  storage_id       = data.procurator_datastore.ds.id
  power_state      = "stopped"
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
    size           = 10
    create         = true
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

output "vm_id" {
  value = [for vm in procurator_vm.test_vm : vm.id]
}

output "vm_uuid" {
  value = [for vm in procurator_vm.test_vm : vm.uuid]
}