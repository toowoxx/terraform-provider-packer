package provider

import (
	"context"
	"os"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toowoxx/go-lib-userspace-common/cmds"
)

func New() tfsdk.Provider {
	return &provider{}
}

type provider struct {
}

func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"dummy": {
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "Used as a placeholder. Do not use.",
			},
		},
	}, nil
}

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	exe, _ := os.Executable()
	err := cmds.RunCommandWithEnv(exe, map[string]string{packer_interop.TPPRunPacker: "true"}, "version")
	if err != nil {
		panic(err)
	}
}

// GetResources - Defines provider resources
func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"packer_image": resourceImageType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"packer_version": dataSourceVersionType{},
		"packer_files":   dataSourceFilesType{},
		"packer_build":   dataSourceBuildType{},
	}, nil
}
