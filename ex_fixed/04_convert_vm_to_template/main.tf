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

data "procurator_vm" "source_vm" {
  name = "tf-create-vm-01"
}

resource "procurator_vm_convert_to_template" "convert" {
  vm_id = data.procurator_vm.source_vm.id
}

data "procurator_template" "converted" {
  depends_on = [procurator_vm_convert_to_template.convert]
  name       = data.procurator_vm.source_vm.name
}

output "template_id" {
  value = data.procurator_template.converted.id
}

output "template_uuid" {
  value = data.procurator_template.converted.uuid
}