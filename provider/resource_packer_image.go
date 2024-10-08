package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"

	"terraform-provider-packer/hclconv"
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
	ID                 types.String      `tfsdk:"id"`
	Variables          types.Dynamic     `tfsdk:"variables"`
	SensitiveVariables types.Dynamic     `tfsdk:"sensitive_variables"`
	AdditionalParams   []string          `tfsdk:"additional_params"`
	Directory          types.String      `tfsdk:"directory"`
	File               types.String      `tfsdk:"file"`
	Environment        map[string]string `tfsdk:"environment"`
	IgnoreEnvironment  types.Bool        `tfsdk:"ignore_environment"`
	Triggers           map[string]string `tfsdk:"triggers"`
	Force              types.Bool        `tfsdk:"force"`
	BuildUUID          types.String      `tfsdk:"build_uuid"`
	Name               types.String      `tfsdk:"name"`
}

type resourceImageTypeV0 struct {
	ID                types.String            `tfsdk:"id"`
	Variables         map[string]types.String `tfsdk:"variables"`
	AdditionalParams  []string                `tfsdk:"additional_params"`
	Directory         types.String            `tfsdk:"directory"`
	File              types.String            `tfsdk:"file"`
	Environment       map[string]string       `tfsdk:"environment"`
	IgnoreEnvironment types.Bool              `tfsdk:"ignore_environment"`
	Triggers          map[string]string       `tfsdk:"triggers"`
	Force             types.Bool              `tfsdk:"force"`
	BuildUUID         types.String            `tfsdk:"build_uuid"`
	Name              types.String            `tfsdk:"name"`
}

type resourceImageTypeV1 struct {
	ID                types.String      `tfsdk:"id"`
	Variables         types.Dynamic     `tfsdk:"variables"`
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
				"variables": schema.DynamicAttribute{
					Description: "Variables to pass to Packer. Must be map or object. " +
						"Can contain following types: bool, number, string, list(string), set(string).",
					Optional: true,
				},
				"sensitive_variables": schema.DynamicAttribute{
					Description: "Sensitive variables to pass to Packer " +
						"(does the same as variables, but makes sure Terraform knows these values are sensitive). " +
						"Can contain following types: bool, number, string, list(string), set(string).",
					Sensitive: true,
					Optional:  true,
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
					Description: "Environment variables to pass to Packer",
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
			Version: 2,
		},
	}
}

func (r resourceImage) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Optional: true,
					},
					"variables": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"additional_params": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"directory": schema.StringAttribute{
						Optional: true,
					},
					"file": schema.StringAttribute{
						Optional: true,
					},
					"force": schema.BoolAttribute{
						Optional: true,
					},
					"environment": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"ignore_environment": schema.BoolAttribute{
						Optional: true,
					},
					"triggers": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"build_uuid": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData resourceImageTypeV0
				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}
				convertedVariables, err := types.MapValueFrom(ctx, types.StringType, priorStateData.Variables)
				if err != nil {
					resp.Diagnostics.Append(err.Errors()...)
					return
				}
				upgradedStateData := resourceImageType{
					Variables:         types.DynamicValue(convertedVariables),
					AdditionalParams:  priorStateData.AdditionalParams,
					Directory:         priorStateData.Directory,
					File:              priorStateData.File,
					Environment:       priorStateData.Environment,
					IgnoreEnvironment: priorStateData.IgnoreEnvironment,
					Triggers:          priorStateData.Triggers,
					Force:             priorStateData.Force,
					BuildUUID:         priorStateData.BuildUUID,
					Name:              priorStateData.Name,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
		1: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Optional: true,
					},
					"variables": schema.DynamicAttribute{
						Optional: true,
					},
					"additional_params": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"directory": schema.StringAttribute{
						Optional: true,
					},
					"file": schema.StringAttribute{
						Optional: true,
					},
					"force": schema.BoolAttribute{
						Optional: true,
					},
					"environment": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"ignore_environment": schema.BoolAttribute{
						Optional: true,
					},
					"triggers": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"build_uuid": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData resourceImageTypeV1
				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := resourceImageType{
					Variables:          priorStateData.Variables,
					SensitiveVariables: types.DynamicNull(),
					AdditionalParams:   priorStateData.AdditionalParams,
					Directory:          priorStateData.Directory,
					File:               priorStateData.File,
					Environment:        priorStateData.Environment,
					IgnoreEnvironment:  priorStateData.IgnoreEnvironment,
					Triggers:           priorStateData.Triggers,
					Force:              priorStateData.Force,
					BuildUUID:          priorStateData.BuildUUID,
					Name:               priorStateData.Name,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
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
	output, err := cmd.CombinedOutput()
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

	newParams, err := createParametersFromVariables(&resourceState.Variables)
	if err != nil {
		return errors.Wrap(err, "failed to create parameters from variables")
	}
	params = append(params, newParams...)

	newParams, err = createParametersFromVariables(&resourceState.SensitiveVariables)
	if err != nil {
		return errors.Wrap(err, "failed to create parameters from sensitive variables")
	}
	params = append(params, newParams...)

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

func createParametersFromVariables(variables *types.Dynamic) ([]string, error) {
	var params []string
	if !variables.IsNull() && !variables.IsUnknown() &&
		!variables.IsUnderlyingValueNull() && !variables.IsUnderlyingValueUnknown() {
		switch value := variables.UnderlyingValue().(type) {
		case types.Map:
			for key, elementValue := range value.Elements() {
				finalValue, err := hclconv.ConvertDynamicAttributeToString(key, elementValue)
				if err != nil {
					return nil, errors.Wrap(err, fmt.Sprintf(
						"could not convert dynamic value (%s, type %s) to string",
						key,
						reflect.TypeOf(elementValue).String()))
				}
				params = append(params, "-var", key+"="+finalValue)
			}
		case types.Object:
			for key, elementValue := range value.Attributes() {
				finalValue, err := hclconv.ConvertDynamicAttributeToString(key, elementValue)
				if err != nil {
					return nil, errors.Wrap(err, fmt.Sprintf(
						"could not convert dynamic value (%s, type %s) to string",
						key,
						reflect.TypeOf(elementValue).String()))
				}
				params = append(params, "-var", key+"="+finalValue)
			}
		default:
			return nil, errors.New(
				"only maps and objects are supported for the variables attribute. Instead got: " +
					reflect.TypeOf(variables.UnderlyingValue()).String())
		}
	}
	return params, nil
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
