resource "http_request_nonfatal" "ok" {
  method  = "GET"
  path    = "/status/200"
  headers = { "Accept" = "application/json" }
}

resource "http_request_nonfatal" "not_found" {
  method  = "GET"
  path    = "/status/404"
  headers = { "Accept" = "application/json" }
}

output "ok_code"        { value = http_request_nonfatal.ok.response_code }
output "not_found_code" { value = http_request_nonfatal.not_found.response_code }
