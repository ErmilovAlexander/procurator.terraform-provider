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

resource "procurator_vm_network_attachment" "nic2" {
  vm_id      = data.procurator_vm.existing.id
  network    = "VLAN106"
  target     = "eth1"
  model      = "virtio"
  boot_order = 10
  vlan       = 0
}

output "vm_id" {
  value = data.procurator_vm.existing.id
}

output "network_attachment_id" {
  value = procurator_vm_network_attachment.nic2.id
}

output "nic_target" {
  value = procurator_vm_network_attachment.nic2.target
}

output "nic_mac" {
  value = procurator_vm_network_attachment.nic2.mac
}
