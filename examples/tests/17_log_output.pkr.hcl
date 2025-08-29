source "file" "example" {
  content = ""
  target  = "/dev/null"
}

build {
  sources = ["sources.file.example"]
}
