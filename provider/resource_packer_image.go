package provider

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/toowoxx/go-lib-userspace-common/cmds"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type resourceImageType struct {
	ID               types.String      `tfsdk:"id"`
	Variables        map[string]string `tfsdk:"variables"`
	AdditionalParams []string          `tfsdk:"additional_params"`
	Directory        types.String      `tfsdk:"directory"`
	File             types.String      `tfsdk:"file"`
	Environment      map[string]string `tfsdk:"environment"`
	Triggers         map[string]string `tfsdk:"triggers"`
	Force            types.Bool        `tfsdk:"force"`
	BuildUUID        types.String      `tfsdk:"build_uuid"`
}

func (r resourceImageType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:     types.StringType,
				Computed: true,
			},
			"variables": {
				Description: "Variables to pass to Packer",
				Type:        types.MapType{ElemType: types.StringType},
				Optional:    true,
			},
			"additional_params": {
				Description: "Additional parameters to pass to Packer",
				Type:        types.SetType{ElemType: types.StringType},
				Optional:    true,
			},
			"directory": {
				Description: "Working directory to run Packer inside. Default is cwd.",
				Type:        types.StringType,
				Optional:    true,
			},
			"file": {
				Description: "Packer file to use for building",
				Type:        types.StringType,
				Optional:    true,
			},
			"force": {
				Description: "Force overwriting existing images",
				Type:        types.BoolType,
				Optional:    true,
			},
			"environment": {
				Description: "Environment variables",
				Type:        types.MapType{ElemType: types.StringType},
				Optional:    true,
			},
			"triggers": {
				Description: "Values that, when changed, trigger an update of this resource",
				Type:        types.MapType{ElemType: types.StringType},
				Optional:    true,
			},
			"build_uuid": {
				Description: "UUID that is updated whenever the build has finished. This allows detecting changes.",
				Type:        types.StringType,
				Computed:    true,
			},
		},
	}, nil
}

func (r resourceImageType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceImage{
		p: *(p.(*provider)),
	}, nil
}

type resourceImage struct {
	p provider
}

func (r resourceImage) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func (r resourceImage) getDir(dir types.String) string {
	dirVal := dir.Value
	if dir.Unknown || len(dirVal) == 0 {
		dirVal = "."
	}
	return dirVal
}

func (r resourceImage) getFileParam(resourceState *resourceImageType) string {
	if resourceState.File.Null || len(resourceState.File.Value) == 0 {
		return "."
	} else {
		return resourceState.File.Value
	}
}

func (r resourceImage) packerInit(resourceState *resourceImageType) error {
	envVars := map[string]string{}
	for key, value := range resourceState.Environment {
		envVars[key] = value
	}
	envVars[tppRunPacker] = "true"

	params := []string{"init"}
	params = append(params, r.getFileParam(resourceState))

	exe, _ := os.Executable()
	output, err := cmds.RunCommandInDirWithEnvReturnOutput(exe, r.getDir(resourceState.Directory), envVars, params...)

	if err != nil {
		return errors.Wrap(err, "could not run packer command; output: "+string(output))
	}

	return nil
}

func (r resourceImage) packerBuild(resourceState *resourceImageType) error {
	envVars := map[string]string{}
	for key, value := range resourceState.Environment {
		envVars[key] = value
	}
	envVars[tppRunPacker] = "true"

	params := []string{"build"}
	for key, value := range resourceState.Variables {
		params = append(params, "-var", key+"="+value)
	}
	if resourceState.Force.Value {
		params = append(params, "-force")
	}
	params = append(params, r.getFileParam(resourceState))
	params = append(params, resourceState.AdditionalParams...)

	exe, _ := os.Executable()
	output, err := cmds.RunCommandInDirWithEnvReturnOutput(exe, r.getDir(resourceState.Directory), envVars, params...)
	if err != nil {
		return errors.Wrap(err, "could not run packer command; output: "+string(output))
	}

	return nil
}

func (r resourceImage) updateState(resourceState *resourceImageType) error {
	if resourceState.ID.Unknown {
		resourceState.ID = types.String{Value: uuid.Must(uuid.NewRandom()).String()}
	}
	resourceState.BuildUUID = types.String{Value: uuid.Must(uuid.NewRandom()).String()}

	return nil
}

func (r resourceImage) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	resourceState := resourceImageType{}
	diags := req.Config.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.packerInit(&resourceState)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer init", err.Error())
		return
	}
	err = r.packerBuild(&resourceState)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer build", err.Error())
		return
	}
	err = r.updateState(&resourceState)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
		return
	}

	diags = resp.State.Set(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceImage) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state resourceImageType
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceImage) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan resourceImageType
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state resourceImageType
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.packerInit(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer init", err.Error())
		return
	}
	err = r.packerBuild(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer build", err.Error())
		return
	}
	err = r.updateState(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceImage) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state resourceImageType
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}
