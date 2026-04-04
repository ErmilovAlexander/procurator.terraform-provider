
---

## `docs/data-sources/network.md`

```md
# network Data Source

Finds a network by ID or name.

## Example Usage

```terraform
data "procurator_network" "example" {
  name = "VLAN106"
}