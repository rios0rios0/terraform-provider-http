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
   go build -o terraform-provider-http
   ```

## Using the Provider Locally

1. Create the local plugin directory structure:

   ```sh
   mkdir -p ~/.terraform.d/plugins/local/http/1.0.0/linux_amd64
   ```

2. Copy the provider binary to the local plugin directory:

   ```sh
   cp terraform-provider-http ~/.terraform.d/plugins/local/http/1.0.0/linux_amd64/
   ```

3. Create a Terraform configuration file (`main.tf`):

   ```hcl
   terraform {
     required_providers {
       http = {
         source = "local/http"
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
   ```

4. Initialize and apply the configuration:

   ```sh
   terraform init
   terraform apply
   ```

## Using the Provider from GitHub

1. Tag your provider release in your GitHub repository.

2. Update your Terraform configuration file (`main.tf`) to use the GitHub provider source:

   ```hcl
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
