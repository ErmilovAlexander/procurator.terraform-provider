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

data "procurator_datastore" "ds" {
  name = "DEV-STOR-0"
}

data "procurator_template" "tpl" {
  name = "tf-create-vm-01"
}

resource "procurator_vm" "from_template" {
  name            = "tf-from-template-01"
  template_id     = data.procurator_template.tpl.id
  storage_id      = data.procurator_datastore.ds.id
  power_state     = "stopped"
}

output "vm_id" {
  value = procurator_vm.from_template.id
}

output "vm_uuid" {
  value = procurator_vm.from_template.uuid
}