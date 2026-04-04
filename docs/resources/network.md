
---

## `docs/resources/network.md`

```md
# network Resource

Manages a network in Umbra.

## Example Usage

```terraform
resource "procurator_network" "example" {
  name      = "prod-net"
  vlan      = 120
  switch_id = "uSwitch0"
}