Dev environment setup

# Run full node (chainsdk)
 RUM_KSPASSWD=123 go run cmd/main.go -peername n1 -listen /ip4/127.0.0.1/tcp/7002 -apilisten :8002 -peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAkwZ53wCxAecczHiypKFJhWXwfP1G87n8G5R4i5qhszy8v -keystoredir n1keystore -jsontracer n1tracer.json --debug true

# Run light node (nodesdk)
RUM_KSPASSWD=123 go run ./pkg/nodesdk/cmd/main.go -peername nodesdk -listen /ip4/127.0.0.1/tcp/6002  -keystoredir nodesdkkeystore --debug true

# KeyStore 
## Create new signature keypair with Alias
curl -k -X POST -H 'Content-Type: application/json' -d '{"alias":"my_signature", "type":"sign"}' https://127.0.0.1:5216/nodesdk_api/v1/keystore/create | jq

Result:
{
  "Alias": "my_signature",
  "Keyname": "d69b8175-31af-4a9c-8177-4ff64b1e3a64",
  "KeyType": "sign",
  "Pubkey": "04ed309b6cc7f175bab9d467b1eaa3ca2bad9ff5e42a6d0add884e45a0ead039225cca5c1d5afb3589881f05d0cb3392f332c03546af41fff15ce84831dc364e43"
}

## Create new encrypt keypair with Alias
curl -k -X POST -H 'Content-Type: application/json' -d '{"alias":"my_encrypt", "type":"encrypt"}' https://127.0.0.1:5216/nodesdk_api/v1/keystore/create | jq

{
  "Alias": "my_encrypt",
  "Keyname": "c2d14c51-f19d-4094-9ddb-ec7bf7862b9d",
  "KeyType": "encrypt",
  "Pubkey": "age1emlvjpwg3v6vkym54dkwcadzu9h397fuvpz02nvu7qjl6delxgdq3d57rf"
}

## List all keypairs
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:5216/nodesdk_api/v1/keystore/listall | jq

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

## Remove an exist alias (unbind an alias with keypair)
curl -k -X POST -H 'Content-Type: application/json' -d '{"alias":"my_sign_alias"}' https://127.0.0.1:5216/nodesdk_api/v1/keystore/remove | jq

## Rebind alias with other keypair (keyname)
curl -k -X POST -H 'Content-Type: application/json' -d '{"alias":"my_sign_alias", "keyname":"9e183036-ceb1-4ce5-a61e-147779db5ed4", "type":"sign"}' https://127.0.0.1:5216/nodesdk_api/v1/keystore/bindalias | jq


# Group
## Full node create a group
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' https://127.0.0.1:8002/api/v1/group | jq

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

## light node join group
 curl -k -X POST -H 'Content-Type: application/json' -d '{"seed":{"genesis_block":{"BlockId":"cb25698d-dd90-4f18-bbae-f22233256a39","GroupId":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06","ProducerPubKey":"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==","Hash":"Xu8/VIr37w/TrmPsf95lmhwVTtGFP4ZFxj/hvM03RJA=","Signature":"MEQCIANcuz+opbZfpUqLTT2os9sEG3tqOelMqXhN6H559UwjAiAhFhaiSzuspWpv1wtRrokDyvaKjVgoj9gnNnh0ZUN/zQ==","TimeStamp":"1652985871447913818"},"group_id":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06","group_name":"my_test_group","owner_pubkey":"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==","consensus_type":"poa","encryption_type":"public","cipher_key":"7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c","app_key":"test_app","signature":"3045022100e1b793c1ff8e02f93c4e54debb39de23a0f19c69f088bdf62354ac233b0f0341022039223bd22f08db98ad7c0af7aeb6b597c5bcda4712772b210c395f6f93fdfe66"}, "sign_alias":"my_signature", "encrypt_alias":"my_encrypt", "urls":["https://127.0.0.1:8002"]}' https://127.0.0.1:5216/nodesdk_api/v1/group/join

 {"group_id":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06","group_name":"my_test_group","owner_pubkey":"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==","sign_alias":"my_signature","encrypt_alias":"my_signature","consensus_type":"poa","encryption_type":"public","cipher_key":"7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c","app_key":"test_app","signature":"304402206b2ccbe262cc20e3020b4bd9f89b2909b1a07a54403e501d15a7f4cecfdb866e02207a8142721d677a5c1053cc7c897688b89d2edcc2b0217fce109d567c8a8e31af"}

## light node list all groups

curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:5216/nodesdk_api/v1/group/listall | jq

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
        "https://127.0.0.1:8002"
      ]
    }
  ]
}

## Light node list 1 group by groupid
 curl -k -X GET -H 'Content-Type: application/json' -d '{"group_id":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06"}' https://127.0.0.1:5216/nodesdk_api/v1/group/list | jq

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
    "https://127.0.0.1:8002"
  ]
}

## Light node get group seed
curl -k -X GET -H 'Content-Type: application/json' -d '{"group_id":"93502b34-b3ec-43bf-b96a-2b1f42ef6e06"}' https://127.0.0.1:5216/nodesdk_api/v1/group/seed | jq

"{\"genesis_block\":{\"BlockId\":\"cb25698d-dd90-4f18-bbae-f22233256a39\",\"GroupId\":\"93502b34-b3ec-43bf-b96a-2b1f42ef6e06\",\"ProducerPubKey\":\"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==\",\"Hash\":\"Xu8/VIr37w/TrmPsf95lmhwVTtGFP4ZFxj/hvM03RJA=\",\"Signature\":\"MEQCIANcuz+opbZfpUqLTT2os9sEG3tqOelMqXhN6H559UwjAiAhFhaiSzuspWpv1wtRrokDyvaKjVgoj9gnNnh0ZUN/zQ==\",\"TimeStamp\":\"1652985871447913818\"},\"group_id\":\"93502b34-b3ec-43bf-b96a-2b1f42ef6e06\",\"group_name\":\"my_test_group\",\"owner_pubkey\":\"CAISIQL/jjZqqCV//uWoTjF1G1YvYDQTp1p05xudX2NaabSLDg==\",\"consensus_type\":\"poa\",\"encryption_type\":\"public\",\"cipher_key\":\"7d6cce45adcdb58ad10400df079e0a8655df7c633b51e76c796cf0e413827d4c\",\"app_key\":\"test_app\",\"signature\":\"3045022100e1b793c1ff8e02f93c4e54debb39de23a0f19c69f088bdf62354ac233b0f0341022039223bd22f08db98ad7c0af7aeb6b597c5bcda4712772b210c395f6f93fdfe66\"}"

## Light node post to group
curl -k -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"fae1a2a3-6453-4762-9bcb-f1bf030b3ec5", "type":"Group"}}'  https://127.0.0.1:5216/nodesdk_api/v1/group/content

{"trx_id":"8277955f-ec2a-4901-b62e-0e3f70863044","err_info":"OK"}

## Light node get trx info
curl -k -X GET -H 'Content-Type: application/json' -d {} https://127.0.0.1:5216/nodesdk_api/v1/trx/fae1a2a3-6453-4762-9bcb-f1bf030b3ec5/f0c6c39f-d451-4cca-90d2-875dc7b6c239 | jq

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

## Light node get block 
curl -k -X GET -H 'Content-Type: application/json' -d {} https://127.0.0.1:5216/nodesdk_api/v1/block/fae1a2a3-6453-4762-9bcb-f1bf030b3ec5/815ae44a-bef8-43e2-9af5-62451e85b8c8 | jq

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