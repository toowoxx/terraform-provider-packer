package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	provider_schema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func New() provider.Provider {
	return &tfProvider{}
}

type tfProvider struct {
	packerBinary string
}

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
					Description: "Optional path to a Packer binary to use instead of the embedded one. " +
						"Conflicts with `packer_binary_url`.",
					Optional: true,
				},
				"packer_binary_url": provider_schema.StringAttribute{
					Description: "Optional http(s) URL to download a Packer-compatible binary from, " +
						"used instead of the embedded one. The URL may serve a raw executable or a zip archive " +
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
		func() datasource.DataSource { return &dataSourceVersion{p: *p} },
		func() datasource.DataSource { return &dataSourceFiles{p: *p} },
	}
}

func (p *tfProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return &resourceImage{p: *p} },
	}
}

type providerSettings struct {
	PackerBinary string
}

func runCommandWithEnvCapture(bin string, env map[string]string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)
	// start from current env and override with provided map
	envMap := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range env {
		envMap[k] = v
	}
	// marshal back to slice
	cmd.Env = cmd.Env[:0]
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	return cmd.CombinedOutput()
}

func knownStringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func (p *tfProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Read provider config
	var cfg struct {
		PackerBinary         types.String `tfsdk:"packer_binary"`
		PackerBinaryURL      types.String `tfsdk:"packer_binary_url"`
		PackerBinaryChecksum types.String `tfsdk:"packer_binary_checksum"`
	}
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	binPath := knownStringValue(cfg.PackerBinary)
	binURL := knownStringValue(cfg.PackerBinaryURL)
	checksum := knownStringValue(cfg.PackerBinaryChecksum)

	if binPath != "" && binURL != "" {
		resp.Diagnostics.AddError(
			"Conflicting provider configuration",
			"packer_binary and packer_binary_url are mutually exclusive. Configure at most one of them.",
		)
		return
	}
	if checksum != "" && binURL == "" {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			"packer_binary_checksum requires packer_binary_url to be set.",
		)
		return
	}

	// Resolve binary to use and validate
	bin := binPath
	if binURL != "" {
		downloaded, err := ensureDownloadedPackerBinary(ctx, binURL, checksum)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to download Packer binary",
				fmt.Sprintf("Could not provide a Packer binary from packer_binary_url.\nURL: %s\nError: %v", binURL, err),
			)
			return
		}
		bin = downloaded
	}
	if bin != "" {
		// Validate external packer with pass-through env; do not force embedded re-exec
		envExternal := map[string]string{"CHECKPOINT_DISABLE": "1"}
		if out, err := runCommandWithEnvCapture(bin, envExternal, "version"); err != nil {
			resp.Diagnostics.AddError(
				"Invalid packer_binary",
				fmt.Sprintf(
					"Failed to execute provided Packer binary.\nBinary: %s\nError: %v\nOutput:\n%s",
					bin,
					err,
					strings.TrimSpace(string(out)),
				),
			)
			return
		}
	} else {
		exe, _ := os.Executable()
		// Validate embedded packer with re-exec env
		envEmbedded := packer_interop.EnvVars(map[string]string{}, true)
		if out, err := runCommandWithEnvCapture(exe, envEmbedded, "version"); err != nil {
			resp.Diagnostics.AddError(
				"Embedded Packer unavailable",
				fmt.Sprintf(
					"Failed to execute embedded Packer.\nError: %v\nOutput:\n%s",
					err,
					strings.TrimSpace(string(out)),
				),
			)
			return
		}
	}

	p.packerBinary = bin
	settings := providerSettings{PackerBinary: p.packerBinary}
	resp.DataSourceData = settings
	resp.ResourceData = settings
}
