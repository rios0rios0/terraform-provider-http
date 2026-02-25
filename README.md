<h1 align="center">Terraform Provider HTTP</h1>
<p align="center">
    <a href="https://github.com/rios0rios0/terraform-provider-http/releases/latest">
        <img src="https://img.shields.io/github/release/rios0rios0/terraform-provider-http.svg?style=for-the-badge&logo=github" alt="Latest Release"/></a>
    <a href="https://github.com/rios0rios0/terraform-provider-http/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/rios0rios0/terraform-provider-http.svg?style=for-the-badge&logo=github" alt="License"/></a>
    <a href="https://github.com/rios0rios0/terraform-provider-http/actions/workflows/default.yaml">
        <img src="https://img.shields.io/github/actions/workflow/status/rios0rios0/terraform-provider-http/default.yaml?branch=main&style=for-the-badge&logo=github" alt="Build Status"/></a>
</p>

A Terraform provider that facilitates the execution of HTTP requests and enables the storage of responses within the Terraform state. The primary advantage is the ability to utilize stored responses in subsequent requests.

While this provider is not designed to replace the [http](https://registry.terraform.io/providers/hashicorp/http/latest/docs) provider, it can be used alongside it. Notably, the official [http](https://registry.terraform.io/providers/hashicorp/http/latest/docs) provider does not store responses in the state, which limits its ability to use responses in future requests.

This provider supports specifying the URL, method, and headers, and it captures both the response body and response code.

## Requirements

- [Go](https://golang.org/doc/install) >= 1.23.4
- [Terraform](https://www.terraform.io/downloads.html) >= 1.10.4 (tested and approved)

## Installation

Add the provider to your Terraform configuration:
```hcl
terraform {
  required_providers {
    http = {
      source = "rios0rios0/http"
    }
  }
}
```

## Usage

```hcl
resource "http_request" "example" {
  url    = "https://api.example.com/data"
  method = "GET"
  headers = {
    "Authorization" = "Bearer ${var.token}"
  }
}

output "response_body" {
  value = http_request.example.response_body
}
```

## Contributing

Contributions are welcome! Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## References

- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework/resources/create)
- [Develop a Terraform provider (Terraform HashiCups Provider)](https://github.com/hashicorp/terraform-provider-hashicups)
- [Terraform Provider Scaffolding (Terraform Plugin Framework)](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout/tree/master?tab=readme-ov-file)
