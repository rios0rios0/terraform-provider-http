terraform {
  required_providers {
    http = {
      source = "rios0rios0/http"
    }
  }
}

provider "http" {}

resource "http_request" "example" {
  url     = "https://jsonplaceholder.typicode.com/posts/1"
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
