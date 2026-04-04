
---

## `docs/resources/vm.md`

```md
# vm Resource

Manages a virtual machine.

## Example Usage

### Create from scratch

```terraform
resource "procurator_vm" "example" {
  name            = "vm-example"
  storage_id      = "DATASTORE_ID"
  power_state     = "stopped"
  vcpus           = 2
  max_vcpus       = 2
  core_per_socket = 1
  memory_size_mb  = 4096
  cpu_model       = "host-model"
  machine_type    = "pc-q35-6.2"

  disk_devices {
    bus            = "virtio"
    target         = "vda"
    size           = 30
    create         = true
    boot_order     = 1
    storage_id     = "DATASTORE_ID"
    provision_type = "thin"
    read_only      = false
    disk_mode      = "dependent"
    device_type    = "disk"
  }

  network_devices {
    network    = "VLAN106"
    model      = "virtio"
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