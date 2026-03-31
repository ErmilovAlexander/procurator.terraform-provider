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

data "procurator_networks" "all" {}

output "network_names" {
  value = [for n in data.procurator_networks.all.networks : n.name]
}