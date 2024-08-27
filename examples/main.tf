terraform {
  required_providers {
    http = {
      source = "rios0rios0/http"
    }
  }
}

provider "http" {
  url = "https://jsonplaceholder.typicode.com"
  basic_auth = {
    username = "something"
    password = "***"
  }
  ignore_tls = true
}

resource "http_request" "example1" {
  method  = "GET"
  path    = "/posts/1"
  headers = {
    "Content-Type" = "application/json"
  }
}

resource "http_request" "example2" {
  method  = "GET"
  path    = "/posts/2"
  headers = {
    "Content-Type" = "application/json"
  }
  request_body = jsonencode({
    test = lookup(jsondecode(http_request.example1.response_body_json), "id", "")
  })
}

output "response_body" {
  value = http_request.example.response_body
}

output "response_code" {
  value = http_request.example.response_code
}
