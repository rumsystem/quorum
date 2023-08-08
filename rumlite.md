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

