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
data "procurator_nics" "all" {}

output "nics" {
value = data.procurator_nics.all.nics
}
