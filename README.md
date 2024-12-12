# Terraform Provider for HTTP Requests
This Terraform provider allows you to execute HTTP requests and store the response in the Terraform state. It supports specifying the URL, method, headers, and captures both the response body and response code.

## Requirements
- [Go](https://golang.org/doc/install) 1.16+
- [Terraform](https://www.terraform.io/downloads.html) 0.13+

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
       username = "something"
       password = "***"
     }
     ignore_tls = true
   }

   resource "http_request" "example1" {
     method  = "GET"
     path     = "/posts/1"
     headers = {
       "Content-Type" = "application/json"
     }
     is_response_body_json = true
     response_body_id_filter = "$.id"
   }

   resource "http_request" "example2" {
     method  = "POST"
     path     = "/posts"
     headers = {
       "Content-Type" = "application/json"
     }
     request_body = jsonencode({
       anything = http_request.example1.response_body_json["id"]
     })
   }

   output "example1_response_body_id" {
     value = http_request.example1.response_body_id
   }

   output "example2_response_body_id" {
     value = http_request.example1.response_body_json["id"]
   }
   
   output "example2_response_body_param" {
     value = lookup(http_request.example1.response_body_json, "param1.param2.value", "")
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

## TODO

- accept when the ID changes, because the request has changed
- create the delete feature to delete from the state
- create the import feature to import the HTTP requests

## References
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework/resources/create)
- [Develop a Terraform provider (Terraform HashiCups Provider)](https://github.com/hashicorp/terraform-provider-hashicups)
- [Terraform Provider Scaffolding (Terraform Plugin Framework)](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout/tree/master?tab=readme-ov-file)
