terraform {
  required_providers {
    procurator = {
      source  = "local/procurator/procurator"
      version = "0.1.0"
    }
  }
}

provider "procurator" {
  endpoint = "10.10.102.23:3641"
  username = "root"
  password = "P@ssw0rd"
  insecure = false
}

resource "procurator_vm_datastore_migration" "move_all_disks" {
  vm_id               = "ateycw1t"
  target_datastore_id = "2n3q4dtaw"
}