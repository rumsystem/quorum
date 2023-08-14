- a group needs 3 keys
1. group owner sign key    - the owner of the group, trx sign by this key has the suprior previllage, this key should be used only when necessary
2. group trx sign key      - group user's sign key, use to identify "who are you" in this group, regular trx send to this group should be signed by this key
3. group producer sign key - the "producer" of a group, all blocks belongs to the group should be created and sign by the node who has this key in local keystore

let's try create the first group seed
1. create owner sign key with given keyname
curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"my_test_app_sign_key"}'  http://127.0.0.1:8002/api/v2/rumlite/keystore/createsignkey
{
  "key_alias": "f5aa0cf7-b406-4df4-bb1a-58083d98d5c0",
  "key_name": "my_test_app_sign_key",
  "pubkey": "A2gAvNbJexiJk3cjiaXtc5cmvIGgp5WzWUZmVq5VlvG1"
}

2. create trx sign key with given keyname
curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"my_test_app_sign_key"}'  http://127.0.0.1:8002/api/v2/rumlite/keystore/createsignkey
{
  "key_alias": "f5aa0cf7-b406-4df4-bb1a-58083d98d5c0",
  "key_name": "my_test_app_sign_key",
  "pubkey": "A2gAvNbJexiJk3cjiaXtc5cmvIGgp5WzWUZmVq5VlvG1"
}

3. create producer sign key with given keyname
curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"my_test_app_producer_key"}'  http://127.0.0.1:8002/api/v2/rumlite/keystore/createsignkey
{
  "key_alias": "61bd981b-5559-4580-9220-52b9701d1af9",
  "key_name": "my_test_app_producer_key",
  "pubkey": "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY"
}

Now let's create the first group seed

- parameters
1. app_id : a group should belongs to an "app", even a "dummy_app", a uuid should be provided, the "cellar" will accept/reject  a group seed by using app_id
2. app_name : app_name, app_id and app_name can be identical among different groups, these 2 parameters should be used based on your app design
3. consensus_type : poa or pos, now only poa is supported
4. sync_type: public or privatre, a public group can be synced by any node, sync from a private group is by request (each pubkey)
5. encrypt_trx: if set to true, a cipher key will be created an used to encrypt the "content" of a trx
6. ctn_type : blob or service
    a. a blob group is consider as a "file", after created, user can "upload" a binary file to it and "seal" it by using a "SEAL" type trx
    b. a service is group is "dynamic", a "active" producer is needed to collect trx and build block in realtime
7. owner_keyname : who is the owner of this group, given by keyname and the keyname MUST be existed in local keystoree group
9. neoproducer_sign_keyname : keyname for the first (neo) group producer, genesis block will be created and signed by using the key pair associated with this keyname 
10. url: a url point some where (for example the developer or app's website)

curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"4c0bd5c5-35b6-43b4-92a7-e067a8e7865e", "app_name":"dummy_app_name", "consensus_type":"poa", "sync_type":"public", "encrypt_trx":true, "ctn_type":"blob", "epoch_duration":1000, "owner_keyname":"my_test_app_owner_key", "neoproducer_sign_keyname":"my_test_app_producer_key", "url":"dummy_url_point_to_mywebsite"}' http://127.0.0.1:8002/api/v2/rumlite/group/newseed | jq

