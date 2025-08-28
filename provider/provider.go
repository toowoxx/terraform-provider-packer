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
					Description: "Optional path to a Packer binary to use instead of the embedded one.",
					Optional:    true,
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

func (p *tfProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Read provider config
	var cfg struct {
		PackerBinary types.String `tfsdk:"packer_binary"`
	}
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve binary to use and validate
	bin := ""
	if !cfg.PackerBinary.IsNull() && !cfg.PackerBinary.IsUnknown() && cfg.PackerBinary.ValueString() != "" {
		bin = cfg.PackerBinary.ValueString()
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
