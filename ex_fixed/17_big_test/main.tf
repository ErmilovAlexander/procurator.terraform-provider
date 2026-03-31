terraform {
required_providers {
procurator = {
source  = "local/procurator/procurator"
version = "0.1.0"
}
}
}

############################################

# PROVIDER

############################################

provider "procurator" {
endpoint         = var.endpoint
#umbra_endpoint   = var.umbra_endpoint
#storage_endpoint = var.storage_endpoint

username   = var.username
password   = var.password
insecure   = var.insecure
#ca_file    = var.ca_file
#authority  = var.authority
}

############################################

# VARIABLES

############################################

variable "endpoint" {
type = string
}

#variable "umbra_endpoint" {
#type = string
#}

#variable "storage_endpoint" {
#type = string
#}

#variable "token" {
#type    = string
#default = ""
#}

variable "username" {
type    = string
default = ""
}

variable "password" {
type    = string
default = ""
sensitive = true
}

variable "insecure" {
type    = bool
default = false
}

#variable "ca_file" {
#type    = string
#default = ""
#}

#variable "authority" {
#type    = string
#default = ""
#}

############################################

# GLOBAL INPUTS

############################################

variable "existing_datastore_name" {
type        = string
description = "Existing datastore name used for VM tests and generic reads."
}

variable "existing_vm_name" {
type        = string
default     = ""
description = "Existing VM name used for snapshot, disk attach, NIC attach, migration, convert-to-template."
}

variable "existing_template_name" {
type        = string
default     = ""
description = "Existing template name used for deploy-from-template."
}

variable "existing_network_name" {
type        = string
default     = ""
description = "Existing network name used for VM NIC attach or VM create when no test network is created."
}

variable "existing_switch_id" {
type        = string
default     = ""
description = "Existing switch ID used when testing network creation against a pre-existing switch."
}

variable "nic_name" {
type        = string
default     = ""
description = "NIC name from data.procurator_nics.all.nics[*].name for switch create test."
}

variable "lvm_device_name" {
type        = string
default     = ""
description = "Free storage device name for datastore LVM test. Prefer the 'name' from data.procurator_storage_devices.all.items[*].name, e.g. sdb."
}

variable "migration_target_datastore_id" {
type        = string
default     = ""
description = "Target datastore ID for VM datastore migration test."
}

variable "folder_path_suffix" {
type        = string
default     = "/tf-smoke-folder"
description = "Folder suffix appended to datastore_id for datastore folder test."
}

############################################

# FEATURE TOGGLES

############################################

variable "enable_inventory" {
type    = bool
default = true
}

variable "enable_switch_create" {
type    = bool
default = false
}

variable "enable_network_create" {
type    = bool
default = false
}

variable "enable_datastore_lvm_create" {
type    = bool
default = false
}

variable "enable_datastore_folder" {
type    = bool
default = false
}

variable "enable_vm_create" {
type    = bool
default = false
}

variable "enable_vm_snapshot" {
type    = bool
default = false
}

variable "enable_vm_disk_attachment" {
type    = bool
default = false
}

variable "enable_vm_network_attachment" {
type    = bool
default = false
}

variable "enable_vm_convert_to_template" {
type    = bool
default = false
}

variable "enable_vm_deploy_from_template" {
type    = bool
default = false
}

variable "enable_vm_datastore_migration" {
type    = bool
default = false
}

############################################

# TEST VALUES

############################################

variable "test_switch_mtu" {
type    = number
default = 1500
}

variable "test_network_name" {
type    = string
default = "tf-net-120"
}

variable "test_network_vlan" {
type    = number
default = 120
}

variable "test_datastore_lvm_name" {
type    = string
default = "tf-data-lvm-01"
}

variable "test_vm_name" {
type    = string
default = "tf-create-vm-01"
}

variable "test_vm_power_state" {
type    = string
default = "stopped"
}

variable "test_vm_vcpus" {
type    = number
default = 2
}

variable "test_vm_max_vcpus" {
type    = number
default = 2
}

variable "test_vm_core_per_socket" {
type    = number
default = 1
}

variable "test_vm_memory_size_mb" {
type    = number
default = 4096
}

variable "test_vm_cpu_model" {
type    = string
default = "host-model"
}

variable "test_vm_machine_type" {
type    = string
default = "pc-q35-6.2"
}

variable "test_vm_disk_size_gb" {
type    = number
default = 30
}

variable "test_vm_network_model" {
type    = string
default = "virtio"
}

variable "test_snapshot_name" {
type    = string
default = "tf-snap-01"
}

variable "test_snapshot_description" {
type    = string
default = "terraform snapshot smoke test"
}

variable "test_attach_disk_target" {
type    = string
default = "vdb"
}

variable "test_attach_disk_size_gb" {
type    = number
default = 5
}

variable "test_attach_nic_target" {
type    = string
default = "eth1"
}

variable "test_deployed_vm_name" {
type    = string
default = "tf-from-template-01"
}

############################################

