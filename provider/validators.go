package provider

import (
	"context"
	"fmt"

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

func (n NonEmptyStringValidator) Validate(
	ctx context.Context, request tfsdk.ValidateAttributeRequest, response *tfsdk.ValidateAttributeResponse,
) {
	var str types.String
	diags := tfsdk.ValueAs(ctx, request.AttributeConfig, &str)
	response.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	if str.Unknown || str.Null {
		return
	}

	if len(str.Value) == 0 {
		response.Diagnostics.AddAttributeError(
			request.AttributePath,
			"String is empty",
			fmt.Sprintf("String should not be empty."),
		)

		return
	}
}

var (
	_ tfsdk.AttributeValidator = (*NonEmptyStringValidator)(nil)
)
