
---

## `docs/data-sources/vm.md`

```md
# vm Data Source

Finds a virtual machine by ID, deployment name, UUID, or name depending on backend matching logic.

## Example Usage

```terraform
data "procurator_vm" "example" {
  name = "vm-example"
}