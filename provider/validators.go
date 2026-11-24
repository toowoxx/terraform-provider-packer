package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NonEmptyStringValidator struct{}

func (n NonEmptyStringValidator) Description(_ context.Context) string {
	return "Checks if the given string is not empty."
}

func (n NonEmptyStringValidator) MarkdownDescription(ctx context.Context) string {
	return n.Description(ctx)
}

func (n NonEmptyStringValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	var str types.String
	diags := tfsdk.ValueAs(ctx, request.ConfigValue, &str)
	response.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	if str.IsUnknown() || str.IsNull() {
		return
	}

	if len(str.ValueString()) == 0 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"String is empty",
			fmt.Sprintf("String should not be empty."),
		)

		return
	}
}

func (p *tfProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var cfg providerConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.PackerBinary.ValueString() != "" && cfg.PackerBinaryURL.ValueString() != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("packer_binary"),
			"Conflicting provider configuration",
			"packer_binary and packer_binary_url are mutually exclusive. Configure at most one of them.",
		)
	}
	if cfg.PackerBinaryChecksum.ValueString() != "" && !cfg.PackerBinaryURL.IsUnknown() && cfg.PackerBinaryURL.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("packer_binary_checksum"),
			"Invalid provider configuration",
			"packer_binary_checksum requires packer_binary_url to be set.",
		)
	}
}

var (
	_ validator.String                    = (*NonEmptyStringValidator)(nil)
	_ provider.ProviderWithValidateConfig = (*tfProvider)(nil)
)
