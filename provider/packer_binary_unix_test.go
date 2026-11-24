//go:build unix

package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"terraform-provider-packer/packer_interop"
)

func writeShellScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "packer")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadPackerVersionParsesOutput(t *testing.T) {
	bin := writeShellScript(t, `echo "Packer v1.2.3"`)
	version, diags := readPackerVersion(packerBinary{Path: bin})
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if version != "1.2.3" {
		t.Errorf("got version %q, want %q", version, "1.2.3")
	}
}

func TestReadPackerVersionPassesEmbeddedEnvOnlyToEmbeddedPacker(t *testing.T) {
	t.Setenv(packer_interop.TPPRunPacker, "")
	bin := writeShellScript(t, `echo "Packer v1.2.3-TPP_RUN_PACKER=$TPP_RUN_PACKER"`)
	for _, test := range []struct {
		name   string
		binary packerBinary
		want   string
	}{
		{name: "embedded", binary: embeddedPackerIn(bin), want: "1.2.3-TPP_RUN_PACKER=true"},
		{name: "external", binary: packerBinary{Path: bin}, want: "1.2.3-TPP_RUN_PACKER="},
	} {
		t.Run(test.name, func(t *testing.T) {
			version, diags := readPackerVersion(test.binary)
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if version != test.want {
				t.Errorf("got version %q, want %q", version, test.want)
			}
		})
	}
}

func TestReadPackerVersionReportsFailingBinary(t *testing.T) {
	bin := writeShellScript(t, "echo broken install\nexit 1")
	_, diags := readPackerVersion(packerBinary{Path: bin})
	if !diags.HasError() {
		t.Fatal("expected an error for a binary exiting with status 1")
	}
	if !strings.Contains(diags[0].Detail(), "broken install") {
		t.Errorf("error detail does not contain the binary output: %q", diags[0].Detail())
	}
}

func TestReadPackerVersionReportsBinaryWithoutVersion(t *testing.T) {
	bin := writeShellScript(t, "true")
	if _, diags := readPackerVersion(packerBinary{Path: bin}); !diags.HasError() {
		t.Error("expected an error for a binary that prints no version")
	}
}

func TestReadPackerVersionReportsMissingBinary(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "missing-packer")
	if _, diags := readPackerVersion(packerBinary{Path: bin}); !diags.HasError() {
		t.Error("expected an error for a binary that does not exist")
	}
}
