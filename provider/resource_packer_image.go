package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceImageType struct {
	ID                types.String      `tfsdk:"id"`
	Variables         map[string]string `tfsdk:"variables"`
	AdditionalParams  []string          `tfsdk:"additional_params"`
	Directory         types.String      `tfsdk:"directory"`
	File              types.String      `tfsdk:"file"`
	Environment       map[string]string `tfsdk:"environment"`
	IgnoreEnvironment types.Bool        `tfsdk:"ignore_environment"`
	Triggers          map[string]string `tfsdk:"triggers"`
	Force             types.Bool        `tfsdk:"force"`
	BuildUUID         types.String      `tfsdk:"build_uuid"`
	Name              types.String      `tfsdk:"name"`
}

func (r resourceImageType) NewResource(_ context.Context, p provider.Provider) (resource.Resource, diag.Diagnostics) {
	return resourceImage{
		p: *(p.(*tfProvider)),
	}, nil
}

type resourceImage struct {
	p tfProvider
}

func (r resourceImage) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	*resp = resource.MetadataResponse{
		TypeName: "packer_image",
	}
}

func (r resourceImage) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	*response = resource.SchemaResponse{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed: true,
				},
				"name": schema.StringAttribute{
					Description: "Name of this build. This value is not passed to Packer.",
					Optional:    true,
				},
				"variables": schema.MapAttribute{
					Description: "Variables to pass to Packer",
					ElementType: types.StringType,
					Optional:    true,
				},
				"additional_params": schema.SetAttribute{
					Description: "Additional parameters to pass to Packer. Consult Packer documentation for details. " +
						"Example: `additional_params = [\"-parallel-builds=1\"]`",
					ElementType: types.StringType,
					Optional:    true,
				},
				"directory": schema.StringAttribute{
					Description: "Working directory to run Packer inside. Default is cwd.",
					Optional:    true,
				},
				"file": schema.StringAttribute{
					Description: "Packer file to use for building",
					Optional:    true,
				},
				"force": schema.BoolAttribute{
					Description: "Force overwriting existing images",
					Optional:    true,
				},
				"environment": schema.MapAttribute{
					Description: "Environment variables",
					ElementType: types.StringType,
					Optional:    true,
				},
				"ignore_environment": schema.BoolAttribute{
					Description: "Prevents passing all environment variables of the provider through to Packer",
					Optional:    true,
				},
				"triggers": schema.MapAttribute{
					Description: "Values that, when changed, trigger an update of this resource",
					ElementType: types.StringType,
					Optional:    true,
				},
				"build_uuid": schema.StringAttribute{
					Description: "UUID that is updated whenever the build has finished. This allows detecting changes.",
					Computed:    true,
				},
			},
		},
	}
}

func (r resourceImage) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Empty().AtName("id"), req, resp)
}

func (r resourceImage) getDir(dir types.String) string {
	dirVal := dir.ValueString()
	if dir.IsUnknown() || len(dirVal) == 0 {
		dirVal = "."
	}
	return dirVal
}

func (r resourceImage) getFileParam(resourceState *resourceImageType) string {
	if resourceState.File.IsNull() || len(resourceState.File.ValueString()) == 0 {
		return "."
	} else {
		return resourceState.File.ValueString()
	}
}

func RunCommandInDirWithEnvReturnOutput(
	diags *diag.Diagnostics, name string, dir string, env map[string]string, params ...string,
) ([]byte, error) {
	cmd := exec.Command(name, params...)
	if dir != "." {
		cmd.Dir = dir
	}
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	output, err := cmd.Output()
	if err != nil {
		// Create a JSON of the parameters to make it crystal clear
		// what was passed to the command.
		paramJSON, jsonErr := json.Marshal(params)
		if jsonErr != nil {
			paramJSON = []byte("<could not marshal params to JSON>")
		}
		diags.AddWarning(
			"Failed to run command "+cmd.String(),
			"Env vars: "+fmt.Sprintf("%v", env)+"\n"+
				"Dir: "+dir+"\n"+
				"Params: "+string(paramJSON)+"\n"+
				"Output: "+string(output)+"\n"+
				"Error: "+err.Error()+"\n",
		)
		diags.AddError("Error during command", err.Error())
	}
	return output, err
}

func (r resourceImage) packerInit(resourceState *resourceImageType, diags *diag.Diagnostics) error {
	envVars := packer_interop.EnvVars(resourceState.Environment, !resourceState.IgnoreEnvironment.ValueBool())

	params := []string{"init"}
	params = append(params, r.getFileParam(resourceState))

	exe, _ := os.Executable()
	output, err := RunCommandInDirWithEnvReturnOutput(diags, exe, r.getDir(resourceState.Directory), envVars, params...)

	if err != nil {
		return errors.Wrap(err, "could not run packer command ; output: "+string(output))
	}

	return nil
}

func (r resourceImage) packerBuild(resourceState *resourceImageType, diags *diag.Diagnostics) error {
	envVars := packer_interop.EnvVars(resourceState.Environment, !resourceState.IgnoreEnvironment.ValueBool())

	params := []string{"build"}
	for key, value := range resourceState.Variables {
		params = append(params, "-var", key+"="+value)
	}
	if resourceState.Force.ValueBool() {
		params = append(params, "-force")
	}
	params = append(params, resourceState.AdditionalParams...)
	params = append(params, r.getFileParam(resourceState))

	exe, _ := os.Executable()
	output, err := RunCommandInDirWithEnvReturnOutput(diags, exe, r.getDir(resourceState.Directory), envVars, params...)
	if err != nil {
		return errors.Wrap(err, "could not run packer command; output: "+string(output))
	}

	return nil
}

func (r resourceImage) updateState(resourceState *resourceImageType, _ *diag.Diagnostics) error {
	if resourceState.ID.IsUnknown() {
		resourceState.ID = types.StringValue(uuid.Must(uuid.NewRandom()).String())
	}
	resourceState.BuildUUID = types.StringValue(uuid.Must(uuid.NewRandom()).String())

	return nil
}

func (r resourceImage) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resourceState := resourceImageType{}
	diags := req.Config.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.packerInit(&resourceState, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer init", err.Error())
		return
	}
	err = r.packerBuild(&resourceState, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer build", err.Error())
		return
	}
	err = r.updateState(&resourceState, &resp.Diagnostics)
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

func (r resourceImage) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var resourceState resourceImageType
	diags := req.State.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceImage) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceImageType
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var resourceState resourceImageType
	diags = req.State.Get(ctx, &resourceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.packerInit(&plan, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer init", err.Error())
		return
	}
	err = r.packerBuild(&plan, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer build", err.Error())
		return
	}
	err = r.updateState(&plan, &resp.Diagnostics)
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

func (r resourceImage) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceImageType
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}
