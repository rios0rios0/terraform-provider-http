# 1) Basic GET
resource "http_request" "get_example" {
  method  = "GET"
  path    = "/posts/1"
  headers = {
    "Accept" = "application/json"
  }
  is_response_body_json   = true
  response_body_id_filter = "$.id"
}

# 2) Resource-level base URL (allows different APIs per resource)
resource "http_request" "different_api" {
  method   = "GET"
  path     = "/posts/1"
  base_url = "https://api.example.com"  # Override provider URL
  
  headers = {
    "Accept" = "application/json"
  }
  is_response_body_json   = true
  response_body_id_filter = "$.id"
}

# 3) Resource-level authentication (per-resource credentials)
resource "http_request" "with_auth" {
  method   = "GET"
  path     = "/protected/data"
  base_url = "https://secure-api.example.com"
  
  basic_auth = {
    username = "api-user"
    password = "secret-key"
  }
  
  is_response_body_json   = true
  response_body_id_filter = "$.id"
}

# 4) Resource-level TLS configuration
resource "http_request" "insecure_api" {
  method     = "GET"
  path       = "/data"
  base_url   = "https://self-signed.example.com"
  ignore_tls = true  # Skip TLS verification for this resource
  
  is_response_body_json   = true
  response_body_id_filter = "$.id"
}

# 5) Using count with different APIs (solves the original issue!)
variable "apis" {
  default = [
    {
      name     = "api1"
      base_url = "https://api1.example.com"
      path     = "/users/1"
    },
    {
      name     = "api2" 
      base_url = "https://api2.example.com"
      path     = "/profiles/1"
    }
  ]
}

resource "http_request" "multi_api_calls" {
  count = length(var.apis)
  
  method   = "GET"
  path     = var.apis[count.index].path
  base_url = var.apis[count.index].base_url
  
  is_response_body_json   = true
  response_body_id_filter = "$.id"
}

# 6) POST create + enable real DELETE using JSONPath token in delete_path
# When `is_delete_enabled = true` and `delete_path` contains JSONPath tokens (e.g. $.id),
# the provider resolves the token(s) against the JSON `response_body` from create and stores
# the computed value in `delete_resolved_path`. On `terraform destroy`, it will send the
# selected `delete_method` (default DELETE) to `delete_resolved_path`.
#
# NOTE: Changing any delete_* field (is_delete_enabled, delete_method, delete_path,
# delete_headers, delete_request_body) does NOT trigger resource replacement.
# These fields only affect behavior during `terraform destroy`.
resource "http_request" "create_then_delete" {
  method  = "POST"
  path    = "/posts"
  headers = {
    "Content-Type" = "application/json"
    "Accept"       = "application/json"
  }
  request_body = jsonencode({
    title  = "hello world"
    body   = "example payload"
    userId = 123
  })

  is_response_body_json   = true
  response_body_id_filter = "$.id"

  # ---- destroy controls (changes do NOT trigger replacement) ----
  is_delete_enabled = true
  # Resolve to something like /posts/101 using id from the POST response
  delete_path = "/posts/$.id"
}

# 7) POST create + SOFT DELETE via POST with custom headers and body
# Some APIs require a non-DELETE verb or a specific body to "archive"/"deactivate".
resource "http_request" "create_then_soft_delete" {
  method  = "POST"
  path    = "/posts"
  headers = {
    "Content-Type" = "application/json"
    "Accept"       = "application/json"
  }
  request_body = jsonencode({
    title  = "soft delete example"
    body   = "created then archived"
    userId = 456
  })

  is_response_body_json   = true
  response_body_id_filter = "$.id"

  # ---- destroy controls (changes do NOT trigger replacement) ----
  is_delete_enabled   = true
  delete_method       = "POST"
  delete_path         = "/posts/$.id/archive"
  delete_headers = {
    "X-Force-Archive" = "true"
  }
  delete_request_body = jsonencode({
    reason = "terraform destroy"
    actor  = "automation"
  })
}

# 8) GET with query_parameters (echoed into the request URL)
resource "http_request" "with_query_params" {
  method = "GET"
  path   = "/comments"

  query_parameters = {
    postId = "1"
    q      = "foo"
  }

  headers = {
    "Accept" = "application/json"
  }
  is_response_body_json   = true
  response_body_id_filter = "$[0].id"
}

# 9) Ignore volatile inputs (headers or JSON fragments)
# Use `ignore_changes` to prevent resource replacement when specific fields change.
# This is useful for fields that contain dynamic values like UUIDs, timestamps, etc.
# Supports:
#   - Full attributes: "request_body", "headers"
#   - Map keys: "headers.X-Correlation-Id"
#   - JSON paths: "request_body.metadata.trace_id"
#
# NOTE: Delete fields (is_delete_enabled, delete_method, delete_path, delete_headers,
# delete_request_body) do NOT need to be in ignore_changes - they never trigger
# replacement because they only affect `terraform destroy` behavior.
resource "http_request" "idempotent_post" {
  method = "POST"
  path   = "/posts"
  headers = {
    "Content-Type"      = "application/json"
    "X-Correlation-Id"  = uuid()
  }

  request_body = jsonencode({
    title    = "immutable values"
    metadata = {
      trace_id = uuid()
      owner    = "terraform"
    }
  })

  ignore_changes = [
    "headers.X-Correlation-Id",
    "request_body.metadata.trace_id",
  ]
}
