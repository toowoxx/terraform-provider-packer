package provider

import (
	"context"
	"path/filepath"

	"terraform-provider-packer/crypto_util"

	"github.com/pkg/errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceFilesType struct {
	File             types.String `tfsdk:"file"`
	FilesHash        types.String `tfsdk:"files_hash"`
	FileDependencies []string     `tfsdk:"file_dependencies"`
	Directory        types.String `tfsdk:"directory"`
}

func (d dataSourceFiles) updateAutoComputed(resourceState *dataSourceFilesType) error {
	deps := resourceState.FileDependencies
	if resourceState.File.Null || len(resourceState.File.Value) == 0 {
		dir := resourceState.Directory.Value
		if resourceState.Directory.Unknown || len(dir) == 0 {
			dir = "."
		}
		hclFiles, err := filepath.Glob(dir + "/*.pkr.hcl")
		if err != nil {
			return errors.Wrap(err, "bug")
		}
		jsonFiles, err := filepath.Glob(dir + "/*.pkr.json")
		if err != nil {
			return errors.Wrap(err, "bug")
		}
		deps = append(append(deps, hclFiles...), jsonFiles...)
	} else {
		deps = append([]string{resourceState.File.Value}, deps...)
	}

	depFilesHash, err := crypto_util.FilesSHA256(deps...)
	if err != nil {
		return err
	}
	resourceState.FilesHash = types.String{Value: depFilesHash}

	return nil
}

func (d dataSourceFilesType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Specify files to detect changes. By default, the current directory will be used.",
		Attributes: map[string]tfsdk.Attribute{
			"file": {
				Description: "Packer file to use for building",
				Type:        types.StringType,
				Optional:    true,
			},
			"files_hash": {
				Description: "Hash of the files provided. Used for updates.",
				Type:        types.StringType,
				Computed:    true,
			},
			"file_dependencies": {
				Description: "Files that should be depended on so that the resource is updated when these files change",
				Type:        types.SetType{ElemType: types.StringType},
				Optional:    true,
			},
			"directory": {
				Description: "Directory to run packer in. Defaults to cwd.",
				Type:        types.StringType,
				Optional:    true,
			},
		},
	}, nil
}

func (d dataSourceFilesType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceFiles{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceFiles struct {
	p provider
}

func (d dataSourceFiles) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	resourceState := dataSourceFilesType{}
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
