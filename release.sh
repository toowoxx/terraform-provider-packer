#!/bin/bash

set -euo pipefail

image_name="terraform_builder_$(pwd | sha1sum | cut -c 1-8)"

echo "Using image name $image_name"

mkdir -p bin

setfacl -Rdm u:$UID:rwX bin
setfacl -Rm u:$UID:rwX bin

docker build -t $image_name .
docker run -v $(pwd):/data -t $image_name bash -c 'cd /data && make build-release'

make release

