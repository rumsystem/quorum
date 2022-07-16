# NodeSDK 

  - NodeSDK provide several APIs for app creator to quick development quorum app
  - NodeSDK doesn't require user node to sync all block/trx data with full node (chainsdk)
  - NodeSDK should connect chain API provided by other full node which has the specified group used by NodeSDK
  - Currently NodeSDK does not support private group

# NodeSDK local r&d env quick start

## Download source code
  https://github.com/rumsystem/quorum/tree/nodesdk

## Run full node (chainsdk)
 RUM_KSPASSWD=123 go run main.go fullnode --peername n1 --listen /ip4/127.0.0.1/tcp/7002 --apiport 8002 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAkwZ53wCxAecczHiypKFJhWXwfP1G87n8G5R4i5qhszy8v --keystoredir n1keystore --jsontracer n1tracer.json --debug true

## Run light node (nodesdk)
RUM_KSPASSWD=123 go run main.go lightnode --peername nodesdk --apihost 127.0.0.1 --apiport 6002 --keystoredir nodesdkkeystore --debug true

Params
  - `--apihost`: nodesdk listening host
  - `--apiport`: nodesdk listening port

## Full node create a group
curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' http://127.0.0.1:8002/api/v1/group | jq

{
    "genesis_block": {
        "BlockId": "80e3dbd6-24de-46cd-9290-ed2ae93ec3ac",
        "GroupId": "c0020941-e648-40c9-92dc-682645acd17e",
        "ProducerPubKey": "CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg==",
        "Hash": "LOZa0CLITIpuQqpvXb6LyXV9z+2rSoU4JwBq0BCXttc=",
        "Signature": "MEQCICAXCicQ6f4hRNSoJR89DF3a6AKpe6ZgLXsjXqH9H3jxAiA8dpukcriwEu8amouh2ZEKA2peXr3ctKQwxI3R6+nrfg==",
        "Timestamp": 1632503907836381400
    },
    "group_id": "c0020941-e648-40c9-92dc-682645acd17e",
    "group_name": "my_test_group",
    "owner_pubkey": "CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg==",
    "owner_encryptpubkey": "age18kngxt6lkxqulldvxu8xs2ey77rrzwjhqpdey527ad4gkn3euu9sj3ah5j",
    "consensus_type": "poa",
    "encryption_type": "public",
    "cipher_key": "8e9bd83f84cf1408484d24f486861947a1db3fbe6eb3c61e31af55a4803aedc1",
    "app_key": "test_app",
    "signature": "304502206897c3c67247cba2e8d5991501b3fd471fcca06f15915efdcd814b9e99c9a48a022100aa3024eb5663da6cbbde150132a4ff52c6c6aeeb49e0c039b4c28e72b071382f"
}

## Full node add appconfig
curl -X POST -H 'Content-Type: application/json' -d '{"action":"add", "group_id":"80ca64d5-de6f-4a92-9c3d-43390b22fdfc", "name":"test_bool", "type":"bool", "value":"false", "memo":"add test_bool to group"}' http://127.0.0.1:8002/api/v1/group/appconfig | jq

{
  "group_id": "80ca64d5-de6f-4a92-9c3d-43390b22fdfc",
  "signature": "3046022100bf921a9fd48fca33092cff445680e6f50431354951183dff176fdfebfb160eef022100d8b903fe5a67f1934e2e1229a9b923496146cd885c98c2200def2f3b0b7ee2c6",
  "trx_id": "ba5918b0-fb71-43a4-9569-49a0243d6a72"
}

# NodeSDK APIs

## KeyStore 
  - NodeSDK use keystore and "alias" to manage key pairs.
  - An Alias bundled with a pair of public key/private key.
  - There are 2 types of key, signature key and encrypt key, they are different.
  - When join a group, user are asked to give both signature key and encrypt key by using alias
  - A signature key is as same as "User pubkey" in chainSDK, use to "sign" all user trx send from nodesdk
  - A encrypt key is as same as "User encrypt pubkey" in chainSDK (reverse for future use)
  - A pair of key has a "keyname", which is generated automatically 
  - A pair of key (with keyname) can be paired with MULTIPLE alias

