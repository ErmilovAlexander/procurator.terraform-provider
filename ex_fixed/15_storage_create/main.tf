terraform {
  required_providers {
    procurator = {
      source  = "local/procurator/procurator"
      version = "0.1.0"
    }
  }
}

provider "procurator" {
  endpoint = "10.10.102.22"
  username = "root"
  password = "P@ssw0rd"
  insecure = false
}

resource "procurator_datastore_lvm" "data1" {
name = "data1TFTEST"

devices = [
"/dev/sdb"
]
}

output "datastore_id" {
value = procurator_datastore_lvm.data1.id
}

output "pool_name" {
value = procurator_datastore_lvm.data1.pool_name
}