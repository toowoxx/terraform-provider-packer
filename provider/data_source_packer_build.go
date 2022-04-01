package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceBuildType struct {
	BuildName types.String `tfsdk:"name"`
	BuildUUID types.String `tfsdk:"uuid"`
}

func (d dataSourceBuild) updateAutoComputed(resourceState *dataSourceBuildType) error {
	return nil
}

func (d dataSourceBuildType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Waits for the build specified by its name to finish. This is only useful when you define the " +
			"name outside of the packer_image resource, allowing both the packer_image resource and this data " +
			"source to reference the same build. Use this if you want to have the image created before recreating a " +
			"VM when recreation involves deletion in which case the downtime would include the image build process.",
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

func (d dataSourceBuildType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceBuild{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceBuild struct {
	p provider
}

func (d dataSourceBuild) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
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
