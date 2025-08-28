package provider

import (
	"context"
	"fmt"
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
	return &dataSourceVersion{
		p: *(p.(*tfProvider)),
	}, nil
}

type dataSourceVersion struct {
	p            tfProvider
	packerBinary string
}

func (r dataSourceVersion) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	*resp = datasource.MetadataResponse{
		TypeName: "packer_version",
	}
}

func (r *dataSourceVersion) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if settings, ok := req.ProviderData.(providerSettings); ok {
		r.packerBinary = settings.PackerBinary
	}
}

func (r dataSourceVersion) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	resourceState := dataSourceVersionType{}
	exe := r.packerBinary
	if exe == "" {
		exe, _ = os.Executable()
	}
	// Pass through current env and disable checkpoint to avoid network calls
	env := packer_interop.EnvVars(map[string]string{}, true)
	output, err := cmds.RunCommandWithEnvReturnOutput(exe, env, "version")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to run packer",
			fmt.Sprintf("Command: %s version\nError: %v\nOutput:\n%s", exe, err, strings.TrimSpace(string(output))),
		)
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
