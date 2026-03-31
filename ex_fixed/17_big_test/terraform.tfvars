endpoint         = "10.10.102.22:3641"
#umbra_endpoint   = "10.10.102.22:50051"
#storage_endpoint = "10.10.102.22:3642"
username = "root"
password = "P@ssw0rd"
#token     = "YOUR_TOKEN"
#ca_file   = "/path/to/ca.pem"
#authority = "127.0.0.1"

existing_datastore_name   = "DatastoreHDD_ISCSI"
existing_vm_name          = "tf-create-vm-01"
test_datastore_lvm_name   = "tf-data-lvm-01"
existing_template_name    = ""
existing_network_name     = "VLAN106"
existing_switch_id        = ""
nic_name                  = "ens785f1np1"
lvm_device_name           = "sdc"
migration_target_datastore_id = "2ctu1ywrt"
folder_path_suffix      = "/tf-smoke-folder"

test_vm_name            = "tf-create-vm-02"
test_vm_disk_size_gb    = 30
test_vm_memory_size_mb  = 4096
test_vm_vcpus           = 2

enable_inventory             = false
enable_switch_create         = false
enable_network_create        = false
enable_datastore_lvm_create  = false
enable_datastore_folder      = false
enable_vm_create             = true
enable_vm_snapshot           = false
enable_vm_disk_attachment    = false
enable_vm_network_attachment = false
enable_vm_convert_to_template = false
enable_vm_deploy_from_template = false
enable_vm_datastore_migration = true
