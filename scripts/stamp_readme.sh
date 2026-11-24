#!/usr/bin/env sh
# Renders build/README.md for packaging: the repository README, with links
# into docs/ pointing at the packaged commit on GitHub, plus a
# provenance stamp so loose copies of the archive are self-describing.
# Usage: stamp_readme.sh <version> <commit>
set -eu

version="$1"
commit="$2"

mkdir -p build
sed "s#](docs/#](https://github.com/toowoxx/terraform-provider-packer/blob/${commit}/docs/#g" \
  README.md > build/README.md
cat >> build/README.md <<EOF

---

This copy of the README was packaged with terraform-provider-packer
version ${version}, built from commit ${commit}.
Provider source code: https://github.com/toowoxx/terraform-provider-packer
EOF
