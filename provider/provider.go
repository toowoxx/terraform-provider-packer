package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	provider_schema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func New() provider.Provider {
	return &tfProvider{}
}

type tfProvider struct{}

func (p *tfProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	*resp = provider.MetadataResponse{
		TypeName: "packer",
	}
}

func (p *tfProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	*response = provider.SchemaResponse{
		Schema: provider_schema.Schema{
			Attributes: map[string]provider_schema.Attribute{
				"packer_binary": provider_schema.StringAttribute{
					Description: "Path to a Packer-compatible binary to run. A command name without a path, " +
						"such as `packer`, is looked up on PATH. Conflicts with `packer_binary_url`. " +
						"When neither `packer_binary` nor `packer_binary_url` is set, the provider runs its embedded " +
						"Packer, which is deprecated and will be removed in the next release; from then on it runs " +
						"`packer` found on PATH.",
					Optional: true,
				},
				"packer_binary_url": provider_schema.StringAttribute{
					Description: "Optional http(s) URL to download a Packer-compatible binary from. " +
						"The URL may serve a raw executable or a zip archive " +
						"containing one (a file named `packer`/`packer.exe`, or a single-file archive). " +
						"Downloads are cached locally and reused; changing the URL or checksum triggers a fresh download. " +
						"Conflicts with `packer_binary`. " +
						"This provider is an independent project and is not affiliated with or endorsed by HashiCorp. " +
						"You are responsible for choosing a trustworthy URL and for complying with the license of " +
						"the downloaded binary. Use `packer_binary_checksum` to verify the download.",
					Optional: true,
				},
				"packer_binary_checksum": provider_schema.StringAttribute{
					Description: "Optional SHA-256 checksum (hex, optionally prefixed with `sha256:`) used to verify " +
						"the file downloaded from `packer_binary_url`. The checksum is computed over the downloaded " +
						"artifact itself (e.g. the zip archive, not the binary inside it). Requires `packer_binary_url`.",
					Optional: true,
				},
			},
		},
	}
}

func (p *tfProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &dataSourceVersion{} },
		func() datasource.DataSource { return &dataSourceFiles{} },
	}
}

func (p *tfProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return &resourceImage{} },
	}
}

type providerSettings struct {
	PackerBinary  packerBinary
	PackerVersion string
}

func providerSettingsFrom(providerData any) (providerSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	if providerData == nil {
		return providerSettings{}, diags
	}
	settings, ok := providerData.(providerSettings)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected providerSettings, got %T", providerData))
	}
	return settings, diags
}

type providerConfig struct {
	PackerBinary         types.String `tfsdk:"packer_binary"`
	PackerBinaryURL      types.String `tfsdk:"packer_binary_url"`
	PackerBinaryChecksum types.String `tfsdk:"packer_binary_checksum"`
}

func (p *tfProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bin, diags := choosePackerBinarySource(cfg).resolvePackerBinary(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	version, diags := readPackerVersion(bin)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := providerSettings{PackerBinary: bin, PackerVersion: version}
	resp.DataSourceData = settings
	resp.ResourceData = settings
}
