# generate a random UUID
uuid="00000000-0000-0000-00000-000000000000"

# convert the desired block (http_request.example1) to JSON:
# {
#   "method": "GET",
#   "path": "/posts/1",
#   "headers": {
#     "Content-Type": "application/json"
#   },
#   "is_response_body_json": true,
#   "response_body_id_filter": "$.id"
# }

# encode it to base64
base64="eyJtZXRob2QiOiAiR0VUIiwgInBhdGgiOiAiL3Bvc3RzLzEiLCAiaGVhZGVycyI6IHsiQ29udGVudC1UeXBlIjogImFwcGxpY2F0aW9uL2pzb24ifSwgImlzX3Jlc3BvbnNlX2JvZHlfanNvbiI6IHRydWUsICJyZXNwb25zZV9ib2R5X2lkX2ZpbHRlciI6ICIkLmlkIn0="

# import by using UUID/B64
terraform import http_request.example1 "$uuid/$base64"
