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

output "example1_response_body_id" {
  value = http_request.example1.response_body_id
}

output "example2_response_body_id" {
  value = http_request.example1.response_body_json["id"]
}

output "example2_response_body_param" {
  value = lookup(http_request.example1.response_body_json, "param1.param2.value", "")
}