### Create new signature keypair with Alias
curl -X POST -H 'Content-Type: application/json' -d '{"alias":"my_signature", "type":"sign"}' http://127.0.0.1:6002/nodesdk_api/v1/keystore/create | jq

params:
  alias: alias of the new keypair create
  type : type of key, one of "sign" "encrypt"

Result:
{
  "Alias": "my_signature",
  "Keyname": "d69b8175-31af-4a9c-8177-4ff64b1e3a64",
  "KeyType": "sign",
  "Pubkey": "04ed309b6cc7f175bab9d467b1eaa3ca2bad9ff5e42a6d0add884e45a0ead039225cca5c1d5afb3589881f05d0cb3392f332c03546af41fff15ce84831dc364e43"
}

Alias  : Alias of the key pair
Keyname: name of the key pair
KeyType: type of the key pair
Pubkey : pubkey of the key pair

### Create new encrypt keypair with Alias
curl -X POST -H 'Content-Type: application/json' -d '{"alias":"my_encrypt", "type":"encrypt"}' http://127.0.0.1:6002/nodesdk_api/v1/keystore/create | jq

params:
  alias: alias of the new keypair create
  type : type of key, one of "sign" "encrypt"

Result:
{
  "Alias": "my_encrypt",
  "Keyname": "c2d14c51-f19d-4094-9ddb-ec7bf7862b9d",
  "KeyType": "encrypt",
  "Pubkey": "age1emlvjpwg3v6vkym54dkwcadzu9h397fuvpz02nvu7qjl6delxgdq3d57rf"
}

Alias  : Alias of the key pair
Keyname: name of the key pair
KeyType: type of the key pair
Pubkey : pubkey of the key pair

### List all keypairs

List all keypairs in key store
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/keystore/listall | jq

params:
  //

result：
{
  "keys": [
    {
      "Alias": [
        "my_encrypt"
      ],
      "Keyname": "c2d14c51-f19d-4094-9ddb-ec7bf7862b9d",
      "Keytype": "encrypt"
    },
    {
      "Alias": [
        "my_signature"
      ],
      "Keyname": "d69b8175-31af-4a9c-8177-4ff64b1e3a64",
      "Keytype": "sign"
    },
    {
      "Alias": [],
      "Keyname": "nodesdk_default",
      "Keytype": "sign"
    }
  ]
}

"Alias": All alias paired with this keypair
"Keyname": key name
"Keytype": key type

### Remove an exist alias (unbind an alias with keypair)
curl -X POST -H 'Content-Type: application/json' -d '{"alias":"my_sign_alias"}' http://127.0.0.1:5216/nodesdk_api/v1/keystore/remove | jq
params:
  alias : alias to remove

only the alias will be removed, keypair still in keystore

### Rebind alias with other keypair (keyname)
curl -X POST -H 'Content-Type: application/json' -d '{"alias":"my_sign_alias", "keyname":"9e183036-ceb1-4ce5-a61e-147779db5ed4", "type":"sign"}' http://127.0.0.1:5216/nodesdk_api/v1/keystore/bindalias | jq

bind alias with other key name (key pair)
params:
  alias : alias to remove
  keyname : keyname(key pair) to pair with 


## NodeSDK group

