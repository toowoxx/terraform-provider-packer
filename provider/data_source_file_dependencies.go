package provider

import (
	"context"

	"terraform-provider-packer/crypto_util"
	"terraform-provider-packer/funcs"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceFileDependenciesType struct {
	File                 types.String `tfsdk:"file"`
	FileHash             types.String `tfsdk:"file_hash"`
	FileDependencies     []string     `tfsdk:"file_dependencies"`
	FileDependenciesHash types.String `tfsdk:"file_dependencies_hash"`
}

func (d dataSourceFileDependencies) updateAutoComputed(resourceState *dataSourceFileDependenciesType) error {
	fileHash, err := funcs.FileSHA256(resourceState.File.Value)
	if err != nil {
		return err
	}
	resourceState.FileHash = types.String{Value: fileHash}

	depFilesHash, err := crypto_util.FilesSHA256(resourceState.FileDependencies...)
	if err != nil {
		return err
	}
	resourceState.FileDependenciesHash = types.String{Value: depFilesHash}

	return nil
}

func (d dataSourceFileDependenciesType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"file": {
				Description: "Packer file to use for building",
				Type:        types.StringType,
				Required:    true,
			},
			"file_hash": {
				Description: "Hash of the file provided. Used for updates.",
				Type:        types.StringType,
				Computed:    true,
			},
			"file_dependencies_hash": {
				Description: "Hash of file dependencies combined",
				Type:        types.StringType,
				Computed:    true,
			},
			"file_dependencies": {
				Description: "Files that should be depended on so that the resource is updated when these files change",
				Type:        types.SetType{ElemType: types.StringType},
				Optional:    true,
			},
		},
	}, nil
}

func (d dataSourceFileDependenciesType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceFileDependencies{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceFileDependencies struct {
	p provider
}

func (d dataSourceFileDependencies) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	resourceState := dataSourceFileDependenciesType{}
	diags := req.Config.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := d.updateAutoComputed(&resourceState)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run compute attributes", err.Error())
		return
	}

	diags = resp.State.Set(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
