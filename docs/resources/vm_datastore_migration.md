
---

## `docs/resources/vm_datastore_migration.md`

```md
# vm_datastore_migration Resource

Migrates a VM to another datastore.

## Example Usage

### Migrate all VM disks

```terraform
resource "procurator_vm_datastore_migration" "example" {
  vm_id               = "VM_ID"
  target_datastore_id = "DATASTORE_ID"
}