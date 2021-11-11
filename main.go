package main

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

import (
	"context"
	"log"

	"terraform-provider-packer/provider"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func main() {
	if err := tfsdk.Serve(context.Background(), provider.New, tfsdk.ServeOpts{
		Name: "packer",
	}); err != nil {
		log.Fatal(err)
	}
}
