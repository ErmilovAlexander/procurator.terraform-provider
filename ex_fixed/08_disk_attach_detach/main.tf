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

data "procurator_vm" "existing" {
  name = "tf-create-vm-01"
}

data "procurator_datastore" "ds" {
  name = "DEV-STOR-0"
}

resource "procurator_vm_disk_attachment" "extra_disk" {
  vm_id            = data.procurator_vm.existing.id
  size_gb          = 5
  storage_id       = data.procurator_datastore.ds.id
  device_type      = "disk"
  bus              = "virtio"
  target           = "vdb"
  boot_order       = 2
  provision_type   = "thin"
  disk_mode        = "dependent"
  read_only        = false
  remove_on_detach = true
}

output "vm_id" {
  value = data.procurator_vm.existing.id
}

output "disk_attachment_id" {
  value = procurator_vm_disk_attachment.extra_disk.id
}

output "disk_target" {
  value = procurator_vm_disk_attachment.extra_disk.target
}
