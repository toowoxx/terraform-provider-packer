package provider

import (
	"context"
	"os"

	"terraform-provider-packer/crypto_util"
	"terraform-provider-packer/funcs"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/toowoxx/go-lib-userspace-common/cmds"
)

type resourceBuildType struct {
	ID                   types.String      `tfsdk:"id"`
	Variables            map[string]string `tfsdk:"variables"`
	AdditionalParams     []string          `tfsdk:"additional_params"`
	Directory            types.String      `tfsdk:"directory"`
	File                 types.String      `tfsdk:"file"`
	FileHash             types.String      `tfsdk:"file_hash"`
	FileDependencies     []string          `tfsdk:"file_dependencies"`
	FileDependenciesHash types.String      `tfsdk:"file_dependencies_hash"`
	Environment          map[string]string `tfsdk:"environment"`
	Triggers             map[string]string `tfsdk:"triggers"`
	Force                types.Bool        `tfsdk:"force"`
}

func (r resourceBuildType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
		},
	}, nil
}

func (r resourceBuildType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceBuild{
		p: *(p.(*provider)),
	}, nil
}

type resourceBuild struct {
	p provider
}

func (r resourceBuild) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func (r resourceBuild) packerBuild(resourceState *resourceBuildType) error {
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
	params = append(params, resourceState.File.Value)
	params = append(params, resourceState.AdditionalParams...)

	exe, _ := os.Executable()

	output, err := cmds.RunCommandWithEnvReturnOutput(
		exe,
		envVars,
		params...,
	)
	if err != nil {
		return errors.Wrap(err, "could not run packer command; output: "+string(output))
	}

	return nil
}

func (r resourceBuild) updateAutoComputed(resourceState *resourceBuildType) error {
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

func (r resourceBuild) updateState(resourceState *resourceBuildType) error {
	if resourceState.ID.Unknown {
		resourceState.ID = types.String{Value: uuid.Must(uuid.NewRandom()).String()}
	}

	if err := r.updateAutoComputed(resourceState); err != nil {
		return err
	}

	return nil
}

func (r resourceBuild) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	resourceState := resourceBuildType{}
	diags := req.Config.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.packerBuild(&resourceState)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
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

func (r resourceBuild) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state resourceBuildType
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

func (r resourceBuild) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan resourceBuildType
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state resourceBuildType
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.packerBuild(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
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

func (r resourceBuild) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state resourceBuildType
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}
