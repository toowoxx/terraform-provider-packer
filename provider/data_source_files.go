package provider

import (
	"context"
	"path/filepath"

	"terraform-provider-packer/crypto_util"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/pkg/errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	if resourceState.File.IsNull() || len(resourceState.File.ValueString()) == 0 {
		dir := resourceState.Directory.ValueString()
		if resourceState.Directory.IsUnknown() || len(dir) == 0 {
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
		deps = append([]string{resourceState.File.ValueString()}, deps...)
	}

	depFilesHash, err := crypto_util.FilesSHA256(deps...)
	if err != nil {
		return err
	}
	resourceState.FilesHash = types.StringValue(depFilesHash)

	return nil
}

func (d dataSourceFiles) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	*resp = datasource.SchemaResponse{
		Schema: schema.Schema{
			Description: "Specify files to detect changes. By default, the current directory will be used.",
			Attributes: map[string]schema.Attribute{
				"file": schema.StringAttribute{
					Description: "Packer file to use for building",
					Optional:    true,
				},
				"files_hash": schema.StringAttribute{
					Description: "Hash of the files provided. Used for updates.",
					Computed:    true,
				},
				"file_dependencies": schema.SetAttribute{
					Description: "Files that should be depended on so that the resource is updated when these files change",
					ElementType: types.StringType,
					Optional:    true,
				},
				"directory": schema.StringAttribute{
					Description: "Directory to run packer in. Defaults to cwd.",
					Optional:    true,
				},
			},
		},
	}
}

func (d dataSourceFilesType) NewDataSource(_ context.Context, p provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	return dataSourceFiles{
		p: *(p.(*tfProvider)),
	}, nil
}

type dataSourceFiles struct {
	p tfProvider
}

func (d dataSourceFiles) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	*resp = datasource.MetadataResponse{
		TypeName: "packer_files",
	}
}

func (d dataSourceFiles) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
