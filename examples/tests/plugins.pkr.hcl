packer {
  required_plugins {
    azure = {
      source  = "github.com/hashicorp/azure"
      version = "~> 1"
    }
  }
}

source "file" "example" {
  content =  ""
  target =  "/dev/null"
}

build {
  sources = ["sources.file.example"]
}
