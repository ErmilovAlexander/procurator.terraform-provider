terraform {
  required_version = ">= 1.5.0"

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

variable "vm_name" {
  type    = string
  default = "tf-from-template-01"
}

variable "ssh_user" {
  type    = string
  default = "user"
}

variable "ssh_private_key_path" {
  type    = string
  default = ".ssh/id_ed25519"
}

data "procurator_datastore" "ds" {
  name = "DEV-STOR-0"
}

data "procurator_template" "tpl" {
  name = "ubuntu_serv"
}

resource "procurator_vm" "from_template" {
  name        = var.vm_name
  template_id = data.procurator_template.tpl.id
  storage_id  = data.procurator_datastore.ds.id
  power_state = "running"
}

resource "terraform_data" "configure_vm" {
  depends_on = [procurator_vm.from_template]

  triggers_replace = [
    procurator_vm.from_template.id,
    procurator_vm.from_template.guest_ip,
    procurator_vm.from_template.guest_dns_name,
  ]

  connection {
    type        = "ssh"
    host        = procurator_vm.from_template.guest_ip
    user        = var.ssh_user
    private_key = file(var.ssh_private_key_path)
    timeout     = "10m"
  }

  provisioner "remote-exec" {
    inline = [
      "echo 'Connected to $(hostname)'",
      "echo 'IP: ${procurator_vm.from_template.guest_ip}'",
      "sudo cloud-init status --wait || true",
      "sudo hostnamectl set-hostname ${var.vm_name}",
      "sudo apt-get update -y || true",
      "sudo mkdir -p /opt/bootstrap",
      "echo 'bootstrap done' | sudo tee /opt/bootstrap/status.txt"
    ]
  }
}

output "vm_id" {
  value = procurator_vm.from_template.id
}

output "vm_uuid" {
  value = procurator_vm.from_template.uuid
}

output "guest_ip" {
  value = procurator_vm.from_template.guest_ip
}

output "guest_dns_name" {
  value = procurator_vm.from_template.guest_dns_name
}