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

resource "http_request" "example" {
  path    = "/posts/1"
  method  = "GET"
  headers = {
    "Content-Type" = "application/json"
  }
}

output "response_body" {
  value = http_request.example.response_body
}

output "response_code" {
  value = http_request.example.response_code
}
