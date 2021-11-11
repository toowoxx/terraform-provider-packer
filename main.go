package main

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

import (
	"context"
	"log"
	"os"

	"terraform-provider-packer/provider"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/hashicorp/packer"
)

func main() {
	if os.Getenv("TPP_RUN_PACKER") == "true" {
		os.Exit(packer.Main(os.Args[1:]))
	} else {
		if err := tfsdk.Serve(context.Background(), provider.New, tfsdk.ServeOpts{
			Name: "packer",
		}); err != nil {
			log.Fatal(err)
		}
	}
}
