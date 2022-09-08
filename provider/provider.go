package provider

import (
	"context"
	"os"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/toowoxx/go-lib-userspace-common/cmds"
)

func New() provider.Provider {
	return &tfProvider{}
}

type tfProvider struct {
}

func (p *tfProvider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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

func (p *tfProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
	exe, _ := os.Executable()
	err := cmds.RunCommandWithEnv(exe, map[string]string{packer_interop.TPPRunPacker: "true"}, "version")
	if err != nil {
		panic(err)
	}
}

// GetResources - Defines provider resources
func (p *tfProvider) GetResources(_ context.Context) (map[string]provider.ResourceType, diag.Diagnostics) {
	return map[string]provider.ResourceType{
		"packer_image": resourceImageType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *tfProvider) GetDataSources(_ context.Context) (map[string]provider.DataSourceType, diag.Diagnostics) {
	return map[string]provider.DataSourceType{
		"packer_version": dataSourceVersionType{},
		"packer_files":   dataSourceFilesType{},
	}, nil
}
