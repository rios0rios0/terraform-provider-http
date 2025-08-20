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

# 2) POST create + enable real DELETE using JSONPath token in delete_path
# When `is_delete_enabled = true` and `delete_path` contains JSONPath tokens (e.g. $.id),
# the provider resolves the token(s) against the JSON `response_body` from create and stores
# the computed value in `delete_resolved_path`. On `terraform destroy`, it will send the
# selected `delete_method` (default DELETE) to `delete_resolved_path`.
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

  # ---- destroy controls ----
  is_delete_enabled = true
  # Resolve to something like /posts/101 using id from the POST response
  delete_path = "/posts/$.id"
}

# 3) POST create + SOFT DELETE via POST with custom headers and body
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

  # ---- destroy controls ----
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

# 4) GET with query_parameters (echoed into the request URL)
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