# LOCALS

############################################

locals {
#use_token_auth = trimspace(var.token) != ""

created_switch_id = try(procurator_switch.test[0].id, null)
switch_id_for_network = local.created_switch_id != null ? local.created_switch_id : (
trimspace(var.existing_switch_id) != "" ? var.existing_switch_id : null
)

created_network_name = try(procurator_network.test[0].name, null)
network_name_for_vm = local.created_network_name != null ? local.created_network_name : (
trimspace(var.existing_network_name) != "" ? var.existing_network_name : null
)

created_datastore_id = try(procurator_datastore_lvm.test[0].id, null)
datastore_id_for_folder = local.created_datastore_id != null ? local.created_datastore_id : try(data.procurator_datastore.base.id, null)

created_vm_id = try(procurator_vm.test[0].id, null)
vm_id_for_ops = local.created_vm_id != null ? local.created_vm_id : (
trimspace(var.existing_vm_name) != "" ? try(data.procurator_vm.existing[0].id, null) : null
)

template_id_for_deploy = try(data.procurator_template.converted[0].id, null) != null ? try(data.procurator_template.converted[0].id, null) : (
trimspace(var.existing_template_name) != "" ? try(data.procurator_template.existing[0].id, null) : null
)

datastore_id_for_vm = try(data.procurator_datastore.base.id, null)

folder_full_path = local.datastore_id_for_folder != null ? "${local.datastore_id_for_folder}:${var.folder_path_suffix}" : null
}

############################################

# READ-ONLY INVENTORY

############################################

data "procurator_host" "host" {
count = var.enable_inventory ? 1 : 0
}

data "procurator_datastore" "base" {
name = var.existing_datastore_name
}

data "procurator_storage_adapters" "all" {
count = var.enable_inventory ? 1 : 0
}

data "procurator_storage_devices" "all" {
count = var.enable_inventory ? 1 : 0
}

data "procurator_nics" "all" {
count = var.enable_inventory ? 1 : 0
}

data "procurator_switches" "all" {
count = var.enable_inventory ? 1 : 0
}

data "procurator_networks" "all" {
count = var.enable_inventory ? 1 : 0
}

data "procurator_vm" "existing" {
count = trimspace(var.existing_vm_name) != "" ? 1 : 0
name  = var.existing_vm_name
}

data "procurator_template" "existing" {
count = trimspace(var.existing_template_name) != "" ? 1 : 0
name  = var.existing_template_name
}

data "procurator_switch" "inspect" {
count = trimspace(var.existing_switch_id) != "" ? 1 : 0
id    = var.existing_switch_id
}

data "procurator_network" "inspect_by_name" {
count = trimspace(var.existing_network_name) != "" ? 1 : 0
name  = var.existing_network_name
}

############################################

# SWITCH

############################################

resource "procurator_switch" "test" {
count = var.enable_switch_create ? 1 : 0

mtu = var.test_switch_mtu

nics = {
active  = trimspace(var.nic_name) != "" ? [var.nic_name] : []
standby = []
unused  = []
inherit = false
}
}

############################################

# NETWORK

############################################

resource "procurator_network" "test" {
count = var.enable_network_create ? 1 : 0

name      = var.test_network_name
vlan      = var.test_network_vlan
switch_id = local.switch_id_for_network
}

############################################

# DATASTORE LVM THROUGH CORE

############################################

resource "procurator_datastore_lvm" "test" {
count = var.enable_datastore_lvm_create ? 1 : 0

name    = var.test_datastore_lvm_name
devices = [var.lvm_device_name]
}

############################################

# DATASTORE FOLDER

############################################

resource "procurator_datastore_folder" "test" {
count = var.enable_datastore_folder ? 1 : 0

path = local.folder_full_path
}

############################################

# VM CREATE

############################################

resource "procurator_vm" "test" {
count = var.enable_vm_create ? 1 : 0

name            = var.test_vm_name
storage_id      = local.datastore_id_for_vm
power_state     = var.test_vm_power_state
vcpus           = var.test_vm_vcpus
max_vcpus       = var.test_vm_max_vcpus
core_per_socket = var.test_vm_core_per_socket
memory_size_mb  = var.test_vm_memory_size_mb
cpu_model       = var.test_vm_cpu_model
cpu_hotplug     = false
memory_hotplug  = false
machine_type    = var.test_vm_machine_type

disk_devices {
bus            = "virtio"
target         = "vda"
size           = var.test_vm_disk_size_gb
create         = true
boot_order     = 1
storage_id     = local.datastore_id_for_vm
provision_type = "thin"
read_only      = false
disk_mode      = "dependent"
device_type    = "disk"
}

network_devices {
network    = local.network_name_for_vm
model      = var.test_vm_network_model
boot_order = 0
vlan       = 0
}

boot_options {
firmware      = "efi"
boot_delay_ms = 1000
boot_menu     = true
}

guest_tools {
enabled           = true
synchronized_time = true
}
}

############################################

# SNAPSHOT

