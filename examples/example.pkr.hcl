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

source "file" "var_test" {
  content = jsonencode({
    test_var1 = var.test_var1
    test_var2 = var.test_var2
    test_int = var.test_int
    test_float = var.test_float
    test_big_float = var.test_big_float
    test_bool = var.test_bool
    test_list = var.test_list
  })
  target = "/tmp/tpp_test_vars.json"
}

build {
  sources = ["sources.file.example", "sources.file.var_test"]
}
