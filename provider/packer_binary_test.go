package provider

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestLocalPackerBinaryResolvesToPath(t *testing.T) {
	t.Parallel()
	bin, diags := localPackerBinary{path: "/opt/tools/packer"}.resolvePackerBinary(context.Background())
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if bin.Path != "/opt/tools/packer" {
		t.Errorf("got binary %q, want %q", bin.Path, "/opt/tools/packer")
	}
}

func TestDownloadedPackerBinaryResolvesToDownload(t *testing.T) {
	useTempCacheDir(t)
	content := []byte("#!/bin/sh\necho fake packer\n")
	server, _ := serveArtifact(t, content)

	bin, diags := downloadedPackerBinary{
		url:      server.URL + "/packer",
		checksum: sha256Hex(content),
	}.resolvePackerBinary(context.Background())
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	got, err := os.ReadFile(bin.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Error("resolved binary content does not match served content")
	}
}

func TestEmbeddedPackerWarns(t *testing.T) {
	t.Parallel()
	want, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	bin, diags := embeddedPacker{}.resolvePackerBinary(context.Background())
	if bin.Path != want {
		t.Errorf("got binary %q, want the provider executable %q", bin.Path, want)
	}
	if len(diags) != 1 || diags[0].Severity() != diag.SeverityWarning {
		t.Fatalf("want exactly one warning, got %v", diags)
	}
}

func TestUnknownConfigRunsEmbeddedPackerWithoutWarningOrDownload(t *testing.T) {
	useTempCacheDir(t)
	server, requests := serveArtifact(t, []byte("#!/bin/sh\necho fake packer\n"))
	want, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	source := choosePackerBinarySource(providerConfig{
		PackerBinary:         types.StringNull(),
		PackerBinaryURL:      types.StringValue(server.URL + "/packer"),
		PackerBinaryChecksum: types.StringUnknown(),
	})
	bin, diags := source.resolvePackerBinary(context.Background())
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if bin.Path != want {
		t.Errorf("got binary %q, want the provider executable %q", bin.Path, want)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("got %d download requests, want none", got)
	}
}

func validateProviderConfig(t *testing.T, values map[string]tftypes.Value) diag.Diagnostics {
	t.Helper()
	ctx := context.Background()
	p := &tfProvider{}
	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	attributes := map[string]tftypes.Value{}
	for name := range schemaResp.Schema.Attributes {
		attributes[name] = tftypes.NewValue(tftypes.String, nil)
	}
	for name, value := range values {
		attributes[name] = value
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), attributes),
	}

	var resp provider.ValidateConfigResponse
	p.ValidateConfig(ctx, provider.ValidateConfigRequest{Config: config}, &resp)
	return resp.Diagnostics
}

func assertSingleErrorOnAttribute(t *testing.T, diags diag.Diagnostics, attribute path.Path) {
	t.Helper()
	if len(diags) != 1 || !diags.HasError() {
		t.Fatalf("want exactly one error, got %v", diags)
	}
	withPath, ok := diags[0].(diag.DiagnosticWithPath)
	if !ok || !withPath.Path().Equal(attribute) {
		t.Errorf("want the error on %s, got %v", attribute, diags[0])
	}
}

func TestValidateConfigRejectsPathAndURL(t *testing.T) {
	t.Parallel()
	diags := validateProviderConfig(t, map[string]tftypes.Value{
		"packer_binary":     tftypes.NewValue(tftypes.String, "/opt/tools/packer"),
		"packer_binary_url": tftypes.NewValue(tftypes.String, "https://packer.test/packer"),
	})
	assertSingleErrorOnAttribute(t, diags, path.Root("packer_binary"))
}

func TestValidateConfigRejectsChecksumWithoutURL(t *testing.T) {
	t.Parallel()
	diags := validateProviderConfig(t, map[string]tftypes.Value{
		"packer_binary":          tftypes.NewValue(tftypes.String, "/opt/tools/packer"),
		"packer_binary_checksum": tftypes.NewValue(tftypes.String, strings.Repeat("ab", 32)),
	})
	assertSingleErrorOnAttribute(t, diags, path.Root("packer_binary_checksum"))
}
