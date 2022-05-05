package api

const policyStr = `package quorum.restapi.authz

import future.keywords.in

default allow = false

# Allow all for self role
allow {
  input.role == "self"
}

# Allow role other access GET /api/v1/group/{group_id}/content
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "group", group_id, "content"]
  input.role == "others"
}

# Allow role other access GET /api/v1/trx/{group_id}/{trx_id}
allow {
  some group_id, trx_id
  input.method == "GET"
  input.path = ["api", "v1", "trx", group_id, trx_id]
  input.role == "others"
}`
