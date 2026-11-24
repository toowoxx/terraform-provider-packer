package provider

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"strings"

	"terraform-provider-packer/packer_interop"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const choosePackerBinaryStep = "Set packer_binary to the path of a Packer-compatible binary, or set " +
	"packer_binary_url to a URL serving one and packer_binary_checksum to its SHA-256 checksum."

type packerBinarySource interface {
	resolvePackerBinary(ctx context.Context) (packerBinary, diag.Diagnostics)
}

type localPackerBinary struct {
	path string
}

type downloadedPackerBinary struct {
	url      string
	checksum string
}

var (
	_ packerBinarySource = localPackerBinary{}
	_ packerBinarySource = downloadedPackerBinary{}
	_ packerBinarySource = embeddedPacker{}
	_ packerBinarySource = unknownConfigPackerBinary{}
)

func choosePackerBinarySource(cfg providerConfig) packerBinarySource {
	switch {
	case cfg.PackerBinary.IsUnknown() || cfg.PackerBinaryURL.IsUnknown() || cfg.PackerBinaryChecksum.IsUnknown():
		return unknownConfigPackerBinary{}
	case cfg.PackerBinaryURL.ValueString() != "":
		return downloadedPackerBinary{url: cfg.PackerBinaryURL.ValueString(), checksum: cfg.PackerBinaryChecksum.ValueString()}
	case cfg.PackerBinary.ValueString() != "":
		return localPackerBinary{path: cfg.PackerBinary.ValueString()}
	default:
		return embeddedPacker{}
	}
}

func (s localPackerBinary) resolvePackerBinary(context.Context) (packerBinary, diag.Diagnostics) {
	return packerBinary{Path: s.path}, nil
}

func (s downloadedPackerBinary) resolvePackerBinary(ctx context.Context) (packerBinary, diag.Diagnostics) {
	var diags diag.Diagnostics
	downloaded, err := ensureDownloadedPackerBinary(ctx, s.url, s.checksum)
	if err != nil {
		diags.AddError(
			"Failed to download Packer binary",
			fmt.Sprintf("Could not provide a Packer binary from packer_binary_url.\nURL: %s\nError: %v", s.url, err),
		)
		return packerBinary{}, diags
	}
	return packerBinary{Path: downloaded}, diags
}

type packerBinary struct {
	Path     string
	Env      map[string]string
	Embedded bool
}

func (b packerBinary) envVars(additionalEnvVars map[string]string, passThroughCurrent bool) map[string]string {
	env := packer_interop.EnvVars(additionalEnvVars, passThroughCurrent)
	maps.Copy(env, b.Env)
	return env
}

func envList(env map[string]string) []string {
	list := make([]string, 0, len(env))
	for key, value := range env {
		list = append(list, key+"="+value)
	}
	return list
}

func readPackerVersion(bin packerBinary) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	cmd := exec.Command(bin.Path, "version")
	cmd.Env = envList(bin.envVars(nil, true))
	output, err := cmd.CombinedOutput()
	version := strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(string(output), "Packer")), "v")
	if err == nil && version == "" {
		err = errors.New("the output contains no version")
	}
	if err != nil {
		failure := "Running the configured Packer binary with the version argument failed."
		if bin.Embedded {
			failure = "Running the embedded Packer with the version argument failed."
		}
		diags.AddError(
			"Packer binary not usable",
			fmt.Sprintf(
				"%s\nBinary: %s\nError: %v\nOutput:\n%s\n\n"+
					"Check that this binary runs with the version argument on the machine running Terraform, "+
					"or choose another one. %s",
				failure,
				bin.Path,
				err,
				strings.TrimSpace(string(output)),
				choosePackerBinaryStep,
			),
		)
		return "", diags
	}
	return version, diags
}