result: 
{
  "group_id": "088ac9f5-ec28-41b4-8806-80ed70735862",
  "owner_keyname": "my_test_app_owner_key",
  "producer_sign_keyname": "my_test_app_producer_key",
  "seed": {
    "Group": {
      "GroupId": "088ac9f5-ec28-41b4-8806-80ed70735862",
      "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
      "AppName": "dummy_app_name",
      "OwnerPubKey": "AgJQoeSpqx-3bbeqg9Onw5j1IgnmjXQg-3oErBmc1Oli",
      "CipherKey": "c3196e3079842367f4c59e4f37df6567db452f78e19a7df77bb661ad29428fa0",
      "EncryptTrxCtn": true,
      "SyncType": 1,
      "ConsensusInfo": {
        "Poa": {
          "ConsensusId": "e732ebe9-6f7d-4cf1-87f0-cecc3a8726e1",
          "EpochDuration": 1000,
          "Producers": [
            "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY"
          ]
        }
      },
      "GenesisBlock": {
        "GroupId": "088ac9f5-ec28-41b4-8806-80ed70735862",
        "TimeStamp": "1691520953189144613",
        "ProducerPubkey": "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY",
        "ConsensusInfo": {
          "Poa": {
            "ConsensusId": "e732ebe9-6f7d-4cf1-87f0-cecc3a8726e1",
            "EpochDuration": 1000,
            "Producers": [
              "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY"
            ]
          }
        },
        "BlockHash": "HeWU4rkCbu11wlLq3nuefNyMj2PQ4WpZCKxfGlLrcEs=",
        "ProducerSign": "6AN5rQyf6lMk+Mx3892WdbfJkqVeePxzLjqAzOlZT3QzypuIr2fiFf3oSYPSeVrPPsnjoaJ/XD9y/8M1CLqbgwA="
      },
      "LastUpdate": 1691520953189144600
    },
    "Hash": "9Cfm98zz5WvMqNyc248aOfjhRFFMKTEpDoy8P3EeIF8=",
    "Signature": "Wi6EL+Zd08CjcVDraEUFdHCETewvIw+WCrXzRBXE528gdLFusdUNY2iTiq1Qr9KEJdbR623Z2k+PlFhGBAkFzQA="
  },
  "seed_byts": "CtYECiQwODhhYzlmNS1lYzI4LTQxYjQtODgwNi04MGVkNzA3MzU4NjISJDRjMGJkNWM1LTM1YjYtNDNiNC05MmE3LWUwNjdhOGU3ODY1ZRoOZHVtbXlfYXBwX25hbWUiLEFnSlFvZVNwcXgtM2JiZXFnOU9udzVqMUlnbm1qWFFnLTNvRXJCbWMxT2xpMkBjMzE5NmUzMDc5ODQyMzY3ZjRjNTllNGYzN2RmNjU2N2RiNDUyZjc4ZTE5YTdkZjc3YmI2NjFhZDI5NDI4ZmEwOAFAAVpZClcKJGU3MzJlYmU5LTZmN2QtNGNmMS04N2YwLWNlY2MzYTg3MjZlMRjoByIsQXFvelB6aGdZdklVcUI2cWJoUVlLQWhxbXpPblBZZGNRM0Q1SXZaRWs0TVlingISJDA4OGFjOWY1LWVjMjgtNDFiNC04ODA2LTgwZWQ3MDczNTg2MiilsODHhKvfvBcyLEFxb3pQemhnWXZJVXFCNnFiaFFZS0FocW16T25QWWRjUTNENUl2WkVrNE1ZOlkKVwokZTczMmViZTktNmY3ZC00Y2YxLTg3ZjAtY2VjYzNhODcyNmUxGOgHIixBcW96UHpoZ1l2SVVxQjZxYmhRWUtBaHFtek9uUFlkY1EzRDVJdlpFazRNWUIgHeWU4rkCbu11wlLq3nuefNyMj2PQ4WpZCKxfGlLrcEtKQegDea0Mn+pTJPjMd/PdlnW3yZKlXnj8cy46gMzpWU90M8qbiK9n4hX96EmD0nlazz7J46Gif1w/cv/DNQi6m4MAaKWw4MeEq9+8FxIg9Cfm98zz5WvMqNyc248aOfjhRFFMKTEpDoy8P3EeIF8aQVouhC/mXdPAo3FQ62hFBXRwhE3sLyMPlgq180QVxOdvIHSxbrHVDWNok4qtUK/ShCXW0ett2dpPj5RYRgQJBc0A"
}

-. seed_byts is used for share the group
-. all other items is for app developer to use

when create a group, the owner_keyname and neoproducer_keyname are optional, if no keyname is given, a new keypair and key name will be created for you when create the group seed

curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"4c0bd5c5-35b6-43b4-92a7-e067a8e7865e", "app_name":"dummy_app_name", "consensus_type":"poa", "sync_type":"public", "encrypt_trx":true, "ctn_type":"blob", "epoch_duration":1000, "url":"dummy_url_point_to_mywebsite"}' http://127.0.0.1:8002/api/v2/rumlite/group/newseed | jq

