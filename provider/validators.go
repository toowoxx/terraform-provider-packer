package provider

import (
	"context"
	"fmt"

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

var (
	_ validator.String = (*NonEmptyStringValidator)(nil)
)
