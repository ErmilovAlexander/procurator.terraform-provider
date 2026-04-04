---
page_title: "Procurator Provider"
---

# Procurator Provider

The Procurator provider manages virtualization resources in Procurator over gRPC APIs.

The provider currently supports:
- virtual machines
- templates
- datastores
- datastore folders
- switches
- networks
- VM snapshots
- VM disk attachments
- VM network attachments
- VM datastore migration
- inventory data sources for host, VM, template, datastore, network, switch, NIC, and storage

## Example Usage

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