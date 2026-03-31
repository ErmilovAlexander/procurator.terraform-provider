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

data "procurator_storage_adapters" "all" {}

data "procurator_storage_devices" "all" {}

output "storage_adapters" {
value = data.procurator_storage_adapters.all.items
}

output "storage_devices" {
value = data.procurator_storage_devices.all.items
}