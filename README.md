# Terraform Provider for HTTP Requests

This Terraform provider allows you to execute HTTP requests and store the response in the Terraform state. It supports specifying the URL, method, headers, and captures both the response body and response code.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.13+
- [Go](https://golang.org/doc/install) 1.16+

## Building the Provider

1. Clone the repository:

   ```sh
   git clone https://github.com/yourusername/terraform-provider-http.git
   cd terraform-provider-http
   ```

2. Build the provider:

   ```sh
   make build
   ```

## Using the Provider Locally

1. Create the local plugin directory structure:

   ```sh
   make install
   ```

2. Create a Terraform configuration file (`main.tf`):
   * local provider (for the steps above 1 and 2)

   ```hcl
   terraform {
     required_providers {
       http = {
         source = "hashicorp-local.com/rios0rios0/http"
         version = "1.0.0"
       }
     }
   }
   ```

   * remote provider (skipping the steps above 1 and 2)

   ```hcl
   terraform {
     required_providers {
       http = {
         source = "rios0rios0/http"
         version = "1.0.0"
       }
     }
   }
   ```

   * and add the following configuration:

   ```hcl
   provider "http" {
     url = "https://jsonplaceholder.typicode.com"
     basic_auth = {
       username = "user"
       password = "password"
     }
     ignore_tls = true
   }

   resource "http_request" "example1" {
     method  = "GET"
     path     = "/posts/1"
     headers = {
       "Content-Type" = "application/json"
     }
   }

   resource "http_request" "example2" {
     method  = "GET"
     path     = "/posts/1"
     headers = {
       "Content-Type" = "application/json"
     }
     request_body = jsonencode({
       # TODO: the objective in the future is to avoid using "lookup" and "jsondecode" functions
       #test = http_request.example1.response_body_json["id"]
       test = lookup(jsondecode(http_request.example1.response_body_json), "id", "")
     })
   }

   output "response_body" {
     value = http_request.example.response_body
   }

   output "response_code" {
     value = http_request.example.response_code
   }
   ```

3. Initialize and apply the configuration:

   ```sh
   terraform init
   terraform apply
   ```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## References
- https://developer.hashicorp.com/terraform/plugin/framework/resources/create
- https://github.com/hashicorp/terraform-provider-hashicups
- https://github.com/hashicorp/terraform-provider-scaffolding-framework