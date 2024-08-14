terraform {
  required_providers {
    packer = {
      source = "toowoxx/packer"
    }
  }
}

provider "packer" {}

data "packer_version" "version" {}

data "packer_files" "files1" {
  file = "example.pkr.hcl"
}
data "packer_files" "files2" {
  directory = "subdir"
}

resource "packer_image" "image1" {
  file = data.packer_files.files1.file
  variables = {
    test_var1 = "test 1"
    test_var2 = "test 2"
    test_int = 420
    test_float = 3.1416
  }
  sensitive_variables = {
    test_big_float = 1.234e100
    test_bool = true
    test_list = tolist([
      "element 1", "element 2"
    ])
  }

  triggers = {
    packer_version = data.packer_version.version.version
    files_hash     = data.packer_files.files1.files_hash
  }
}

resource "random_string" "random" {
  length  = 16
  special = false
  lower   = true
  upper   = true
  number  = true
}

resource "packer_image" "image2" {
  directory = data.packer_files.files2.directory
  force     = true
  variables = {
    test_var3 = "test 3"
  }
  sensitive_variables = {
    test_big_float = 1.234e100
  }
  ignore_environment = false
  name               = random_string.random.result
  additional_params = [
    "-parallel-builds=1"
  ]

  triggers = {
    packer_version = data.packer_version.version.version
    files_hash     = data.packer_files.files2.files_hash
  }
}

output "packer_version" {
  value = data.packer_version.version.version
}

output "build_uuid_1" {
  value = resource.packer_image.image1.build_uuid
}

output "build_uuid_2" {
  value = resource.packer_image.image2.build_uuid
}

output "file_hash_1" {
  value = data.packer_files.files1.files_hash
}
