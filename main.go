package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"procurator.terraform-provider/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New("dev"), providerserver.ServeOpts{
		Address: "registry.terraform.io/local/procurator",
	})
	if err != nil {
		log.Fatal(err)
	}
}
