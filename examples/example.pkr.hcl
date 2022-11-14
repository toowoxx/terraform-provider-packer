variable "test_var1" {
  type = string
}

variable "test_var2" {
  type = string
}

source "file" "example" {
  content =  ""
  target =  "/dev/null"
}

build {
  sources = ["sources.file.example"]
}
