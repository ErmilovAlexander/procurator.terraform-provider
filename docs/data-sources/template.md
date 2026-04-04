
---

## `docs/data-sources/template.md`

```md
# template Data Source

Finds a template by ID, UUID, or name.

## Example Usage

```terraform
data "procurator_template" "example" {
  name = "base-template"
}