# Importing never destroys and recreates the resource.
#
# Terraform does not show your configuration to a provider during import, so an identifier that
# does not spell out every argument produces a state that differs from your HCL. Instead of letting
# that difference force a replacement, this provider records what the identifier left unsaid and
# adopts those values from your configuration on the first apply, in place and without sending any
# HTTP request. That apply prints a warning naming exactly what was adopted.
#
# In practice: import with the shortest form that identifies the resource, then `terraform apply`.

# ---------------------------------------------------------------------------------------------
# 1) Bare path -- the shortest form. The method defaults to GET.
# ---------------------------------------------------------------------------------------------
terraform import http_request.example1 '/posts/1'

# ---------------------------------------------------------------------------------------------
# 2) Method and path -- when the resource was not created with GET.
# ---------------------------------------------------------------------------------------------
terraform import http_request.example2 'POST /posts'

# ---------------------------------------------------------------------------------------------
# 3) Raw JSON -- when you want to pin arguments rather than let them be adopted.
#    Only "method" and "path" are required.
# ---------------------------------------------------------------------------------------------
terraform import http_request.example3 '{"method":"GET","path":"/posts/1","headers":{"Accept":"application/json"},"is_response_body_json":true,"response_body_id_filter":"$.id"}'

# ---------------------------------------------------------------------------------------------
# 4) A file holding that same JSON -- avoids shell quoting entirely and can be committed next to
#    the configuration. The path follows the "@" with no space.
# ---------------------------------------------------------------------------------------------
cat > import-example4.json <<'JSON'
{
  "method": "GET",
  "path": "/posts/1",
  "headers": { "Accept": "application/json" },
  "is_response_body_json": true,
  "response_body_id_filter": "$.id"
}
JSON
terraform import http_request.example4 '@./import-example4.json'

# ---------------------------------------------------------------------------------------------
# 5) The identifier the provider itself renders. Capture `import_id` as an output so it survives
#    the loss of the state file that would make it necessary. It encodes only the arguments you
#    configured -- neither `basic_auth` nor the captured response is in it -- so it is safe to
#    paste into a shell or a CI log.
# ---------------------------------------------------------------------------------------------
terraform import http_request.example5 "$(terraform output -raw example5_import_id)"

# ---------------------------------------------------------------------------------------------
# 6) An import block, for Terraform 1.5 and later.
# ---------------------------------------------------------------------------------------------
cat > import-example6.tf <<'HCL'
import {
  to = http_request.example6
  id = "/posts/1"
}
HCL

# ---------------------------------------------------------------------------------------------
# 7) An import block addressing the resource by identity, for Terraform 1.12 and later.
#    "method" and "path" are required; "base_url" is optional.
# ---------------------------------------------------------------------------------------------
cat > import-example7.tf <<'HCL'
import {
  to = http_request.example7
  identity = {
    method = "GET"
    path   = "/posts/1"
  }
}
HCL

# ---------------------------------------------------------------------------------------------
# Capturing the response
#
# For GET and HEAD the provider reads the endpoint during the import, so `response_body`,
# `response_code`, `response_body_id`, `response_body_json` and `delete_resolved_path` are filled
# in from the live API.
#
# It never replays POST, PUT, PATCH or DELETE, because re-sending one would repeat its side effect
# and create a second remote object. Name the object to read with "import_read_path" instead, or
# supply "response_body" directly. This matters whenever `delete_path` contains JSONPath tokens:
# they are resolved against `response_body`, and destroy fails without it.
# ---------------------------------------------------------------------------------------------
terraform import http_request.example8 '{"method":"POST","path":"/posts","import_read_path":"/posts/101"}'

# ---------------------------------------------------------------------------------------------
# Also accepted, for backwards compatibility: "<id>/<base64 of the JSON payload>", and that base64
# payload on its own. Standard, URL-safe, padded and unpadded base64 are all understood -- earlier
# releases rejected any payload whose standard-alphabet encoding happened to contain a "/".
# ---------------------------------------------------------------------------------------------
payload="$(printf '%s' '{"method":"GET","path":"/posts/1"}' | base64 | tr -d '\n')"
terraform import http_request.example9 "00000000-0000-0000-0000-000000000000/$payload"
