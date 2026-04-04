
---

## `docs/resources/datastore_lvm.md`

```md
# datastore_lvm Resource

Creates a datastore backed by block devices through Procurator core.

## Example Usage

```terraform
resource "procurator_datastore_lvm" "example" {
  name = "fast-ssd"

  devices = [
    "sdb"
  ]
}