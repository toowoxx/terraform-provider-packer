package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toowoxx/go-lib-userspace-common/cmds"
)

const tppRunPacker = "TPP_RUN_PACKER"

func New() tfsdk.Provider {
	return &provider{}
}

type provider struct {
}

func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"dummy": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
		},
	}, nil
}

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	exe, _ := os.Executable()
	err := cmds.RunCommandWithEnv(exe, map[string]string{tppRunPacker: "true"}, "version")
	if err != nil {
		panic(err)
	}
}

// GetResources - Defines provider resources
func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"packer_build": resourceBuildType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"packer_version":           dataSourceVersionType{},
		"packer_file_dependencies": dataSourceFileDependenciesType{},
	}, nil
}