{
  "group_id": "214bcc94-a017-40f1-9e3b-526c9407ab49",
  "owner_keyname": "214bcc94-a017-40f1-9e3b-526c9407ab49",
  "producer_sign_keyname": "214bcc94-a017-40f1-9e3b-526c9407ab49_neoproducer_sign_keyname",
  "seed": {
    "Group": {
      "GroupId": "214bcc94-a017-40f1-9e3b-526c9407ab49",
      "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
      "AppName": "dummy_app_name",
      "OwnerPubKey": "Atvwb57dqRE1a1hUSPHwikyqXbGpIDKpYH8Q2JQ2axzj",
      "CipherKey": "66a239dde166b2561a3892c2fcf5c143f9af097207fae8c752a0875d23a439d9",
      "EncryptTrxCtn": true,
      "SyncType": 1,
      "ConsensusInfo": {
        "Poa": {
          "ConsensusId": "ae7b4d6e-106f-427b-a438-8a3cb43420ed",
          "EpochDuration": 1000,
          "Producers": [
            "AzJhxZzHn1EjIOW74qVodiAjwDOwOtQpo8yTJ7Ce6_IE"
          ]
        }
      },
      "GenesisBlock": {
        "GroupId": "214bcc94-a017-40f1-9e3b-526c9407ab49",
        "TimeStamp": "1691517815695263795",
        "ProducerPubkey": "AzJhxZzHn1EjIOW74qVodiAjwDOwOtQpo8yTJ7Ce6_IE",
        "ConsensusInfo": {
          "Poa": {
            "ConsensusId": "ae7b4d6e-106f-427b-a438-8a3cb43420ed",
            "EpochDuration": 1000,
            "Producers": [
              "AzJhxZzHn1EjIOW74qVodiAjwDOwOtQpo8yTJ7Ce6_IE"
            ]
          }
        },
        "BlockHash": "agflmRrSphyvMFQreMn/OCB/WWhn2F1IfTtakETneqU=",
        "ProducerSign": "XX4sR9HawFA8ABj4CDiFGvJrZ2nGpAezbASThVBTNMQlE1DMAbuXsoLyL0G0OglAPaR1GrvSi7mskFtBqvf/dgA="
      },
      "LastUpdate": 1691517815695263700
    },
    "Hash": "tWxfduXbeA0AbLFkSKGPnKGbqhwrL6cjcWoPfSxlbXg=",
    "Signature": "J17CAWV4EzZ7Xe47objwy+bfiSB9+5ayZM0IgXeoNXlT5y94iZ+yTVUS4EDXECBrRaGHCQkhPdU4jQGN5qIuiwE="
  },
  "seed_byts": "CoQFCiQyMTRiY2M5NC1hMDE3LTQwZjEtOWUzYi01MjZjOTQwN2FiNDkSJDRjMGJkNWM1LTM1YjYtNDNiNC05MmE3LWUwNjdhOGU3ODY1ZRoOZHVtbXlfYXBwX25hbWUiLEF0dndiNTdkcVJFMWExaFVTUEh3aWt5cVhiR3BJREtwWUg4UTJKUTJheHpqKixBNGtsQ1dhUVRPWjF1YUlaMU5kUFRHeUxybjdTOGViZTIyOWxlMFo1RWx2YTJANjZhMjM5ZGRlMTY2YjI1NjFhMzg5MmMyZmNmNWMxNDNmOWFmMDk3MjA3ZmFlOGM3NTJhMDg3NWQyM2E0MzlkOTgBQAFaWQpXCiRhZTdiNGQ2ZS0xMDZmLTQyN2ItYTQzOC04YTNjYjQzNDIwZWQY6AciLEF6Smh4WnpIbjFFaklPVzc0cVZvZGlBandET3dPdFFwbzh5VEo3Q2U2X0lFYp4CEiQyMTRiY2M5NC1hMDE3LTQwZjEtOWUzYi01MjZjOTQwN2FiNDkos5CLvtzP3rwXMixBekpoeFp6SG4xRWpJT1c3NHFWb2RpQWp3RE93T3RRcG84eVRKN0NlNl9JRTpZClcKJGFlN2I0ZDZlLTEwNmYtNDI3Yi1hNDM4LThhM2NiNDM0MjBlZBjoByIsQXpKaHhaekhuMUVqSU9XNzRxVm9kaUFqd0RPd090UXBvOHlUSjdDZTZfSUVCIGoH5Zka0qYcrzBUK3jJ/zggf1loZ9hdSH07WpBE53qlSkFdfixH0drAUDwAGPgIOIUa8mtnacakB7NsBJOFUFM0xCUTUMwBu5eygvIvQbQ6CUA9pHUau9KLuayQW0Gq9/92AGizkIu+3M/evBcSILVsX3bl23gNAGyxZEihj5yhm6ocKy+nI3FqD30sZW14GkEnXsIBZXgTNntd7juhuPDL5t+JIH37lrJkzQiBd6g1eVPnL3iJn7JNVRLgQNcQIGtFoYcJCSE91TiNAY3moi6LAQ=="
}

组的同步类型：

任何人可以加入并同步一个public组的数据
任何人都可以加入一根private组，但是需要经过owner同意才能同步该组的数据

组的内容类型：

blob：静态类型，内容可以增长，但是不可以更换producer和consensus相关的其他参数，不可以fork。blob组的用处是存储文件类型的数据

