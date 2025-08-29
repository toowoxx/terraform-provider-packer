variable "test_var3" {
  type = string
}

variable "test_big_float" {
  type = number
}

source "file" "example" {
  content = ""
  target  = "/dev/null"
}

build {
  sources = ["sources.file.example"]
}
