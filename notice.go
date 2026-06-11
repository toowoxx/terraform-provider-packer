package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Set via -ldflags at release time; empty in local builds.
var (
	version string
	commit  string
)

const packerModulePath = "github.com/hashicorp/packer"

// embeddedPackerSource resolves the module that actually provides the
// embedded Packer from the binary's build info, honoring the go.mod
// replace directive so the notice always names the real source.
func embeddedPackerSource() (modulePath string, moduleVersion string) {
	modulePath = packerModulePath
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return modulePath, ""
	}
	for _, dep := range info.Deps {
		if dep.Path != packerModulePath {
			continue
		}
		if dep.Replace != nil {
			return dep.Replace.Path, dep.Replace.Version
		}
		return dep.Path, dep.Version
	}
	return modulePath, ""
}

func embeddedPackerNotice() string {
	var sb strings.Builder
	sb.WriteString("terraform-provider-packer")
	if version != "" {
		fmt.Fprintf(&sb, " v%s", strings.TrimPrefix(version, "v"))
	}
	if commit != "" {
		fmt.Fprintf(&sb, " (commit %s)", commit)
	}
	sb.WriteString(" running in embedded Packer mode.\n")
	sb.WriteString("This binary embeds Packer, Copyright (c) HashiCorp, Inc., " +
		"licensed under the Mozilla Public License 2.0.\n")
	modulePath, moduleVersion := embeddedPackerSource()
	fmt.Fprintf(&sb, "Embedded Packer source code: https://%s", modulePath)
	if moduleVersion != "" {
		fmt.Fprintf(&sb, " (%s)", moduleVersion)
	}
	sb.WriteString("\nThis project is not affiliated with or endorsed by HashiCorp.")
	return sb.String()
}

// suppressEmbeddedPackerNotice reports whether the invocation is a version
// probe. The provider parses the combined output of `packer version`, so
// the notice must not be emitted there.
func suppressEmbeddedPackerNotice(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "version", "-version", "--version", "-v":
		return true
	}
	return false
}
