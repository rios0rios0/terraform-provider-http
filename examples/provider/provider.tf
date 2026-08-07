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

# For an API that authenticates through a header instead of basic auth, set it once on the
# provider. Unlike a resource's own `headers`, these also reach the read an import performs --
# that request is built from the import identifier alone, which never sees the configuration.
provider "http" {
  alias = "bearer"
  url   = "https://jsonplaceholder.typicode.com"
  headers = {
    Authorization = "Bearer ${var.api_token}"
  }
}

variable "api_token" {
  type      = string
  sensitive = true
}
