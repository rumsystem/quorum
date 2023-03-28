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

# Allow access /api/v1/node/:group_id/trx
allow {
  some group_id
  input.method == "POST"
  input.path = ["api", "v1", "node", group_id, "trx"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/groupctn
allow {
  some group_id
  input.method == "POST"
  input.path = ["api", "v1", "node", group_id, "groupctn"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/announce
allow {
  some group_id
  input.method == "POST"
  input.path = ["api", "v1", "node", group_id, "announce"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/auth/by/:trx_type
allow {
  some group_id
  some trx_type
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "auth", "by", trx_type]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/auth/alwlist
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "auth", "alwlist"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/auth/denylist
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "auth", "denylist"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/appconfig/keylist
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "appconfig", "keylist"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/appconfig/by/:key
allow {
  some group_id
  some key
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "appconfig", "by", key]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/announced/producer
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "announced", "producer"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/announced/user
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "announced", "user"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/producers
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "producers"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/info
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "info"]
  input.allow_groups[_] == group_id
}

# Allow access /api/v1/node/:group_id/encryptpubkeys
allow {
  some group_id
  input.method == "GET"
  input.path = ["api", "v1", "node", group_id, "encryptpubkeys"]
  input.allow_groups[_] == group_id
}
`
