
---

## `docs/resources/vm_disk_attachment.md`

```md
# vm_disk_attachment Resource

Attaches an additional virtual disk to an existing VM.

## Example Usage

```terraform
resource "procurator_vm_disk_attachment" "example" {
  vm_id            = "VM_ID"
  size_gb          = 10
  storage_id       = "DATASTORE_ID"
  device_type      = "disk"
  bus              = "virtio"
  target           = "vdb"
  boot_order       = 2
  provision_type   = "thin"
  disk_mode        = "dependent"
  read_only        = false
  remove_on_detach = true
}