
---

## `docs/guides/authentication.md`

```md
---
page_title: "Authentication"
subcategory: "Guides"
---

# Authentication

The Procurator provider communicates with Procurator gRPC endpoints over TLS.

## Token authentication

```terraform
provider "procurator" {
  endpoint   = "10.10.102.22:3641"
  token      = var.token
  ca_file    = var.ca_file
  authority  = var.authority
}