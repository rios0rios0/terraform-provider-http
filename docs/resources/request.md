---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "http_request Resource - terraform-provider-http"
subcategory: ""
description: |-
  Represents an HTTP request resource, allowing configuration of various HTTP request parameters and capturing the response details.
  
  resource "http_request" "example1" {
    method  = "GET"
    path    = "/posts/1"
    headers = {
      "Content-Type" = "application/json"
    }
    is_response_body_json = true
    response_body_id_filter = "$.id"
  }
  
  resource "http_request" "example2" {
    method  = "POST"
    path    = "/posts"
    headers = {
      "Content-Type" = "application/json"
    }
    request_body = jsonencode({
      anything = http_request.example1.response_body_json["id"]
    })
  }
  
  
  See complete example at the GitHub repository https://github.com/rios0rios0/terraform-provider-http/blob/main/examples/main.tf.
---

# http_request (Resource)

Represents an HTTP request resource, allowing configuration of various HTTP request parameters and capturing the response details.

```hcl
resource "http_request" "example1" {
  method  = "GET"
  path    = "/posts/1"
  headers = {
    "Content-Type" = "application/json"
  }
  is_response_body_json = true
  response_body_id_filter = "$.id"
}

resource "http_request" "example2" {
  method  = "POST"
  path    = "/posts"
  headers = {
    "Content-Type" = "application/json"
  }
  request_body = jsonencode({
    anything = http_request.example1.response_body_json["id"]
  })
}

```

See complete example at the [GitHub repository](https://github.com/rios0rios0/terraform-provider-http/blob/main/examples/main.tf).



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `method` (String) The HTTP method to be used for the request (e.g., GET, POST, PUT, DELETE).
- `path` (String) The URL path for the HTTP request. This should be a relative path (e.g., /api/v1/resource).

### Optional

- `headers` (Map of String) A map of HTTP headers to include in the request. Each key-value pair represents a header name and its corresponding value.
- `is_response_body_json` (Boolean) A boolean flag indicating whether the response body is expected to be in JSON format.
- `request_body` (String) The body content to be sent with the HTTP request. This is typically used for POST and PUT requests.
- `response_body_id_filter` (String) A JSONPath filter used to extract a specific ID from the JSON response body. This is useful for identifying unique elements within the response.

### Read-Only

- `id` (String) A unique identifier for the resource. Format: `<RANDOM UNIQUE STRING>/<PARAMETERS ENCODED IN BASE64>`. This is generated by encoding the entire model (excluding the ID itself) in Base64 format.
- `response_body` (String) The raw body content returned by the server in response to the request.
- `response_body_id` (String) The extracted ID from the JSON response body, based on the provided `response_body_id_filter`. This is only populated if `is_response_body_json` is true.
- `response_body_json` (Map of String) The response body parsed as a Terraform map object. Nested items can be accessed using dot notation (e.g., "response_body_json["nested.item.value"]").
- `response_code` (Number) The HTTP status code returned by the server in response to the request (e.g., 200 for success, 404 for not found).