### NodeSDK join group
 curl -X POST -H 'Content-Type: application/json' -d '{"seed":{"genesis_block":{"BlockId":"cb25698d-dd90-4f18-bbae-f22233256a39","GroupId":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06","ProducerPubKey":"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==","Hash":"Xu8/VIr37w/TrmPsf95lmhwVTtGFP4ZFxj/hvM03RJA=","Signature":"MEQCIANcuz+opbZfpUqLTT2os9sEG3tqOelMqXhN6H559UwjAiAhFhaiSzuspWpv1wtRrokDyvaKjVgoj9gnNnh0ZUN/zQ==","TimeStamp":"1652985871447913818"},"group_id":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06","group_name":"my_test_group","owner_pubkey":"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==","consensus_type":"poa","encryption_type":"public","cipher_key":"7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c","app_key":"test_app","signature":"3045022100e1b793c1ff8e02f93c4e54debb39de23a0f19c69f088bdf62354ac233b0f0341022039223bd22f08db98ad7c0af7aeb6b597c5bcda4712772b210c395f6f93fdfe66"}, "sign_alias":"my_signature", "encrypt_alias":"my_encrypt", "urls":["http://127.0.0.1:8002"]}' http://127.0.0.1:6002/nodesdk_api/v1/group/join

API:
  /nodesdk_api/v1/group/join

params:
  seed : group seed
  sign_alias : key pair (by alias) used to sign
  encrypt_alias : key pair (by alias) used to encrypt
  urls: ChainSDK API urls list

Result :
 {
  "group_id":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06",
  "group_name":"my_test_group",
  "owner_pubkey":"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==",
  "sign_alias":"my_signature",
  "encrypt_alias":"my_signature",
  "consensus_type":"poa",
  "encryption_type":"public",
  "cipher_key":"7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c",
  "app_key":"test_app","signature":"304402206b2ccbe262cc20e3020b4bd9f89b2909b1a07a54403e501d15a7f4cecfdb866e02207a8142721d677a5c1053cc7c897688b89d2edcc2b0217fce109d567c8a8e31af"}

### NodeSDK list all local groups
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/listall | jq

API:
  /nodesdk_api/v1/group/listall

result:
{
  "groups": [
    {
      "group_id": "93502b34-b3ec-43bf-b96a-2b1f42ef6e06",
      "group_name": "my_test_group",
      "sign_alias": "my_signature",
      "encrypt_alias": "my_encrypt",
      "user_eth_addr": "0495180230ae0f585ca0b4fc0767e616eaed45e400f470ed50c91668e1ed76c278b7fc5a129ff154c6b200a26cc78b7b4acc5b3915cdf66286c942aa5b65166ff5",
      "consensus_type": "POA",
      "encryption_type": "PUBLIC",
      "cipher_key": "7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c",
      "app_key": "test_app",
      "last_updated": 0,
      "highest_height": 0,
      "highest_block_id": "",
      "chain_apis": [
        "http://127.0.0.1:8002"
      ]
    }
  ]
}

### NodeSDK get group info
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/80ca64d5-de6f-4a92-9c3d-43390b22fdfc/info | jq

API:
  /nodesdk_api/v1/group/:group_id/info

result:
  {
  "GroupId": "80ca64d5-de6f-4a92-9c3d-43390b22fdfc",
  "Owner": "CAISIQKBxdGuEjsljYbPgfbZTti8NAoFbW+Sh8YvCFF/PRqH4A==",
  "HighestBlockId": "fa637f55-a7d2-4475-affe-fabf76034d6b",
  "HighestHeight": 4,
  "LatestUpdate": 1653935114735942400,
  "Provider": "CAISIQKBxdGuEjsljYbPgfbZTti8NAoFbW+Sh8YvCFF/PRqH4A==",
  "Singature": "FAKE_SIGN"
}

After join a group, group info is empty, App use NodeSDK should get groupInfo by itself, if successful,  groupInfo will be update to DB automatially.

### NodeSDK list all local groups after get group info
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/listall | jq

