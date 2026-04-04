
---

## `docs/resources/vm_snapshot.md`

```md
# vm_snapshot Resource

Creates and manages a VM snapshot.

## Example Usage

```terraform
resource "procurator_vm_snapshot" "example" {
  vm_id          = "VM_ID"
  name           = "snap-01"
  description    = "snapshot created by terraform"
  include_memory = false
  quiesce_fs     = false
}