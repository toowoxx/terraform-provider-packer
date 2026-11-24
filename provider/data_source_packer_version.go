package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type dataSourceVersionType struct {
	Version string `tfsdk:"version"`
}

func (r dataSourceVersion) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	*resp = datasource.SchemaResponse{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"version": schema.StringAttribute{
					Description: "Packer version in use",
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
	p        tfProvider
	settings providerSettings
}

func (r dataSourceVersion) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	*resp = datasource.MetadataResponse{
		TypeName: "packer_version",
	}
}

func (r *dataSourceVersion) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	settings, diags := providerSettingsFrom(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.settings = settings
}

func (r dataSourceVersion) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataSourceVersionType{Version: r.settings.PackerVersion})...)
}
