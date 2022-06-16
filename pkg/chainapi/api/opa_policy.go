package api

const policyStr = `package quorum.restapi.authz

import future.keywords.in

default allow = false

# allow all for chain role
allow {
	input.role == "chain"
}

# Allow all user access POST /app/api/v1/token/refresh
allow {
  input.method == "POST"
  input.path = ["app", "api", "v1", "token", "refresh"]
}

# allow all for "*" in allow_groups
allow {
	input.allow_groups == ["*"]
	input.path[0] == "v1"
	input.path[1] == "nodesdk"
}

# Allow access /v1/nodesdk/...
allow {
  some group_id
  input.method == "POST"
  input.path = ["v1", "nodesdk", "groupctn", group_id]
  input.allow_groups[_] == group_id
}
`
