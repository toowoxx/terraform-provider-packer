#!/usr/bin/env sh
# Renders build/README.md for packaging: the repository README plus a
# provenance stamp so loose copies of the archive are self-describing.
# Usage: stamp_readme.sh <version> <commit>
set -eu

version="$1"
commit="$2"

mkdir -p build
cp README.md build/README.md
cat >> build/README.md <<EOF

---

This copy of the README was packaged with terraform-provider-packer
version ${version}, built from commit ${commit}.
Provider source code: https://github.com/toowoxx/terraform-provider-packer
EOF
