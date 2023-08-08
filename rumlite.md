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
7. owner_keyname : who is the owner of this group, given by keyname and the keyname MUST be existed in local keystore
8. trx_sign_keyname : for this group, a node will use this key to sign the trx send by itself, this key is the "identity" of a node for the group
9. neoproducer_sign_keyname : the key for the first (neo) group producer, genesis block will be created and signed by using this key, if no change consensus, all other blocks will also be created and signed by this key
10. url: a url point some where (for example the developer or app's website)

curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"4c0bd5c5-35b6-43b4-92a7-e067a8e7865e", "app_name":"dummy_app_name", "consensus_type":"poa", "sync_type":"public", "encrypt_trx":true, "ctn_type":"blob", "epoch_duration":1000, "owner_keyname":"my_test_app_owner_key", "trx_sign_keyname":"my_test_app_sign_key", "neoproducer_sign_keyname":"my_test_app_producer_key", "url":"dummy_url_point_to_mywebsite"}' http://127.0.0.1:8002/api/v2/rumlite/group/newseed | jq

result: 

{
  "group_id": "e5ff7b25-3c41-4a26-b9cb-9de950d1558f",
  "owner_keyname": "my_test_app_owner_key",
  "trx_sign_keyname": "my_test_app_sign_key",
  "producer_sign_keyname": "my_test_app_producer_key",
  "seed": {
    "Group": {
      "GroupId": "e5ff7b25-3c41-4a26-b9cb-9de950d1558f",
      "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
      "AppName": "dummy_app_name",
      "OwnerPubKey": "AgJQoeSpqx-3bbeqg9Onw5j1IgnmjXQg-3oErBmc1Oli",
      "TrxSignPubkey": "A2gAvNbJexiJk3cjiaXtc5cmvIGgp5WzWUZmVq5VlvG1",
      "CipherKey": "9c076e6a0f73a01d9f10b591fc512f85fb1dbc4341d73b60334b34b8f1ae0156",
      "EncryptTrxCtn": true,
      "SyncType": 1,
      "ConsensusInfo": {
        "Poa": {
          "ConsensusId": "8540e636-2688-4816-b126-0709153417f5",
          "EpochDuration": 1000,
          "Producers": [
            "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY"
          ]
        }
      },
      "GenesisBlock": {
        "GroupId": "e5ff7b25-3c41-4a26-b9cb-9de950d1558f",
        "TimeStamp": "1691512036823929857",
        "ProducerPubkey": "AgJQoeSpqx-3bbeqg9Onw5j1IgnmjXQg-3oErBmc1Oli",
        "ConsensusInfo": {
          "Poa": {
            "ConsensusId": "8540e636-2688-4816-b126-0709153417f5",
            "EpochDuration": 1000,
            "Producers": [
              "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY"
            ]
          }
        },
        "BlockHash": "oHkksiqUZTviJGw6EH7ldQyDQCOePEKMK6bEShgFz7o=",
        "ProducerSign": "N+NQX2XnyRyDDsft7yd8g9RTpK8Xv3Wr4E5XNakMwxlKH/23JljTClM36VTrM10AK0qC5Rp3OFeclxpNY1ClCQA="
      },
      "LastUpdate": 1691512036823929900
    },
    "Hash": "tt2hYQ5aAbz9FzYiGcqVJeVEsxwpwVHrRCztIGhfr6M=",
    "Signature": "RLrqNuUz6UfPExcx2IUO52ZuKj9Gsue4kmAjYQo9asxehFbhn86bKx5JdU56uMz8CZ77f9yyAmuCt8uSgf5lIAE="
  },
  "seed_byts": "CoQFCiRlNWZmN2IyNS0zYzQxLTRhMjYtYjljYi05ZGU5NTBkMTU1OGYSJDRjMGJkNWM1LTM1YjYtNDNiNC05MmE3LWUwNjdhOGU3ODY1ZRoOZHVtbXlfYXBwX25hbWUiLEFnSlFvZVNwcXgtM2JiZXFnOU9udzVqMUlnbm1qWFFnLTNvRXJCbWMxT2xpKixBMmdBdk5iSmV4aUprM2NqaWFYdGM1Y212SUdncDVXeldVWm1WcTVWbHZHMTJAOWMwNzZlNmEwZjczYTAxZDlmMTBiNTkxZmM1MTJmODVmYjFkYmM0MzQxZDczYjYwMzM0YjM0YjhmMWFlMDE1NjgBQAFaWQpXCiQ4NTQwZTYzNi0yNjg4LTQ4MTYtYjEyNi0wNzA5MTUzNDE3ZjUY6AciLEFxb3pQemhnWXZJVXFCNnFiaFFZS0FocW16T25QWWRjUTNENUl2WkVrNE1ZYp4CEiRlNWZmN2IyNS0zYzQxLTRhMjYtYjljYi05ZGU5NTBkMTU1OGYogajAwcSn3bwXMixBZ0pRb2VTcHF4LTNiYmVxZzlPbnc1ajFJZ25talhRZy0zb0VyQm1jMU9saTpZClcKJDg1NDBlNjM2LTI2ODgtNDgxNi1iMTI2LTA3MDkxNTM0MTdmNRjoByIsQXFvelB6aGdZdklVcUI2cWJoUVlLQWhxbXpPblBZZGNRM0Q1SXZaRWs0TVlCIKB5JLIqlGU74iRsOhB+5XUMg0AjnjxCjCumxEoYBc+6SkE341BfZefJHIMOx+3vJ3yD1FOkrxe/davgTlc1qQzDGUof/bcmWNMKUzfpVOszXQArSoLlGnc4V5yXGk1jUKUJAGiBqMDBxKfdvBcSILbdoWEOWgG8/Rc2IhnKlSXlRLMcKcFR60Qs7SBoX6+jGkFEuuo25TPpR88TFzHYhQ7nZm4qP0ay57iSYCNhCj1qzF6EVuGfzpsrHkl1Tnq4zPwJnvt/3LICa4K3y5KB/mUgAQ=="
}

-. seed_byts is used for share the group
-. all other items is for app developer to use

when create a group, the owner_keyname, trx_sign_keyname, neoproducer_keyname are all optional, if no keyname is given, a new keypair and key name will be created for you when create the group seed