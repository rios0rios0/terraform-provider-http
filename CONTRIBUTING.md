## Building the Provider

1. Clone the repository:
   ```sh
   git clone https://github.com/rios0rios0/terraform-provider-http.git
   cd terraform-provider-http
   ```

2. Build the provider:
   ```sh
   make build
   ```

## Development Process

### Before Making Changes

1. Run linting to check for any existing issues:
   ```sh
   make lint
   ```

2. Run tests to ensure everything is working:
   ```sh
   make test
   ```

### Making Changes

1. Make your code changes
2. **Update CHANGELOG.md** - Add your changes to the `[Unreleased]` section following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format
3. Run linting and fix any issues:
   ```sh
   make lint-fix
   ```
4. Run tests to ensure nothing is broken:
   ```sh
   make test
   ```
5. Update documentation if needed:
   ```sh
   make docs
   ```

### Submitting Changes

- Ensure CHANGELOG.md is updated with your changes
- All tests must pass
- Linting issues must be resolved
- Documentation should be updated if the changes affect user-facing functionality

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
         version = "2.2.0"
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
         version = "2.2.0"
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
