terraform {
  required_providers {
    packer = {
      source = "toowoxx/packer"
    }
  }
}

provider "packer" {}

data "packer_version" "version" {}

resource "packer_build" "build" {
  file = "example.pkr.hcl"
}

output "packer_version" {
  value = data.packer_version.version.version
}
