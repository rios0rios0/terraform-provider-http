terraform {
  required_providers {
    http = {
      source = "github.com/rios0rios0/terraform-provider-http"
      version = "1.0.0"
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
