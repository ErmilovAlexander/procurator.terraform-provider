
---

## `docs/resources/datastore_folder.md`

```md
# datastore_folder Resource

Creates or manages a folder inside a datastore.

## Example Usage

```terraform
resource "procurator_datastore_folder" "images" {
  path = "DATASTORE_ID:/images"
}