result:
{
  "groups": [
    {
      "group_id": "80ca64d5-de6f-4a92-9c3d-43390b22fdfc",
      "group_name": "my_test_group",
      "sign_alias": "my_signature",
      "encrypt_alias": "my_encrypt",
      "user_eth_addr": "045c494662152fa5b9c31e06de316b1ad3a9bafe101d7ed314dbb92499490c7e5c3ce3e78db471dc302d05a8418684b588660039358f5f6399c838c01d5ffffd2a",
      "consensus_type": "POA",
      "encryption_type": "PUBLIC",
      "cipher_key": "9aecbdc3010ba4856c59d8c750615c605bcb775f9a2c4fcdeca04317802f56fd",
      "app_key": "test_app",
      "last_updated": 1653935114735942400,
      "highest_height": 4,
      "highest_block_id": "fa637f55-a7d2-4475-affe-fabf76034d6b",
      "chain_apis": [
        "http://127.0.0.1:8002"
      ]
    }
  ]
}

### NodeSDK list group by groupid
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/80ca64d5-de6f-4a92-9c3d-43390b22fdfc/list | jq

API:
  /nodesdk_api/v1/group/list

params:
  group_id : group id

result:
 {
  "group_id": "93502b34-b3ec-43bf-b96a-2b1f42ef6e06",
  "group_name": "my_test_group",
  "sign_alias": "my_signature",
  "encrypt_alias": "my_encrypt",
  "user_eth_addr": "047e067a32cbb677286087868f00161b909616884906400c76bfff3e6541699370a97036e3bfeea16428f95ee60b4713da6054bbbc33a9f28749e913f7adff4cef",
  "consensus_type": "POA",
  "encryption_type": "PUBLIC",
  "cipher_key": "7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c",
  "app_key": "test_app",
  "last_updated": 0,
  "highest_height": 0,
  "highest_block_id": "",
  "chain_apis": [
    "http://127.0.0.1:8002"
  ]
}

### NodeSDK get group seed
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/80ca64d5-de6f-4a92-9c3d-43390b22fdfc/seed | jq

API:
  /nodesdk_api/v1/group/seed

params:
  group_id : group id

result:
"{\"genesis_block\":{\"BlockId\":\"cb25698d-dd90-4f18-bbae-f22233256a39\",\"GroupId\":\"93502b34-b3ec-43bf-b96a-2b1f42ef6e06\",\"ProducerPubKey\":\"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==\",\"Hash\":\"Xu8/VIr37w/TrmPsf95lmhwVTtGFP4ZFxj/hvM03RJA=\",\"Signature\":\"MEQCIANcuz+opbZfpUqLTT2os9sEG3tqOelMqXhN6H559UwjAiAhFhaiSzuspWpv1wtRrokDyvaKjVgoj9gnNnh0ZUN/zQ==\",\"TimeStamp\":\"1652985871447913818\"},\"group_id\":\"93502b34-b3ec-43bf-b96a-2b1f42ef6e06\",\"group_name\":\"my_test_group\",\"owner_pubkey\":\"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==\",\"consensus_type\":\"poa\",\"encryption_type\":\"public\",\"cipher_key\":\"7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c\",\"app_key\":\"test_app\",\"signature\":\"3045022100e1b793c1ff8e02f93c4e54debb39de23a0f19c69f088bdf62354ac233b0f0341022039223bd22f08db98ad7c0af7aeb6b597c5bcda4712772b210c395f6f93fdfe66\"}"

### NodeSDK post to group
curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"fae1a2a3-6453-4762-9bcb-f1bf030b3ec5", "type":"Group"}}'  http://127.0.0.1:6002/nodesdk_api/v1/group/content

API:
  /nodesdk_api/v1/group/content

Params:
  same as chainSDK

Result:
  {
    "trx_id":"8277955f-ec2a-4901-b62e-0e3f70863044",
    "err_info":"OK"
  }

trx_id: trx id
  * After post, chainSDK will start produce new block to package this trx, it takes 10s to finished, nodesdk should query trx info with trx_id from result till success get trx info from chainsdk, otherwise nodesdk should try post again. 

### NodeSDK get trx info

