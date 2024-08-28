output "example1_response_body_id" {
  value = http_request.example1.response_body_id
}

output "example2_response_body_id" {
  value = http_request.example1.response_body_json["id"]
}

output "example2_response_body_param" {
  value = lookup(http_request.example1.response_body_json, "param1.param2.value", "")
}
