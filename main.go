package main

//go:generate terraform fmt -recursive ./examples/
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

import (
	"context"
	"log"
	"os"

	"terraform-provider-packer/packer_interop"
	"terraform-provider-packer/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/hashicorp/packer"
)

func main() {
	if os.Getenv(packer_interop.TPPRunPacker) == "true" {
		os.Exit(packer.Main(os.Args[1:]))
	} else {
		if err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
			Address: "registry.terraform.io/toowoxx/packer",
		}); err != nil {
			log.Fatal(err)
		}
	}
}