############################################

resource "procurator_vm_snapshot" "test" {
count = var.enable_vm_snapshot ? 1 : 0

vm_id          = local.vm_id_for_ops
name           = var.test_snapshot_name
description    = var.test_snapshot_description
include_memory = false
quiesce_fs     = false
}

data "procurator_vm_snapshots" "test" {
count      = var.enable_vm_snapshot ? 1 : 0
depends_on = [procurator_vm_snapshot.test]
vm_id      = local.vm_id_for_ops
}

############################################

# DISK ATTACH / DETACH

############################################

resource "procurator_vm_disk_attachment" "test" {
count = var.enable_vm_disk_attachment ? 1 : 0

vm_id            = local.vm_id_for_ops
size_gb          = var.test_attach_disk_size_gb
storage_id       = local.datastore_id_for_vm
device_type      = "disk"
bus              = "virtio"
target           = var.test_attach_disk_target
boot_order       = 2
provision_type   = "thin"
disk_mode        = "dependent"
read_only        = false
remove_on_detach = true
}

############################################

# NETWORK ATTACH / DETACH

############################################

resource "procurator_vm_network_attachment" "test" {
count = var.enable_vm_network_attachment ? 1 : 0

vm_id      = local.vm_id_for_ops
network    = local.network_name_for_vm
target     = var.test_attach_nic_target
model      = var.test_vm_network_model
boot_order = 10
vlan       = 0
}

############################################

# VM -> TEMPLATE

############################################

resource "procurator_vm_convert_to_template" "test" {
count = var.enable_vm_convert_to_template ? 1 : 0

vm_id = local.vm_id_for_ops
}

data "procurator_template" "converted" {
count      = var.enable_vm_convert_to_template ? 1 : 0
depends_on = [procurator_vm_convert_to_template.test]
name       = var.enable_vm_create ? var.test_vm_name : var.existing_vm_name
}

############################################

# DEPLOY FROM TEMPLATE

############################################

resource "procurator_vm" "from_template" {
count = var.enable_vm_deploy_from_template ? 1 : 0

name        = var.test_deployed_vm_name
template_id = local.template_id_for_deploy
storage_id  = local.datastore_id_for_vm
power_state = "stopped"
}

############################################

# VM DATASTORE MIGRATION

############################################

resource "procurator_vm_datastore_migration" "test" {
count = var.enable_vm_datastore_migration ? 1 : 0

vm_id               = local.vm_id_for_ops
target_datastore_id = var.migration_target_datastore_id
}

############################################

# OUTPUTS

############################################

output "inventory_host" {
value = try(data.procurator_host.host[0], null)
}

output "inventory_storage_adapters" {
value = try(data.procurator_storage_adapters.all[0].items, [])
}

output "inventory_storage_devices" {
value = try(data.procurator_storage_devices.all[0].items, [])
}

output "inventory_nics" {
value = try(data.procurator_nics.all[0].nics, [])
}

output "inventory_switches" {
value = try(data.procurator_switches.all[0].switches, [])
}

output "inventory_networks" {
value = try(data.procurator_networks.all[0].networks, [])
}

output "base_datastore" {
value = data.procurator_datastore.base
}

output "created_switch_id" {
value = try(procurator_switch.test[0].id, null)
}

output "created_network_id" {
value = try(procurator_network.test[0].id, null)
}

output "created_network_name" {
value = try(procurator_network.test[0].name, null)
}

output "created_datastore_lvm_id" {
value = try(procurator_datastore_lvm.test[0].id, null)
}

output "created_datastore_lvm_pool_name" {
value = try(procurator_datastore_lvm.test[0].pool_name, null)
}

output "created_folder_path" {
value = try(procurator_datastore_folder.test[0].path, null)
}

output "created_vm_id" {
value = try(procurator_vm.test[0].id, null)
}

output "created_vm_uuid" {
value = try(procurator_vm.test[0].uuid, null)
}

output "snapshot_resource_id" {
value = try(procurator_vm_snapshot.test[0].id, null)
}

output "snapshot_numeric_id" {
value = try(procurator_vm_snapshot.test[0].snapshot_id, null)
}

output "snapshot_names" {
value = try([for s in data.procurator_vm_snapshots.test[0].items : s.name], [])
}

output "disk_attachment_id" {
value = try(procurator_vm_disk_attachment.test[0].id, null)
}

output "network_attachment_id" {
value = try(procurator_vm_network_attachment.test[0].id, null)
}

output "network_attachment_mac" {
value = try(procurator_vm_network_attachment.test[0].mac, null)
}

output "converted_template_id" {
value = try(data.procurator_template.converted[0].id, null)
}

output "deployed_vm_id" {
value = try(procurator_vm.from_template[0].id, null)
}

output "migration_task_id" {
value = try(procurator_vm_datastore_migration.test[0].task_id, null)
}

output "migration_final_datastore_id" {
value = try(procurator_vm_datastore_migration.test[0].final_datastore_id, null)
}
