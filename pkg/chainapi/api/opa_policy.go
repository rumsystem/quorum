package api

const policyStr = `package quorum.restapi.authz

import future.keywords.in

default allow = false

# allow all for chain role
allow {
	input.role == "chain"
}

#######################
# rules for node role #
#######################

# Allow all user access POST /app/api/v1/token/refresh
allow {
  input.method == "POST"
  input.path = ["app", "api", "v1", "token", "refresh"]
}

# allow all for "*" in allow_groups
allow {
	input.allow_groups == ["*"]
	input.path[0] == "api"
	input.path[1] == "v1"
	input.path[2] == "node"
}

# Allow access GET /api/v1/trx/:group_id/:trx_id
allow {
  some group_id
  some trx_id
  input.method == "GET"
  input.path = ["api", "v1", "trx", group_id, trx_id]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/trx/:group_id
allow {
  some group_id
  input.method == "POST"
  input.path = ["api", "v1", "node", "trx", group_id]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/groupctn/:group_id
allow {
  some group_id
  input.method == "POST"
  input.path = ["api", "v1", "node", "groupctn", group_id]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/getchaindata/:group_id
allow {
  some group_id
  input.method == "POST"
  input.path = ["api", "v1", "node", "getchaindata", group_id]
  input.allow_groups[_] == group_id
}
`
