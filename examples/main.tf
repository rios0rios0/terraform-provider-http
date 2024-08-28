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
