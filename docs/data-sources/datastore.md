
---

## `docs/data-sources/datastore.md`

```md
# datastore Data Source

Finds a datastore by name or ID.

## Example Usage

```terraform
data "procurator_datastore" "example" {
  name = "DEV-STOR-0"
}