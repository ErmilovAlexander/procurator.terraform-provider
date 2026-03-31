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

resource "procurator_vm_snapshot" "snap" {
  vm_id          = data.procurator_vm.existing.id
  name           = "tf-snap-01"
  description    = "terraform snapshot test"
  include_memory = false
  quiesce_fs     = false
}

data "procurator_vm_snapshots" "all" {
  depends_on = [procurator_vm_snapshot.snap]
  vm_id      = data.procurator_vm.existing.id
}

output "vm_id" {
  value = data.procurator_vm.existing.id
}

output "snapshot_resource_id" {
  value = procurator_vm_snapshot.snap.id
}

output "snapshot_numeric_id" {
  value = procurator_vm_snapshot.snap.snapshot_id
}

output "current_snapshot_id" {
  value = data.procurator_vm_snapshots.all.current_id
}

output "snapshot_names" {
  value = [for s in data.procurator_vm_snapshots.all.items : s.name]
}
