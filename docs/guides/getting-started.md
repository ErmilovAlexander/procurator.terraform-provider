
---

## `docs/guides/getting-started.md`

```md
---
page_title: "Getting Started"
subcategory: "Guides"
---

# Getting Started

## Example Provider Configuration

```terraform
terraform {
  required_providers {
    procurator = {
      source  = "ErmilovAlexander/procurator"
      version = "0.1.0"
    }
  }
}

provider "procurator" {
  endpoint         = "10.10.102.22:3641"
  umbra_endpoint   = "10.10.102.22:50051"
  storage_endpoint = "10.10.102.22:3642"

  token     = var.token
  ca_file   = var.ca_file
  authority = var.authority
}