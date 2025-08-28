#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   scripts/test_with_external_packer.sh [PATH_TO_PACKER]
# If PATH_TO_PACKER is not supplied, PACKER_BIN env var is used, otherwise `which packer`.
# Example with Nix:
#   NIXPKGS_ALLOW_UNFREE=1 nix shell github:NixOS/nixpkgs/nixos-unstable#packer --impure --command "./scripts/test_with_external_packer.sh"

PACKER_PATH=${1:-${PACKER_BIN:-}}
if [[ -z "${PACKER_PATH}" ]]; then
  PACKER_PATH=$(command -v packer || true)
fi
if [[ -z "${PACKER_PATH}" ]]; then
  echo "packer not found. Pass a path as first arg or set PACKER_BIN, or ensure 'packer' is in PATH." >&2
  exit 127
fi

# Determine repo root (script dir is scripts/)
REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$REPO_ROOT"

echo "[external] Using Packer: $PACKER_PATH"
echo "[external] Building provider at $REPO_ROOT"
GOFLAGS=${GOFLAGS:-} go build -o terraform-provider-packer

# Terraform CLI config using dev_overrides to point at this repo
TF_RC=$(mktemp -t terraformrc.external.XXXXXX)
cat > "$TF_RC" <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/toowoxx/packer" = "$REPO_ROOT"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$TF_RC"
echo "[external] Using TF_CLI_CONFIG_FILE=$TF_CLI_CONFIG_FILE"

# Work in a temp copy of examples to avoid editing local files
WORKDIR=$(mktemp -d -t tpp-external.XXXXXX)
echo "Workdir: $WORKDIR"
cp -a "$REPO_ROOT/examples" "$WORKDIR/"

# Inject packer_binary into the existing provider block in examples/main.tf
# This avoids creating a conflicting second provider configuration file.
MAIN_TF="$WORKDIR/examples/main.tf"
if [[ -f "$MAIN_TF" ]]; then
  echo "[external] Patching packer provider in main.tf with packer_binary=$PACKER_PATH"
  awk -v path="$PACKER_PATH" '
    BEGIN { inprov=0; inserted=0 }
    # Match provider "packer" opening line
    $0 ~ /^provider[[:space:]]+\"packer\"[[:space:]]*\{/ {
      # If closing brace is on the same line, expand into a multi-line block
      if ($0 ~ /\}[[:space:]]*$/) {
        sub(/\{[[:space:]]*\}[[:space:]]*$/, "{\n  packer_binary = \"" path "\"\n}")
        print
        next
      } else {
        inprov=1; inserted=0; print; next
      }
    }
    # Track if packer_binary already present inside the block
    inprov && $0 ~ /^[[:space:]]*packer_binary[[:space:]]*=/ { inserted=1 }
    # On closing brace, inject packer_binary before it if not already inserted
    inprov && $0 ~ /^[[:space:]]*\}/ {
      if (!inserted) { print "  packer_binary = \"" path "\"" }
      inprov=0; inserted=0; print; next
    }
    { print }
  ' "$MAIN_TF" > "$MAIN_TF.tmp" && mv "$MAIN_TF.tmp" "$MAIN_TF"
else
  echo "[external] main.tf not found; creating override file with provider block"
  cat > "$WORKDIR/examples/local_provider_override.tf" <<EOF
provider "packer" {
  packer_binary = "$PACKER_PATH"
}
EOF
fi

cd "$WORKDIR/examples"
echo "[external] Terraform init/apply in $PWD"
terraform version || { echo "Terraform CLI not found in PATH" >&2; exit 127; }
terraform init -upgrade -input=false
terraform apply -auto-approve -input=false

echo "[external] Done. Temp dir kept at: $WORKDIR"
echo "[external] To destroy: (cd $WORKDIR/examples && terraform destroy -auto-approve)"
