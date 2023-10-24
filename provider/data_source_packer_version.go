package provider

import (
	"context"
	"os"
	"strings"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/toowoxx/go-lib-userspace-common/cmds"
)

type dataSourceVersionType struct {
	Version string `tfsdk:"version"`
}

func (r dataSourceVersion) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	*resp = datasource.SchemaResponse{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"version": schema.StringAttribute{
					Description: "Version of embedded Packer",
					Computed:    true,
				},
			},
		},
	}
}

func (r dataSourceVersionType) NewDataSource(_ context.Context, p provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	return dataSourceVersion{
		p: *(p.(*tfProvider)),
	}, nil
}

type dataSourceVersion struct {
	p tfProvider
}

func (r dataSourceVersion) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	*resp = datasource.MetadataResponse{
		TypeName: "packer_version",
	}
}

func (r dataSourceVersion) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	resourceState := dataSourceVersionType{}
	exe, _ := os.Executable()
	output, err := cmds.RunCommandWithEnvReturnOutput(
		exe,
		map[string]string{packer_interop.TPPRunPacker: "true"},
		"version")
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
		return
	}

	if len(output) == 0 {
		resp.Diagnostics.AddError("Unexpected output", "Packer did not output anything")
		return
	}

	resourceState.Version = strings.TrimPrefix(
		strings.TrimSpace(strings.TrimPrefix(string(output), "Packer")), "v")

	diags := resp.State.Set(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
