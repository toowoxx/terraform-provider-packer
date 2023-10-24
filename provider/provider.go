package provider

import (
	"context"
	"os"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	provider_schema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/toowoxx/go-lib-userspace-common/cmds"
)

func New() provider.Provider {
	return &tfProvider{}
}

type tfProvider struct {
}

func (p *tfProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	*resp = provider.MetadataResponse{
		TypeName: "packer",
	}
}

func (p *tfProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	*response = provider.SchemaResponse{
		Schema: provider_schema.Schema{
			Attributes: map[string]provider_schema.Attribute{},
		},
	}
}

func (p *tfProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return dataSourceVersion{p: *p} },
		func() datasource.DataSource { return dataSourceFiles{p: *p} },
	}
}

func (p *tfProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return resourceImage{p: *p} },
	}
}

func (p *tfProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
	exe, _ := os.Executable()
	err := cmds.RunCommandWithEnv(exe, map[string]string{packer_interop.TPPRunPacker: "true"}, "version")
	if err != nil {
		panic(err)
	}
}
