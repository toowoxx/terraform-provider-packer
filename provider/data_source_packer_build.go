package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceBuildType struct {
	BuildName types.String `tfsdk:"name"`
	BuildUUID types.String `tfsdk:"uuid"`
}

func (d dataSourceBuild) updateAutoComputed(_ *dataSourceBuildType) error {
	return nil
}

func (d dataSourceBuildType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "[Experimental] Waits for the build specified by its name to finish. This is only useful when you define the " +
			"name outside of the packer_image resource, allowing both the packer_image resource and this data " +
			"source to reference the same build. This data source is experimental and may change or be removed at any " +
			"time without prior notice.",
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Description: "Name of the build. Use a resource, like random_string otherwise it hangs during plan.",
				Type:        types.StringType,
				Validators: []tfsdk.AttributeValidator{
					NonEmptyStringValidator{},
				},
				Required: true,
			},
			"uuid": {
				Description: "Build UUID of the referenced build",
				Type:        types.StringType,
				Computed:    true,
			},
		},
	}, nil
}

func (d dataSourceBuildType) NewDataSource(_ context.Context, p provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	return dataSourceBuild{
		p: *(p.(*tfProvider)),
	}, nil
}

type dataSourceBuild struct {
	p tfProvider
}

func (d dataSourceBuild) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	resourceState := dataSourceBuildType{}
	diags := req.Config.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	build := state.GetBuild(resourceState.BuildName.Value)
	build.AwaitCompletion()

	resourceState.BuildUUID = types.String{Value: build.ImageResourceData.BuildUUID.Value}

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
