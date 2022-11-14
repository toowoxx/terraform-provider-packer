variable "test_var3" {
  type = string
}

source "file" "example" {
  content =  ""
  target =  "/dev/null"
}

build {
  sources = ["sources.file.example"]
}
