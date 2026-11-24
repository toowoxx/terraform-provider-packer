package provider

import (
	"context"
	"fmt"
	"os"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const embeddedPackerDeprecationDetail = "Neither packer_binary nor packer_binary_url is set, so the provider runs " +
	"the Packer build embedded in it. The embedded Packer is deprecated and will be removed in the next release " +
	"of this provider. From that release on, when neither attribute is set, the provider runs the packer command " +
	"found on PATH.\n\n" +
	choosePackerBinaryStep + " " +
	"packer_binary = \"packer\" runs the packer command found on PATH, as the next release does by default. " +
	"This provider does not supply or recommend any particular binary.\n\n" +
	"A binary that reports a different version than the embedded Packer changes packer_version, which replaces " +
	"packer_image resources."

func embeddedPackerIn(executable string) packerBinary {
	return packerBinary{
		Path:     executable,
		Env:      map[string]string{packer_interop.TPPRunPacker: "true"},
		Embedded: true,
	}
}

type embeddedPacker struct{}

type unknownConfigPackerBinary struct{}

func (embeddedPacker) resolvePackerBinary(context.Context) (packerBinary, diag.Diagnostics) {
	bin, diags := locateEmbeddedPacker()
	if !diags.HasError() {
		diags.AddWarning("Embedded Packer is deprecated", embeddedPackerDeprecationDetail)
	}
	return bin, diags
}

func (unknownConfigPackerBinary) resolvePackerBinary(context.Context) (packerBinary, diag.Diagnostics) {
	return locateEmbeddedPacker()
}

func locateEmbeddedPacker() (packerBinary, diag.Diagnostics) {
	var diags diag.Diagnostics
	exe, err := os.Executable()
	if err != nil {
		diags.AddError(
			"Embedded Packer unavailable",
			fmt.Sprintf(
				"Could not locate the provider executable that runs the embedded Packer: %v\n\n%s",
				err,
				choosePackerBinaryStep,
			),
		)
		return packerBinary{}, diags
	}
	return embeddedPackerIn(exe), diags
}
