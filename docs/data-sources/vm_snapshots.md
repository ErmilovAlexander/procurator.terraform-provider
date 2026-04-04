
---

## `docs/data-sources/vm_snapshots.md`

```md
# vm_snapshots Data Source

Returns snapshots for a VM.

## Example Usage

```terraform
data "procurator_vm_snapshots" "example" {
  vm_id = "VM_ID"
}