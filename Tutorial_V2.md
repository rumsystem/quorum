# How to setup and test new consensus 

## start boot strap node
go run main.go bootstrapnode --listen /ip4/0.0.0.0/tcp/10666

## start owner node (full node)
 RUM_KSPASSWD=123 go run main.go fullnode --peername owner --listen /ip4/127.0.0.1/tcp/7002 --apiport 8002 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm68RHVt6NedGSiKmkRhadaxoKSRuawnEnmi7jznN7fwLm --configdir config --datadir data --keystoredir ownerkeystore  --jsontracer ownertracer.json --debug=true

## Owner create a new group
 curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' http://127.0.0.1:8002/api/v1/group

result :
{
  "seed": "rum://seed?v=1\u0026e=0\u0026n=0\u0026c=Gw1snbZWKc0cf_LzWiV4_nc4QD9MRnBZ5TX9BDd7ejo\u0026g=3yToSGPTQk-W5MCo6EE7cQ\u0026k=A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1\u0026s=TOT9QGcLR0sWocTl4TLOkOiCP-eIaaL0tyB3kw9vC6hZaqo2jO6DXjTazLSZ87iTj87EDsJtlwfiasTbflDOcwA\u0026t=FxCUiL8iSTw\u0026a=my_test_group\u0026y=test_app\u0026u=http%3A%2F%2F127.0.0.1%3A8002%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyJkZjI0ZTg0OC02M2QzLTQyNGYtOTZlNC1jMGE4ZTg0MTNiNzEiXSwiZXhwIjoxODE5NjcxNTc4LCJuYW1lIjoiYWxsb3ctZGYyNGU4NDgtNjNkMy00MjRmLTk2ZTQtYzBhOGU4NDEzYjcxIiwicm9sZSI6Im5vZGUifQ.2rkrV62zi_9BcBNV5StDC0KoSuc5HXf56XBcZx9agQk",
  "group_id": "df24e848-63d3-424f-96e4-c0a8e8413b71"
}

## Start user node (full node)
RUM_KSPASSWD=123 go run main.go fullnode --peername u1 --listen /ip4/127.0.0.1/tcp/7003 --apiport 8003 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm68RHVt6NedGSiKmkRhadaxoKSRuawnEnmi7jznN7fwLm --configdir config --datadir data --keystoredir u1keystore  --jsontracer u1tracer.json --debug=true

## User node join group
 curl -X POST -H 'Content-Type: application/json' -d '{"seed": "rum://seed?v=1\u0026e=0\u0026n=0\u0026c=Gw1snbZWKc0cf_LzWiV4_nc4QD9MRnBZ5TX9BDd7ejo\u0026g=3yToSGPTQk-W5MCo6EE7cQ\u0026k=A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1\u0026s=TOT9QGcLR0sWocTl4TLOkOiCP-eIaaL0tyB3kw9vC6hZaqo2jO6DXjTazLSZ87iTj87EDsJtlwfiasTbflDOcwA\u0026t=FxCUiL8iSTw\u0026a=my_test_group\u0026y=test_app\u0026u=http%3A%2F%2F127.0.0.1%3A8002%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyJkZjI0ZTg0OC02M2QzLTQyNGYtOTZlNC1jMGE4ZTg0MTNiNzEiXSwiZXhwIjoxODE5NjcxNTc4LCJuYW1lIjoiYWxsb3ctZGYyNGU4NDgtNjNkMy00MjRmLTk2ZTQtYzBhOGU4NDEzYjcxIiwicm9sZSI6Im5vZGUifQ.2rkrV62zi_9BcBNV5StDC0KoSuc5HXf56XBcZx9agQk"}' http://127.0.0.1:8003/api/v2/group/join

result:
{
  "group_id": "df24e848-63d3-424f-96e4-c0a8e8413b71",
  "group_name": "my_test_group",
  "owner_pubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
  "user_pubkey": "AuPOnQs4Tt_wOcysGXeafaWZ1FqqX5QABMqSazyljnde",
  "user_encryptpubkey": "age12amrr79drwkxsd8aefqpe75w53elzm6ka5rwqjyxwlqr98epepmqzea4ng",
  "consensus_type": "poa",
  "encryption_type": "public",
  "cipher_key": "1b0d6c9db65629cd1c7ff2f35a2578fe7738403f4c467059e535fd04377b7a3a",
  "app_key": "test_app",
  "signature": "52d888d3a793174061f971f96a63ce3ac8fa5746ad9c177d3f79a56eb586a70331a05185149a7d92e69459916bda082cfc5884a31294a567defbca9461fc979a01"
}

## Post content to group
 curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"df24e848-63d3-424f-96e4-c0a8e8413b71","type":"Group"}}'  http://127.0.0.1:8002/api/v1/group/content
{
  "trx_id": "9251dae7-a5d9-4d41-a223-0e9d75a5a42a"
}

## Check group info for both nodes

### Owner node (full node)
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/groups
{
  "groups": [
    {
      "group_id": "df24e848-63d3-424f-96e4-c0a8e8413b71",
      "group_name": "my_test_group",
      "owner_pubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
      "user_pubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
      "user_eth_addr": "0x2C0cc9a42733f864365CA2f5c73C18A5346689bb",
      "consensus_type": "POA",
      "encryption_type": "PUBLIC",
      "cipher_key": "1b0d6c9db65629cd1c7ff2f35a2578fe7738403f4c467059e535fd04377b7a3a",
      "app_key": "test_app",
      "epoch": 1,
      "last_updated": 1661991862923341740,
      "group_status": ""
    }
  ]
}