curl -X GET -H 'Content-Type: application/json' -d {} http://127.0.0.1:6002/nodesdk_api/v1/trx/fae1a2a3-6453-4762-9bcb-f1bf030b3ec5/f0c6c39f-d451-4cca-90d2-875dc7b6c239 | jq


API:
 /nodesdk_api/v1/trx/<group_id>/<trx_id>

result:

{
  "TrxId": "f0c6c39f-d451-4cca-90d2-875dc7b6c239",
  "Type": "POST",
  "GroupId": "fae1a2a3-6453-4762-9bcb-f1bf030b3ec5",
  "Data": "ktDQD1YRTmCpDKoqAf4lk+lymozmtJMPplJHThCfRwCPYV6lTG8ZesTAHTaSc8LR6q57H7EtHiByEsAihnb99siBAfmy73xwrxghyL2glSSiydbqaAWSIPC56jwA04blqaMavH5ewwE/OCfjf3YCHQ==",
  "TimeStamp": "1653592748642466862",
  "Version": "1.0.0",
  "Expired": "1653592778642466912",
  "ResendCount": "0",
  "Nonce": "6",
  "SenderPubkey": "04a8d571195dfc2accac482dc1940e663e988a5a6f932a5aaa3cf6c5add1f40aa76e4c00aecebce0530f4d6a99a06364f6fdab767b6a631fbcaa45e374319a1ed2",
  "SenderSign": "MEYCIQCKh+XO22ETgkYq+0J7xMZ5//ujudDaGYbl4MgysBYi1QIhAOPuVMB7jW/3i5cVNWHXjepJ+6j9RWuS3Ia5CaoJsvQ3",
  "StorageType": "CHAIN"
}

### NodeSDK get block 

curl -X GET -H 'Content-Type: application/json' -d {} http://127.0.0.1:5216/nodesdk_api/v1/block/fae1a2a3-6453-4762-9bcb-f1bf030b3ec5/815ae44a-bef8-43e2-9af5-62451e85b8c8 | jq

API:
 /nodesdk_api/v1/trx/<group_id>/<block_id>

{
  "BlockId": "815ae44a-bef8-43e2-9af5-62451e85b8c8",
  "GroupId": "fae1a2a3-6453-4762-9bcb-f1bf030b3ec5",
  "PrevBlockId": "6ee607f1-b3d7-4471-a42f-e12da407b4b4",
  "PreviousHash": "ayQras7VNjOhHOy1NXWIYMoVd/q0pbrfPEbvBBKx/FE=",
  "Trxs": [
    {
      "TrxId": "8277955f-ec2a-4901-b62e-0e3f70863044",
      "GroupId": "fae1a2a3-6453-4762-9bcb-f1bf030b3ec5",
      "Data": "Iz0KH0FVgNq9D2Z2YrQg1M3BYQd6sqeVSOuNA3C5NQVFr/jdmhY0II1dLyzsh6sH8r88kCFu5WCA0QaGIZkrUSgEl8A94yDVIzJ2bm47xlQYWgrRnwINyf8amsK0pkztWLUVFV66hlihsoksVkeBkw==",
      "TimeStamp": "1653594366233701647",
      "Version": "1.0.0",
      "Expired": 1653594396233701600,
      "Nonce": 8,
      "SenderPubkey": "04a8d571195dfc2accac482dc1940e663e988a5a6f932a5aaa3cf6c5add1f40aa76e4c00aecebce0530f4d6a99a06364f6fdab767b6a631fbcaa45e374319a1ed2",
      "SenderSign": "MEUCIQDgo+ATJbebWSYoDlxx6zNhHRwBz0lipFqmN9V3eT1mDwIgGJwuyi8naMltgvgRs7o/7+P1hQP2T5A+Nga/aAdTfY0="
    }
  ],
  "ProducerPubKey": "CAISIQJQR9aBGKpOONEoIWVKENiCEaDXXNZyoyHwHaQ2WqA/tw==",
  "Hash": "6PIdwTkzJgkb8i60JJvorMP9YW26npgFoLqOi2VbuyU=",
  "Signature": "MEQCIDfU6P6+vWgdvwQIzPNfdx1XzOg+rYFo+G9W0CwPgkFbAiAseQ8hEzKmk7XNIxQLuery3lo2linl57zkOHf6Eqdc0A==",
  "TimeStamp": "1653594371238961928"
}

 ### NodeSDK get content
 curl -X POST -H 'Content-Type: application/json'  -d '{"group_id":"3e91ed4e-e36c-4beb-a290-e20b84e31b76", "num":20, "start_trx":"cfb7ee1d-7eb3-49f6-8659-d5256b631262", "reverse":"false", "include_start_trx":"false"}'  http://127.0.0.1:6002/nodesdk_api/v1/group/getctn | jq

