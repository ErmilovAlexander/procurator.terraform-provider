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

resource "procurator_network" "prod_vlan120" {
name = "prod-vlan120"
vlan = 120
switch_id = "uSwitch1"
}

output "network_id" {
value = procurator_network.prod_vlan120.id
}