### user node (full node)
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8003/api/v1/groups
{
  "groups": [
    {
      "group_id": "df24e848-63d3-424f-96e4-c0a8e8413b71",
      "group_name": "my_test_group",
      "owner_pubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
      "user_pubkey": "AuPOnQs4Tt_wOcysGXeafaWZ1FqqX5QABMqSazyljnde",
      "user_eth_addr": "0xe6205CA329BDA40E4584F3B3664aa017Fe225c32",
      "consensus_type": "POA",
      "encryption_type": "PUBLIC",
      "cipher_key": "1b0d6c9db65629cd1c7ff2f35a2578fe7738403f4c467059e535fd04377b7a3a",
      "app_key": "test_app",
      "epoch": 1,
      "last_updated": 1661991862923999059,
      "group_status": ""
    }
  ]
}

## Check all nodes get newly created block

### Owner node (full node)
curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:8002/api/v1/block/df24e848-63d3-424f-96e4-c0a8e8413b71/1
{
  "GroupId": "df24e848-63d3-424f-96e4-c0a8e8413b71",
  "Epoch": 1,
  "PrevEpochHash": "a4Kh66NV8BwDvm7LayhG+eL9dvvCHhsBdF4jJaFjFAQ=",
  "Trxs": [
    {
      "TrxId": "9251dae7-a5d9-4d41-a223-0e9d75a5a42a",
      "GroupId": "df24e848-63d3-424f-96e4-c0a8e8413b71",
      "Data": "EbrTtvaMhNunHn6ESvQQGKipwOX9mEWmpqrNVtXZmy8c7/6wjaWHsRETZaps7y46YZQG3mF2WmYMZHKGrQswY71swol6nIHtkjBEtTOjuEYjZUanZjnO3NOTQ9ccCQbCEioeFd1IK48HxDeBbEwMXw==",
      "TimeStamp": "1661991862919130626",
      "Version": "1.0.0",
      "Expired": 1661991892919130686,
      "SenderPubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
      "SenderSign": "kiDhgE3eZlNNGxXe2U/2WkBjaymSBkDmcMGSgVwoKeEmuReqBVST5il3Lhc0l7k73LmchYalQ7Whe8KuKw1EswE="
    }
  ],
  "EpochHash": "vjpuirlx7I+aDJzgr4dggYTcLyW5Jtw0C6ps4ozn0hU=",
  "TimeStamp": "1661991862922241590",
  "BlockHash": "hR00PVI3ckrz3Dv9hGuhuPAVnOB3t2hm26Qh1R/QHhw=",
  "BookkeepingPubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
  "BookkeepingSignature": "v9sD61BEZR/k/8qBx2PcbFjW4F9qO2+artvBpQb7WXtF7/LHHHncKZKIh7UK/JhrkfOuU74BtDhigDzXD593cAE="
}

### user node (full node)
 curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:8003/api/v1/block/df24e848-63d3-424f-96e4-c0a8e8413b71/1
{
  "GroupId": "df24e848-63d3-424f-96e4-c0a8e8413b71",
  "Epoch": 1,
  "PrevEpochHash": "a4Kh66NV8BwDvm7LayhG+eL9dvvCHhsBdF4jJaFjFAQ=",
  "Trxs": [
    {
      "TrxId": "9251dae7-a5d9-4d41-a223-0e9d75a5a42a",
      "GroupId": "df24e848-63d3-424f-96e4-c0a8e8413b71",
      "Data": "EbrTtvaMhNunHn6ESvQQGKipwOX9mEWmpqrNVtXZmy8c7/6wjaWHsRETZaps7y46YZQG3mF2WmYMZHKGrQswY71swol6nIHtkjBEtTOjuEYjZUanZjnO3NOTQ9ccCQbCEioeFd1IK48HxDeBbEwMXw==",
      "TimeStamp": "1661991862919130626",
      "Version": "1.0.0",
      "Expired": 1661991892919130686,
      "SenderPubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
      "SenderSign": "kiDhgE3eZlNNGxXe2U/2WkBjaymSBkDmcMGSgVwoKeEmuReqBVST5il3Lhc0l7k73LmchYalQ7Whe8KuKw1EswE="
    }
  ],
  "EpochHash": "vjpuirlx7I+aDJzgr4dggYTcLyW5Jtw0C6ps4ozn0hU=",
  "TimeStamp": "1661991862922241590",
  "BlockHash": "hR00PVI3ckrz3Dv9hGuhuPAVnOB3t2hm26Qh1R/QHhw=",
  "BookkeepingPubkey": "A3QSvT61maJ7MlDVMa1B01s83bi0fSagW9_wWgHumjw1",
  "BookkeepingSignature": "v9sD61BEZR/k/8qBx2PcbFjW4F9qO2+artvBpQb7WXtF7/LHHHncKZKIh7UK/JhrkfOuU74BtDhigDzXD593cAE="
}

## You can post any trx (except announce and producer related trx), new block will be created and broadcast

# start from here 
1. create third user node
2. join the same group
3. implement and test sync logic