API:
  /nodesdk_api/v1/group/getctn 

Params:
  Same as chainSDK

Result:
[
  {
    "TrxId": "5a590b72-3577-4989-a0e3-b50cabf5063a",
    "Publisher": "045c494662152fa5b9c31e06de316b1ad3a9bafe101d7ed314dbb92499490c7e5c3ce3e78db471dc302d05a8418684b588660039358f5f6399c838c01d5ffffd2a",
    "Content": "N3QPdHmLot2ADOWBsB84GQkUkQVl7jVAlZRFf8mCp8pi51d+q2t+TeZ95Lj03nC7iSPUv4QnPMzWvo4A/ROo/Tv4iHQ87HX1guIQ1KC8nff2ksLbl+RqxfXEaaLpQwbSfkOyXLAaKFE48EmMnbOEEw==",
    "TimeStamp": 1653933252237444400
  }
]

### NodeSDK get producers
 curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:6002/nodesdk_api/v1/group/80ca64d5-de6f-4a92-9c3d-43390b22fdfc/producers | jq

 API:
  /nodesdk_api/v1/group/<group_id>/producers

Result:
"[{\"ProducerPubkey\":\"CAISIQKBxdGuEjsljYbPgfbZTti8NAoFbW+Sh8YvCFF/PRqH4A==\",\"OwnerPubkey\":\"CAISIQKBxdGuEjsljYbPgfbZTti8NAoFbW+Sh8YvCFF/PRqH4A==\",\"OwnerSign\":\"3046022100dda6a1f8b2fa56d71eb0e2a030f708de7d8e872587d291fb05c3a73a16f8826e022100c77b5a63ea408a526a33dd32bef16015a3dfb17e2bd54dc4e30af21505c2b649\",\"TimeStamp\":1653931807609873910,\"BlockProduced\":3}]\n"


### NodeSDK list appconfig key
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/80ca64d5-de6f-4a92-9c3d-43390b22fdfc/appconfig/keylist

 API:
  /nodesdk_api/v1/group/<group_id>/appconfig/keylist

Result:
"[{\"Name\":\"test_bool\",\"Type\":\"BOOL\"}]\n"

### NodeSDK get appconfig by key name 
 curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:6002/nodesdk_api/v1/group/80ca64d5-de6f-4a92-9c3d-43390b22fdfc/appconfig/test_bool | jq

 API:
  /nodesdk_api/v1/group/<group_id>/appconfig/<key>

Result:
 "{\"Name\":\"test_bool\",\"Type\":\"BOOL\",\"Value\":\"false\",\"OwnerPubkey\":\"CAISIQKBxdGuEjsljYbPgfbZTti8NAoFbW+Sh8YvCFF/PRqH4A==\",\"OwnerSign\":\"3046022100bf921a9fd48fca33092cff445680e6f50431354951183dff176fdfebfb160eef022100d8b903fe5a67f1934e2e1229a9b923496146cd885c98c2200def2f3b0b7ee2c6\",\"Memo\":\"add test_bool to group\",\"TimeStamp\":1653935104730944664}\n"