service：动态类型，内容可以增长，可更换producer和consensus相关的参数（通过fork)。service组的用处是host某种动态服务

酒窖（cella）
酒窖其实也是一个group，类型为service，同步类型可以是public或者private，producer可以是一个或者多个（一旦确定则不可更改，除非停机fork）
酒窖会同步放入其中的所有Seed
酒窖中的所有组会保持打开状态，以随时给不同业务提供block同步或者出块服务
一个酒窖本身的group不能放入其他酒窖
一个酒窖可以同意其他酒窖加入自己并同步酒窖group本身的block

============================================================================================================================

节点，酒窖和种子的互动过程

user story 1：
节点A创建一个blob类型的种子B
- 节点A在本地调用CreateSeed API创建一个种子B


user story2:
节点A向一个Blob类型的种子B添加内容
- 节点A在本地调用JoinGroupBySeed加入group B
- 节点A将内容打散并以POST trx的形式存入 group B （add blocks)
- 节点A在调用CloseGroup关闭group B


user story3:
节点A向一个存在的group B添加内容
- 节点A调用LoadGroupById打开groupB
- 节点A将新内容打散并以POST trx的形式追加到group B
- 节点A在调用Close Group关闭group B

user story4:
节点A创建一个Blob类型的种子B并将B加入酒窖C
-节点A创建group B并添加内容（us1 to us3）
-节点A保持Group B在本地运行
-节点A获取酒窖C的种子
-节点A加载酒窖C的种子，在本地创建一个酒窖C的实例（为了向酒窖C发trx），节点A并不同步酒窖C的block
-节点A向酒窖C的group 发送一条 CELLA_REQ类型的trx，包括
	- Group B的 seed
	-需要酒窖C同步的块数
	-支付凭证（optional）
-节点A持续检查并试图获取自己的CELLA_REQ trx被酒窖C上链（意味着Cella接受并开始同步Group block)
- 酒窖C获取该Trx，检查支付凭证，如果同意同步，则将该Trx上链（add trx to cella group)
	- 通过seed B加入GroupB
	- 试图开始同步
- 如果种子B的类型为private，则节点A需要发送UPD_SYNCER Trx到group B，将Cella C的pubkey加入同步白名单
- Cella在完成同步之后，发送一条 CELLA_RESP TRX 到 Group B和cella group 作为同步完成的证明
** 和节点不同，一个Cella在完成同步某个group后，并不关闭这个group，而且作为一个在线服务，提供该group的block（follow group同步名单设置）

- 如果需要，节点A可以在收到CELLA_RESP后，关闭本地的group B （关闭本地文件）
- 如果需要，节点A可以在收到CELLA_RESP后，关闭本地的group C （退出酒窖）

user story5:
节点A在将blob group B添加到Cella C后，添加 Group B的内容
-节点A在本地调用LoadGroupById打开group B
-节点A向Group B中添加一些新的block (POST trx or other type trx)
-节点A向酒窖C的group发送一条CELLA_REQ类型的trx，包括
	- Group B的seed
	- 需要酒窖C同步的块数 
	- 支付凭证（optional
- Cella在完成同步之后，发送一条 CELLA_RESP TRX 到 Group B和cella group 作为同步完成的证明

user story6:
节点A在将blob group B添加到Cella C后，修改可以同步groupB block的syncer名单
-节点A在本地调用LoadGroupById打开group B
-节点A向Group B中发送一条UPD_SYNCER类型的trx并正确出块，得以更新可以sync本组的pubkey名单
-节点A向酒窖C的group发送一条CELLA_REQ类型的trx，包括
	- Group B的seed
	- 需要酒窖C同步的块数 （包含新打包的UPD_SYNCER trx）
	- 支付凭证（optional）
- Cella在完成同步之后，发送一条 CELLA_RESP TRX 到 Group B和cella group 作为同步完成的证明
- Cella C通过apply trx的方式，更新本地的group B的syncer名单

user story7:
节点A在blob group B同时添加内容和修改syncer 白名单


user story 8：
节点A创建一个Service类型的种子B并将B加入酒窖C
- 节点A在本地创建一个Service类型的种子B
- 节点A加载这个种子，在本地创建Group B
- 节点A自己作为producer（host the group producer key at local keychain)，生产一些group block
...  



节点可能提供的酒窖API
	- 创建一个酒窖（公开/私有）
	- 删除一个酒窖
	- 列出所有酒窖
	- 列出某个酒窖的所有组
	- 列出某个酒窖的所有申请
	- 批准/拒绝某个种子的加入申请
	- 列出一个酒窖里所有group的状态