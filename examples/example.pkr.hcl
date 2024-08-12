variable "test_var1" {
  type = string
}

variable "test_var2" {
  type = string
}

variable "test_int" {
  type = number
}

variable "test_float" {
  type = number
}

variable "test_big_float" {
  type = number
}

variable "test_bool" {
  type = bool
}

variable "test_list" {
  type = list(string)
}

source "file" "example" {
  content =  ""
  target =  "/dev/null"
}

build {
  sources = ["sources.file.example"]
}
