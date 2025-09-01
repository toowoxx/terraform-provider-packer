package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"terraform-provider-packer/hclconv"
	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

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
	PackerVersion      types.String      `tfsdk:"packer_version"`
	ManifestPath       types.String      `tfsdk:"manifest_path"`
	Manifest           types.Dynamic     `tfsdk:"manifest"`
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

// Version 2 state (before introducing packer_version)
type resourceImageTypeV2 struct {
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

func (r resourceImageType) NewResource(_ context.Context, p provider.Provider) (resource.Resource, diag.Diagnostics) {
	return &resourceImage{
		p: *(p.(*tfProvider)),
	}, nil
}

type resourceImage struct {
	p            tfProvider
	packerBinary string
}

func (r resourceImage) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	*resp = resource.MetadataResponse{
		TypeName: "packer_image",
	}
}

func (r *resourceImage) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if settings, ok := req.ProviderData.(providerSettings); ok {
		r.packerBinary = settings.PackerBinary
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
					WriteOnly: true,
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
				"packer_version": schema.StringAttribute{
					Description: "Detected Packer version used for this resource. Changing this forces replacement.",
					Computed:    true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				"manifest_path": schema.StringAttribute{
					Description: "Path to the Packer manifest JSON to read after build. If set, a manifest must be written to that path. If unset, the provider passes a temporary path via environment variable TPP_MANIFEST_PATH; if Packer does not create it, the manifest remains null.",
					Optional:    true,
				},
				"manifest": schema.DynamicAttribute{
					Description: "Packer manifest content decoded as a dynamic value. Access fields directly in Terraform.",
					Computed:    true,
				},
			},
			Version: 5,
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
		2: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                  schema.StringAttribute{Computed: true},
					"name":                schema.StringAttribute{Optional: true},
					"variables":           schema.DynamicAttribute{Optional: true},
					"sensitive_variables": schema.DynamicAttribute{Optional: true, Sensitive: true},
					"additional_params":   schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"directory":           schema.StringAttribute{Optional: true},
					"file":                schema.StringAttribute{Optional: true},
					"force":               schema.BoolAttribute{Optional: true},
					"environment":         schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"ignore_environment":  schema.BoolAttribute{Optional: true},
					"triggers":            schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"build_uuid":          schema.StringAttribute{Computed: true},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior resourceImageTypeV2
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				upgraded := resourceImageType{
					Variables:          prior.Variables,
					SensitiveVariables: prior.SensitiveVariables,
					AdditionalParams:   prior.AdditionalParams,
					Directory:          prior.Directory,
					File:               prior.File,
					Environment:        prior.Environment,
					IgnoreEnvironment:  prior.IgnoreEnvironment,
					Triggers:           prior.Triggers,
					Force:              prior.Force,
					BuildUUID:          prior.BuildUUID,
					Name:               prior.Name,
					PackerVersion:      types.StringNull(),
					ManifestPath:       types.StringNull(),
					Manifest:           types.DynamicNull(),
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
		3: {
			// Prior schema is the v3 schema (before manifest support)
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                  schema.StringAttribute{Computed: true},
					"name":                schema.StringAttribute{Optional: true},
					"variables":           schema.DynamicAttribute{Optional: true},
					"sensitive_variables": schema.DynamicAttribute{Optional: true, Sensitive: true},
					"additional_params":   schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"directory":           schema.StringAttribute{Optional: true},
					"file":                schema.StringAttribute{Optional: true},
					"force":               schema.BoolAttribute{Optional: true},
					"environment":         schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"ignore_environment":  schema.BoolAttribute{Optional: true},
					"triggers":            schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"build_uuid":          schema.StringAttribute{Computed: true},
					"packer_version":      schema.StringAttribute{Computed: true},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior resourceImageType
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				upgraded := resourceImageType{
					ID:                 prior.ID,
					Variables:          prior.Variables,
					SensitiveVariables: prior.SensitiveVariables,
					AdditionalParams:   prior.AdditionalParams,
					Directory:          prior.Directory,
					File:               prior.File,
					Environment:        prior.Environment,
					IgnoreEnvironment:  prior.IgnoreEnvironment,
					Triggers:           prior.Triggers,
					Force:              prior.Force,
					BuildUUID:          prior.BuildUUID,
					Name:               prior.Name,
					PackerVersion:      prior.PackerVersion,
					ManifestPath:       types.StringNull(),
					Manifest:           types.DynamicNull(),
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
		4: {
			// Prior schema is the v4 schema (before write-only sensitive_variables)
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                  schema.StringAttribute{Computed: true},
					"name":                schema.StringAttribute{Optional: true},
					"variables":           schema.DynamicAttribute{Optional: true},
					"sensitive_variables": schema.DynamicAttribute{Optional: true, Sensitive: true},
					"additional_params":   schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"directory":           schema.StringAttribute{Optional: true},
					"file":                schema.StringAttribute{Optional: true},
					"force":               schema.BoolAttribute{Optional: true},
					"environment":         schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"ignore_environment":  schema.BoolAttribute{Optional: true},
					"triggers":            schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"build_uuid":          schema.StringAttribute{Computed: true},
					"packer_version":      schema.StringAttribute{Computed: true},
					"manifest_path":       schema.StringAttribute{Optional: true},
					"manifest":            schema.DynamicAttribute{Computed: true},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior resourceImageType
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				// Purge any persisted sensitive_variables from prior states.
				upgraded := resourceImageType{
					ID:                 prior.ID,
					Variables:          prior.Variables,
					SensitiveVariables: types.DynamicNull(),
					AdditionalParams:   prior.AdditionalParams,
					Directory:          prior.Directory,
					File:               prior.File,
					Environment:        prior.Environment,
					IgnoreEnvironment:  prior.IgnoreEnvironment,
					Triggers:           prior.Triggers,
					Force:              prior.Force,
					BuildUUID:          prior.BuildUUID,
					Name:               prior.Name,
					PackerVersion:      prior.PackerVersion,
					ManifestPath:       prior.ManifestPath,
					Manifest:           prior.Manifest,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
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

	exe := r.getPackerExecutable()
	output, err := RunCommandInDirWithEnvReturnOutput(diags, exe, r.getDir(resourceState.Directory), envVars, params...)

	if err != nil {
		return errors.Wrap(err, "could not run packer command ; output: "+string(output))
	}

	return nil
}

func (r resourceImage) getManifestPath(resourceState *resourceImageType) (path string, fromUser bool, err error) {
	// If user specified a path, use it without creating the file; ensure directory exists.
	if !resourceState.ManifestPath.IsNull() && !resourceState.ManifestPath.IsUnknown() {
		p := strings.TrimSpace(resourceState.ManifestPath.ValueString())
		if p == "" {
			return "", true, fmt.Errorf("manifest_path is empty")
		}
		dir := filepath.Dir(p)
		fi, statErr := os.Stat(dir)
		if statErr != nil {
			return "", true, fmt.Errorf("directory for manifest_path %q does not exist: %v", p, statErr)
		}
		if !fi.IsDir() {
			return "", true, fmt.Errorf("directory for manifest_path %q is not a directory", p)
		}
		return p, true, nil
	}
	// Otherwise, return a temp path (not creating the file). Use a UUID-based name.
	rand := uuid.Must(uuid.NewRandom()).String()
	p := filepath.Join(os.TempDir(), fmt.Sprintf("packer-manifest-%s.json", rand))
	return p, false, nil
}

func (r resourceImage) packerBuild(resourceState *resourceImageType, diags *diag.Diagnostics, manifestPath string) error {
	envVars := packer_interop.EnvVars(resourceState.Environment, !resourceState.IgnoreEnvironment.ValueBool())
	if manifestPath != "" {
		envVars[packer_interop.TPPManifestPath] = manifestPath
	}

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

	exe := r.getPackerExecutable()
	output, err := RunCommandInDirWithEnvReturnOutput(diags, exe, r.getDir(resourceState.Directory), envVars, params...)
	if err != nil {
		return errors.Wrap(err, "could not run packer command; output: "+string(output))
	}

	return nil
}

func (r resourceImage) getPackerExecutable() string {
	if r.packerBinary != "" {
		return r.packerBinary
	}
	exe, _ := os.Executable()
	return exe
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

func (r resourceImage) detectPackerVersion(resourceState *resourceImageType, diags *diag.Diagnostics) {
	exe := r.getPackerExecutable()
	env := packer_interop.EnvVars(resourceState.Environment, !resourceState.IgnoreEnvironment.ValueBool())
	output, err := RunCommandInDirWithEnvReturnOutput(diags, exe, r.getDir(resourceState.Directory), env, "version")
	if err != nil || len(output) == 0 {
		return
	}
	v := strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(string(output), "Packer")), "v")
	resourceState.PackerVersion = types.StringValue(v)
}

// readManifestFromPath reads and decodes the manifest JSON into a dynamic value.
func (r resourceImage) readManifestFromPath(path string, resourceState *resourceImageType, diags *diag.Diagnostics) error {
	if strings.TrimSpace(path) == "" {
		// Do not set null here to keep plan-time semantics; caller controls when to set
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		diags.AddError("Failed to read Packer manifest", fmt.Sprintf("Could not read manifest file %q: %v", path, err))
		return err
	}
	if len(raw) == 0 {
		diags.AddError(
			"Empty Packer manifest",
			fmt.Sprintf(
				"Manifest file %q is empty. This usually means the Packer template did not configure a manifest post-processor to write to this file or produced no builds. Ensure a post-processor \"manifest\" is present and writes to the path provided via %s (e.g., a variable default = env(\"%s\") and post-processor output = var.tpp_manifest_path).",
				path, packer_interop.TPPManifestPath, packer_interop.TPPManifestPath,
			),
		)
		return fmt.Errorf("empty manifest at %s", path)
	}
	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		diags.AddError("Failed to parse Packer manifest JSON", fmt.Sprintf("File %q is not valid JSON: %v", path, err))
		return err
	}
	v, err := convertJSONToAttr(decoded)
	if err != nil {
		diags.AddError("Failed to convert manifest JSON", err.Error())
		return err
	}
	resourceState.Manifest = types.DynamicValue(v)
	return nil
}

// convertJSONToAttr converts an arbitrary decoded JSON value into a Terraform attr.Value.
func convertJSONToAttr(v interface{}) (attr.Value, error) {
	switch t := v.(type) {
	case map[string]interface{}:
		attrs := make(map[string]attr.Value, len(t))
		typesMap := make(map[string]attr.Type, len(t))
		for k, vv := range t {
			if vv == nil {
				attrs[k] = types.DynamicNull()
				typesMap[k] = types.DynamicType
				continue
			}
			av, err := convertJSONToAttr(vv)
			if err != nil {
				return nil, err
			}
			attrs[k] = av
			typesMap[k] = av.Type(context.Background())
		}
		ov, diags := types.ObjectValue(typesMap, attrs)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to build object value: %v", diags.Errors())
		}
		return ov, nil
	case []interface{}:
		elems := make([]attr.Value, 0, len(t))
		elemTypes := make([]attr.Type, 0, len(t))
		for _, vv := range t {
			if vv == nil {
				elems = append(elems, types.DynamicNull())
				elemTypes = append(elemTypes, types.DynamicType)
				continue
			}
			av, err := convertJSONToAttr(vv)
			if err != nil {
				return nil, err
			}
			elems = append(elems, av)
			elemTypes = append(elemTypes, av.Type(context.Background()))
		}
		tv, diags := types.TupleValue(elemTypes, elems)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to build tuple value: %v", diags.Errors())
		}
		return tv, nil
	case string:
		return types.StringValue(t), nil
	case float64:
		return types.NumberValue(big.NewFloat(t)), nil
	case bool:
		return types.BoolValue(t), nil
	case nil:
		return types.DynamicNull(), nil
	default:
		return nil, fmt.Errorf("unsupported JSON value type: %T", t)
	}
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
	// Generate a manifest path for this run and pass it via env
	manifestPath, fromUser, mpErr := r.getManifestPath(&resourceState)
	if mpErr != nil {
		resp.Diagnostics.AddError("Failed to generate manifest path", mpErr.Error())
		return
	}

	err = r.packerBuild(&resourceState, &resp.Diagnostics, manifestPath)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer build", err.Error())
		return
	}

	// If user-specified path, require manifest presence; else, optional
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		if fromUser {
			resp.Diagnostics.AddError("Packer manifest not found", fmt.Sprintf("Expected manifest at %q but it was not created. Ensure a manifest post-processor writes to this path (env %s).", manifestPath, packer_interop.TPPManifestPath))
			return
		}
		// Auto path mode: leave manifest null (user not using manifest)
		resourceState.Manifest = types.DynamicNull()
	} else {
		if err := r.readManifestFromPath(manifestPath, &resourceState, &resp.Diagnostics); err != nil {
			return
		}
	}
	err = r.updateState(&resourceState, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
		return
	}
	r.detectPackerVersion(&resourceState, &resp.Diagnostics)

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

	// For write-only attributes (e.g., sensitive_variables), Terraform does not
	// persist values into the plan/state. Read the config to access them during apply.
	var cfg resourceImageType
	diags = req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.SensitiveVariables = cfg.SensitiveVariables

	err := r.packerInit(&plan, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer init", err.Error())
		return
	}
	manifestPath, fromUser, mpErr := r.getManifestPath(&plan)
	if mpErr != nil {
		resp.Diagnostics.AddError("Failed to generate manifest path", mpErr.Error())
		return
	}

	err = r.packerBuild(&plan, &resp.Diagnostics, manifestPath)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer build", err.Error())
		return
	}

	if _, statErr := os.Stat(manifestPath); statErr != nil {
		if fromUser {
			resp.Diagnostics.AddError("Packer manifest not found", fmt.Sprintf("Expected manifest at %q but it was not created. Ensure a manifest post-processor writes to this path (env %s).", manifestPath, packer_interop.TPPManifestPath))
			return
		}
		plan.Manifest = types.DynamicNull()
	} else {
		if err := r.readManifestFromPath(manifestPath, &plan, &resp.Diagnostics); err != nil {
			return
		}
	}
	err = r.updateState(&plan, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to run packer", err.Error())
		return
	}
	r.detectPackerVersion(&plan, &resp.Diagnostics)

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

