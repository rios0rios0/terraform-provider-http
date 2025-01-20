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
