#!/usr/bin/env bash
set -euo pipefail

# Determine repo root (script dir is scripts/)
REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$REPO_ROOT"

echo "[embedded] Building provider at $REPO_ROOT"
GOFLAGS=${GOFLAGS:-} go build -o terraform-provider-packer

# Terraform CLI config using dev_overrides to point at this repo
TF_RC=$(mktemp -t terraformrc.embedded.XXXXXX)
cat > "$TF_RC" <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/toowoxx/packer" = "$REPO_ROOT"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$TF_RC"
echo "[embedded] Using TF_CLI_CONFIG_FILE=$TF_CLI_CONFIG_FILE"

# Use the existing examples directory (and its terraform.tfstate)
cd "$REPO_ROOT/examples"

echo "[embedded] Terraform init/apply in $PWD (reuses existing state)"
terraform version || { echo "Terraform CLI not found in PATH" >&2; exit 127; }
terraform init -upgrade -input=false
terraform apply -auto-approve -input=false

echo "[embedded] Done. To destroy: (cd $REPO_ROOT/examples && terraform destroy -auto-approve)"
