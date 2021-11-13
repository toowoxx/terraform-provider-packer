terraform {
  required_providers {
    packer = {
      source = "toowoxx/packer"
    }
  }
}

provider "packer" {}

data "packer_version" "version" {}

data "packer_file_dependencies" "deps1" {
  file = "example.pkr.hcl"
}
data "packer_file_dependencies" "deps2" {
  file = "example2.pkr.hcl"
}

resource "packer_build" "build1" {
  file = data.packer_file_dependencies.deps1.file
  variables = {
    test_var1 = "test 1"
  }

  triggers = {
    packer_version = data.packer_version.version.version
    file_hash = data.packer_file_dependencies.deps1.file_hash
    file_dependencies_hash = data.packer_file_dependencies.deps1.file_dependencies_hash
  }
}

resource "packer_build" "build2" {
  file = data.packer_file_dependencies.deps2.file
  force = true
  variables = {
    test_var2 = "test 2"
  }

  triggers = {
    packer_version = data.packer_version.version.version
    file_hash = data.packer_file_dependencies.deps2.file_hash
    file_dependencies_hash = data.packer_file_dependencies.deps2.file_dependencies_hash
  }
}

output "packer_version" {
  value = data.packer_version.version.version
}

output "build_uuid_1" {
  value = resource.packer_build.build1.build_uuid
}

output "build_uuid_2" {
  value = resource.packer_build.build2.build_uuid
}

output "file_hash_1" {
  value = data.packer_file_dependencies.deps1.file_hash
}
