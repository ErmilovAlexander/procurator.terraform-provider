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

#data "procurator_switch" "one" {
#id = "uSwitch1"
#}

#output "switch" {
#value = data.procurator_switch.one
#}

resource "procurator_switch" "main" {
mtu = 1500
nics = {
active = ["enp24s0f1np1"]
standby = []
unused = []
inherit = false
}
}

output "switch_id" {
value = procurator_switch.main.id
}