<h1 align="center">Terraform Provider HTTP</h1>
<p align="center">
    <a href="https://github.com/rios0rios0/terraform-provider-http/releases/latest">
        <img src="https://img.shields.io/github/release/rios0rios0/terraform-provider-http.svg?style=for-the-badge&logo=github" alt="Latest Release"/></a>
    <a href="https://github.com/rios0rios0/terraform-provider-http/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/rios0rios0/terraform-provider-http.svg?style=for-the-badge&logo=github" alt="License"/></a>
    <a href="https://github.com/rios0rios0/terraform-provider-http/actions/workflows/default.yaml">
        <img src="https://img.shields.io/github/actions/workflow/status/rios0rios0/terraform-provider-http/default.yaml?branch=main&style=for-the-badge&logo=github" alt="Build Status"/></a>
    <a href="https://sonarcloud.io/summary/overall?id=rios0rios0_terraform-provider-http">
        <img src="https://img.shields.io/sonar/coverage/rios0rios0_terraform-provider-http?server=https%3A%2F%2Fsonarcloud.io&style=for-the-badge&logo=sonarqubecloud" alt="Coverage"/></a>
    <a href="https://sonarcloud.io/summary/overall?id=rios0rios0_terraform-provider-http">
        <img src="https://img.shields.io/sonar/quality_gate/rios0rios0_terraform-provider-http?server=https%3A%2F%2Fsonarcloud.io&style=for-the-badge&logo=sonarqubecloud" alt="Quality Gate"/></a>
    <a href="https://www.bestpractices.dev/projects/12034">
        <img src="https://img.shields.io/cii/level/12034?style=for-the-badge&logo=opensourceinitiative" alt="OpenSSF Best Practices"/></a>
</p>

A Terraform provider that facilitates the execution of HTTP requests and enables the storage of responses within the Terraform state. The primary advantage is the ability to utilize stored responses in subsequent requests.

While this provider is not designed to replace the [http](https://registry.terraform.io/providers/hashicorp/http/latest/docs) provider, it can be used alongside it. Notably, the official [http](https://registry.terraform.io/providers/hashicorp/http/latest/docs) provider does not store responses in the state, which limits its ability to use responses in future requests.

This provider supports specifying the URL, method, and headers, and it captures both the response body and response code.

## Requirements

- [Go](https://golang.org/doc/install) >= 1.26.5
- [Terraform](https://www.terraform.io/downloads.html) >= 1.11 (write-only attributes); >= 1.12 to
  import with an `identity` block

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

The provider holds the base URL; each resource adds a `path`. A resource can override the base
with its own `base_url`.

```hcl
provider "http" {
  url = "https://api.example.com"
}

resource "http_request" "example" {
  method = "GET"
  path   = "/data"
  headers = {
    "Authorization" = "Bearer ${var.token}"
  }
}

output "response_body" {
  value = http_request.example.response_body
}
```

### Import

Importing an existing resource never destroys and recreates it.

Terraform does not show your configuration to a provider during import, so an identifier that does
not spell out every argument produces a state that differs from your HCL. Rather than let that
difference land on the replacement rules, this provider records what the identifier left unsaid and
adopts those values from your configuration on the first apply -- in place, without sending any HTTP
request, and with a warning naming exactly what was adopted. You need neither `terraform state rm`
nor a `lifecycle { ignore_changes }` block.

Import with the shortest form that identifies the resource, then `terraform apply`:

```bash
terraform import http_request.example '/data'
```

Six identifier forms are accepted, each distinguished by its first character so none can be
mistaken for another:

| Form                | Example                                          |
|---------------------|--------------------------------------------------|
| bare path           | `/posts/1` (the method defaults to `GET`)        |
| method and path     | `POST /posts`                                    |
| raw JSON            | `{"method":"GET","path":"/posts/1"}`             |
| JSON from a file    | `@./import.json`                                 |
| `<id>/<base64>`     | `0b7f.../eyJtZXRob2Q...` (accepted for backwards compatibility) |
| bare base64         | `eyJtZXRob2Q...`                                 |

Only `method` and `path` are ever required. Terraform 1.5 `import` blocks and Terraform 1.12
`import { identity = { method = ..., path = ... } }` blocks are both supported.

The provider also renders the identifier that re-imports a resource, as the computed `import_id`
attribute. Capture it as an output so it survives the loss of the state file that would make it
necessary -- it never contains credentials:

```hcl
output "example_import_id" {
  value = http_request.example.import_id
}
```

For `GET` and `HEAD` the provider reads the endpoint during the import, so the captured response
attributes are filled in from the live API. It never replays `POST`, `PUT`, `PATCH` or `DELETE`,
because re-sending one would repeat its side effect; name the object to read with
`import_read_path` in the payload instead. See [the resource documentation](docs/resources/request.md#import)
for the full set of examples.

### Drift detection

`Read` leaves the captured response alone by default, which is what existing configurations rely
on. Setting `is_refresh_enabled = true` makes every refresh re-read the resource and update the
captured response, so changes made outside Terraform appear as a diff. Because a generic HTTP
resource cannot know which endpoint reflects the object it created -- for a `POST` the creation path
is the collection, not the object -- `refresh_path` names the object and supports the same inline
JSONPath tokens as `delete_path`:

```hcl
resource "http_request" "watched" {
  method = "POST"
  path   = "/posts"

  is_response_body_json   = true
  response_body_id_filter = "$.id"

  is_refresh_enabled = true
  refresh_path       = "/posts/$.id"
}
```

A refresh whose response is neither successful nor listed in `tolerated_status_codes` removes the
resource from state, so it is planned for creation again rather than left pointing at something
that no longer exists.

### Timeouts and retries

Both the provider and the `http_request` resource accept a `request_timeout_ms` argument and a
`retry` block. Set on the provider they apply to every request; set on a resource they override
the provider defaults. This prevents a request from hanging indefinitely against a slow or
unreachable endpoint and transparently retries transient failures (connection errors and 5xx
responses, except 501) using an exponential backoff:

```hcl
provider "http" {
  url                = "https://api.example.com"
  request_timeout_ms = 60000 # give up on any request that exceeds 60s

  retry {
    attempts     = 5     # up to 5 retries (6 attempts in total)
    min_delay_ms = 1000  # optional, defaults to 1000
    max_delay_ms = 30000 # optional, defaults to 30000
  }
}

resource "http_request" "example" {
  method = "POST"
  path   = "/data"

  # Override the provider defaults for this request only.
  request_timeout_ms = 10000

  retry {
    attempts = 2
  }
}
```

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## References

- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework/resources/create)
- [Develop a Terraform provider (Terraform HashiCups Provider)](https://github.com/hashicorp/terraform-provider-hashicups)
- [Terraform Provider Scaffolding (Terraform Plugin Framework)](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout/tree/master?tab=readme-ov-file)
