terraform {
  required_providers {
    packer = {
      source = "toowoxx/packer"
    }
  }
}

provider "packer" {}

data "packer_version" "version" {}

resource "packer_build" "build1" {
  file = "example.pkr.hcl"
  variables = {
    test_var1 = "test 1"
  }

  triggers = {
    packer_version = data.packer_version.version.version
  }
}

resource "packer_build" "build2" {
  file = "example2.pkr.hcl"
  variables = {
    test_var2 = "test 2"
  }

  triggers = {
    packer_version = data.packer_version.version.version
  }
}

output "packer_version" {
  value = data.packer_version.version.version
}