// Ensure the Resource satisfies the resource.ResourceWithModifyPlan interface.
var _ resource.ResourceWithModifyPlan = (*resourceImage)(nil)

func (r resourceImage) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
    // If this is a destroy plan, nothing to do.
    if req.Plan.Raw.IsNull() {
        return
    }

    // Read prior state if it exists (for updates), otherwise zero value (creates)
    var prior resourceImageType
    if !req.State.Raw.IsNull() {
        resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
        if resp.Diagnostics.HasError() {
            return
        }
    }

    // Read config to detect the Packer version that will be used during apply
    var cfg resourceImageType
    resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Detect Packer version using config's environment and directory
    var detectDiags diag.Diagnostics
    r.detectPackerVersion(&cfg, &detectDiags)
    if detectDiags.HasError() {
        // If detection fails, do not modify the plan or force replacement
        return
    }

    // Set the planned value to the detected version so it's not null/unknown
    if !cfg.PackerVersion.IsNull() && !cfg.PackerVersion.IsUnknown() {
        resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("packer_version"), cfg.PackerVersion)...)
    }

    // If updating and version changed, require replacement
    if !req.State.Raw.IsNull() {
        oldV := ""
        if !prior.PackerVersion.IsNull() && !prior.PackerVersion.IsUnknown() {
            oldV = prior.PackerVersion.ValueString()
        }
        newV := ""
        if !cfg.PackerVersion.IsNull() && !cfg.PackerVersion.IsUnknown() {
            newV = cfg.PackerVersion.ValueString()
        }
        if oldV != "" && newV != "" && oldV != newV {
            resp.RequiresReplace = append(resp.RequiresReplace, path.Root("packer_version"))
        }
    }
}
