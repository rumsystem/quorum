Start a bootstrap node
RUM_KSPASSWD=123 go run main.go bootstrapnode --listen /ip4/0.0.0.0/tcp/10666 --loglevel "debug"

you need the bootstrap id for the next step, try find 

  ->bootstrap host created, ID:<16Uiu2HAm9w95mPtMLghqw6c2Zua7rX36zJAd7bMRonUvS7R9d4w2>

Start the first rumlite node "o1"
  
  RUM_KSPASSWD=123 go run main.go rumlitenode --peername o1 --listen /ip4/127.0.0.1/tcp/7002 --apiport 8002 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm9w95mPtMLghqw6c2Zua7rX36zJAd7bMRonUvS7R9d4w2 --configdir config --datadir data --keystoredir o1keystore  --loglevel "debug"

Now we can create the group seed

- a group needs 3 keys
  1. group owner sign key    - the owner of the group, trx sign by this key has the suprior previllage, this key should be used only when necessary
  2. group trx sign key      - group user's sign key, use to identify "who are you" in this group, after you join a group, trx send to this group should be signed by this key
  3. group producer sign key - the "producer" of a group, all blocks in this group should be created and sign by the node who has this key in local keystore

Create keys

1. create owner key with given keyname
curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"my_test_app_owner_key"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey
{
  "key_alias": "f5aa0cf7-b406-4df4-bb1a-58083d98d5c0",
  "key_name": "my_test_app_owner_key",
  "pubkey": "A2gAvNbJexiJk3cjiaXtc5cmvIGgp5WzWUZmVq5VlvG1"
}

result for createsignkey api has 3 parameters:
  key_alias: UUID for the newly created key
  key_name:  key_name
  pubkey:    pubkey 

2. create trx sign key with given keyname
curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"my_test_app_sign_key"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey
{
  "key_alias": "f5aa0cf7-b406-4df4-bb1a-58083d98d5c0",
  "key_name": "my_test_app_sign_key",
  "pubkey": "A2gAvNbJexiJk3cjiaXtc5cmvIGgp5WzWUZmVq5VlvG1"
}

3. create producer sign key with given keyname
curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"my_test_app_producer_key"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey
{
  "key_alias": "61bd981b-5559-4580-9220-52b9701d1af9",
  "key_name": "my_test_app_producer_key",
  "pubkey": "AqozPzhgYvIUqB6qbhQYKAhqmzOnPYdcQ3D5IvZEk4MY"
}

You List all key pairs from local keystore
curl -X GET -H 'Content-Type: application/json'  -d '{}' http://127.0.0.1:8002/api/v2/keystore/getallkeys

result:
{
  "keys_list": [
    {
      "pubkey": "A4wTJWRtunlQ15fjwxUJUxySfNaoYuYYnhPELSo7ZiG0",
      "key_name": "35a451d1-60dc-4503-bf30-ffb7a4013a61",
      "alias": []
    },
    ...

    {
      "pubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
      "key_name": "my_test_app_owner_key",
      "alias": [
        "714fb1a5-e3c2-4281-b318-4885c900f4d2"
      ]
    },
    {
      "pubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
      "key_name": "my_test_app_producer_key",
      "alias": [
        "184bd896-faa8-4bea-a9ff-280d769e8432"
      ]
    },
    {
      "pubkey": "AkO8otfcqU5nYPyrvWLY3ypdglA5GXW-pYjYmTuJfOMU",
      "key_name": "my_test_app_sign_key",
      "alias": [
        "7df85dfc-0b11-4c71-bc31-f56c18633890"
      ]
    },
    {
      "pubkey": "AhUoPM_ak59Z53_wypZ-fLyqr3khfdyCSdMYaa9WhiPQ",
      "key_name": "my_test_app_trx_sign_key",
      "alias": [
        "7acc1940-0ad4-4bd1-952e-3a4abf78ec0a"
      ] 
    }
  ]
}

List key pair by given keyname
curl -X GET -H 'Content-Type: application/json'  -d '{"key_name":"my_test_app_trx_sign_key"}' http://127.0.0.1:8002/api/v2/keystore/getkeybykeyname

result:
{
  "pubkey": "AhUoPM_ak59Z53_wypZ-fLyqr3khfdyCSdMYaa9WhiPQ",
  "key_name": "my_test_app_trx_sign_key",
  "alias": [
    "7acc1940-0ad4-4bd1-952e-3a4abf78ec0a"
  ]
}

Now let's create the first group seed
curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"4c0bd5c5-35b6-43b4-92a7-e067a8e7865e", "app_name":"dummy_app", "group_name":"index_group", "consensus_type":"poa", "sync_type":"public", "epoch_duration":5000, "owner_keyname":"my_test_app_owner_key", "neoproducer_sign_keyname":"my_test_app_producer_key", "url":"dummy_url_point_to_mywebsite"}' http://127.0.0.1:8002/api/v2/group/newseed | jq

- parameters
1. app_id : a group should belongs to an "app", even a "dummy_app", a uuid should be provided, the "cellar" will accept/reject  a group seed by using app_id
2. app_name : app_name, app_id and app_name can be identical among different groups, these 2 parameters should be used based on your app design
3. consensus_type : poa or pos, now only poa is supported
4. sync_type: public or privatre, a public group can be synced by any node, sync from a private group is by request (each pubkey)
5. owner_keyname : who is the owner of this group, given by keyname and the keyname MUST be existed in local keystoree group
6. neoproducer_sign_keyname : keyname for the first (neo) group producer, genesis block will be created and signed by using the key pair associated with this keyname
7. epoch_length: for how long the producer will wait to propose trxs in an epoch (in ms)
7. url: a url point some where (for example the developer or app's website)

result:
{
  "group_id": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
  "owner_keyname": "my_test_app_owner_key",
  "producer_sign_keyname": "my_test_app_producer_key",
  "seed": {
    "GenesisBlock": {
      "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
      "ProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
      "TimeStamp": "1693419634998367277",
      "Consensus": {
        "Data": "CiRhNjZlZTBmMi1hYjY4LTQ5ZGYtYWU5OS1iMzNkNzUzN2E4MzEiZQokNjE3YzM5ZTQtNGQ2OS00MTlhLWJiYTYtZmJhZjlkMzVhZmIwKIgnMixBc0RFOHZhUUU4S3F3S1BrdTg0S3FRZENXMS1fNW1ab3Q4VjdfWFFiTllBZDoMSW5pdGlhbCBGb3Jr"
      },
      "BlockHash": "VruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgAM=",
      "ProducerSign": "5aac/iXJdxlNNa5ZuVFLitXqXpklNeA6/Zu5TAoqVR1s2KZOT+r9cLdvQoyl5iZNDkMyetZoK7Ag9+7mtziycwE="
    },
    "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
    "GroupName": "index_group",
    "OwnerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
    "SyncType": 1,
    "CipherKey": "a4f74bc8a97f3f8ebc51222713fce0aa94d4994c9214f17bfee9bd6afc52d2d2",
    "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
    "AppName": "dummy_app",
    "Hash": "22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9Jacs=",
    "Signature": "1/+/8VoMpIMSoJLJ6eGu+AaUEccvXg+zikL6jHGJMcsrTGsoFBMdxyDEl73JV8svSmuIA2YIrT0gVjTe2bFDQwE="
  },
  "seed_byts": "CtYCCiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAiLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMK2o/735hY/AFzqQARKNAQokYTY2ZWUwZjItYWI2OC00OWRmLWFlOTktYjMzZDc1MzdhODMxImUKJDYxN2MzOWU0LTRkNjktNDE5YS1iYmE2LWZiYWY5ZDM1YWZiMCiIJzIsQXNERTh2YVFFOEtxd0tQa3U4NEtxUWRDVzEtXzVtWm90OFY3X1hRYk5ZQWQ6DEluaXRpYWwgRm9ya0IgVruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgANKQeWmnP4lyXcZTTWuWblRS4rV6l6ZJTXgOv2buUwKKlUdbNimTk/q/XC3b0KMpeYmTQ5DMnrWaCuwIPfu5rc4snMBEiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAaC2luZGV4X2dyb3VwIixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSgBMkBhNGY3NGJjOGE5N2YzZjhlYmM1MTIyMjcxM2ZjZTBhYTk0ZDQ5OTRjOTIxNGYxN2JmZWU5YmQ2YWZjNTJkMmQyOiQ0YzBiZDVjNS0zNWI2LTQzYjQtOTJhNy1lMDY3YThlNzg2NWVCCWR1bW15X2FwcEog22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9JactSQdf/v/FaDKSDEqCSyenhrvgGlBHHL14Ps4pC+oxxiTHLK0xrKBQTHccgxJe9yVfLL0priANmCK09IFY03tmxQ0MB"
}

-. seed_byts is used for
  1. share the group 
  2. provide seed_byts when register your group to a  cella
-. all other items is for app developer to use


when create a group, the owner_keyname and neoproducer_keyname are optional, if no keyname is given, a new keypair and key name will be created for you when create the group seed

curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"4c0bd5c5-35b6-43b4-92a7-e067a8e7865e", "app_name":"dummy_app", "group_name":"index_group", "consensus_type":"poa", "sync_type":"public", "epoch_duration":5000, "url":"dummy_url_point_to_mywebsite"}' http://127.0.0.1:8002/api/v2/group/newseed | jq

result
{
  "group_id": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
  "owner_keyname": "my_test_app_owner_key",
  "producer_sign_keyname": "my_test_app_producer_key",
  "seed": {
    "GenesisBlock": {
      "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
      "ProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
      "TimeStamp": "1693419634998367277",
      "Consensus": {
        "Data": "CiRhNjZlZTBmMi1hYjY4LTQ5ZGYtYWU5OS1iMzNkNzUzN2E4MzEiZQokNjE3YzM5ZTQtNGQ2OS00MTlhLWJiYTYtZmJhZjlkMzVhZmIwKIgnMixBc0RFOHZhUUU4S3F3S1BrdTg0S3FRZENXMS1fNW1ab3Q4VjdfWFFiTllBZDoMSW5pdGlhbCBGb3Jr"
      },
      "BlockHash": "VruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgAM=",
      "ProducerSign": "5aac/iXJdxlNNa5ZuVFLitXqXpklNeA6/Zu5TAoqVR1s2KZOT+r9cLdvQoyl5iZNDkMyetZoK7Ag9+7mtziycwE="
    },
    "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
    "GroupName": "index_group",
    "OwnerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
    "SyncType": 1,
    "CipherKey": "a4f74bc8a97f3f8ebc51222713fce0aa94d4994c9214f17bfee9bd6afc52d2d2",
    "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
    "AppName": "dummy_app",
    "Hash": "22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9Jacs=",
    "Signature": "1/+/8VoMpIMSoJLJ6eGu+AaUEccvXg+zikL6jHGJMcsrTGsoFBMdxyDEl73JV8svSmuIA2YIrT0gVjTe2bFDQwE="
  },
  "seed_byts": "CtYCCiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAiLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMK2o/735hY/AFzqQARKNAQokYTY2ZWUwZjItYWI2OC00OWRmLWFlOTktYjMzZDc1MzdhODMxImUKJDYxN2MzOWU0LTRkNjktNDE5YS1iYmE2LWZiYWY5ZDM1YWZiMCiIJzIsQXNERTh2YVFFOEtxd0tQa3U4NEtxUWRDVzEtXzVtWm90OFY3X1hRYk5ZQWQ6DEluaXRpYWwgRm9ya0IgVruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgANKQeWmnP4lyXcZTTWuWblRS4rV6l6ZJTXgOv2buUwKKlUdbNimTk/q/XC3b0KMpeYmTQ5DMnrWaCuwIPfu5rc4snMBEiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAaC2luZGV4X2dyb3VwIixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSgBMkBhNGY3NGJjOGE5N2YzZjhlYmM1MTIyMjcxM2ZjZTBhYTk0ZDQ5OTRjOTIxNGYxN2JmZWU5YmQ2YWZjNTJkMmQyOiQ0YzBiZDVjNS0zNWI2LTQzYjQtOTJhNy1lMDY3YThlNzg2NWVCCWR1bW15X2FwcEog22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9JactSQdf/v/FaDKSDEqCSyenhrvgGlBHHL14Ps4pC+oxxiTHLK0xrKBQTHccgxJe9yVfLL0priANmCK09IFY03tmxQ0MB"

join the group just created

curl -X POST -H 'Content-Type: application/json' -d '{"seed":"CtYCCiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAiLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMK2o/735hY/AFzqQARKNAQokYTY2ZWUwZjItYWI2OC00OWRmLWFlOTktYjMzZDc1MzdhODMxImUKJDYxN2MzOWU0LTRkNjktNDE5YS1iYmE2LWZiYWY5ZDM1YWZiMCiIJzIsQXNERTh2YVFFOEtxd0tQa3U4NEtxUWRDVzEtXzVtWm90OFY3X1hRYk5ZQWQ6DEluaXRpYWwgRm9ya0IgVruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgANKQeWmnP4lyXcZTTWuWblRS4rV6l6ZJTXgOv2buUwKKlUdbNimTk/q/XC3b0KMpeYmTQ5DMnrWaCuwIPfu5rc4snMBEiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAaC2luZGV4X2dyb3VwIixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSgBMkBhNGY3NGJjOGE5N2YzZjhlYmM1MTIyMjcxM2ZjZTBhYTk0ZDQ5OTRjOTIxNGYxN2JmZWU5YmQ2YWZjNTJkMmQyOiQ0YzBiZDVjNS0zNWI2LTQzYjQtOTJhNy1lMDY3YThlNzg2NWVCCWR1bW15X2FwcEog22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9JactSQdf/v/FaDKSDEqCSyenhrvgGlBHHL14Ps4pC+oxxiTHLK0xrKBQTHccgxJe9yVfLL0priANmCK09IFY03tmxQ0MB", "user_sign_keyname":"my_test_app_sign_key"}' http://127.0.0.1:8002/api/v2/group/joingroupbyseed

parameters:
  "seed" : seed_byts
  "user_sign_keyname": user_sign_keyname is the key you will use to sign all trx (send by you) in this group, it works like your "identity" in this group, keyaname must be exit in local keystore

result:
{
  "groupItem": {
    "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
    "GroupName": "index_group",
    "OwnerPubKey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
    "UserSignPubkey": "AkO8otfcqU5nYPyrvWLY3ypdglA5GXW-pYjYmTuJfOMU",
    "LastUpdate": 1693425120194197436,
    "GenesisBlock": {
      "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
      "ProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
      "TimeStamp": "1693419634998367277",
      "Consensus": {
        "Data": "CiRhNjZlZTBmMi1hYjY4LTQ5ZGYtYWU5OS1iMzNkNzUzN2E4MzEiZQokNjE3YzM5ZTQtNGQ2OS00MTlhLWJiYTYtZmJhZjlkMzVhZmIwKIgnMixBc0RFOHZhUUU4S3F3S1BrdTg0S3FRZENXMS1fNW1ab3Q4VjdfWFFiTllBZDoMSW5pdGlhbCBGb3Jr"
      },
      "BlockHash": "VruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgAM=",
      "ProducerSign": "5aac/iXJdxlNNa5ZuVFLitXqXpklNeA6/Zu5TAoqVR1s2KZOT+r9cLdvQoyl5iZNDkMyetZoK7Ag9+7mtziycwE="
    },
    "SyncType": 1,
    "CipherKey": "a4f74bc8a97f3f8ebc51222713fce0aa94d4994c9214f17bfee9bd6afc52d2d2",
    "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
    "AppName": "dummy_app"
  }
}


You can get the group seed
curl -X GET -H 'Content-Type: application/json'  -d '{}'  http://127.0.0.1:8002/api/v1/group/617c39e4-4d69-419a-bba6-fbaf9d35afb0/seed
result:
{
  "seed": {
    "GenesisBlock": {
      "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
      "ProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
      "TimeStamp": "1693419634998367277",
      "Consensus": {
        "Data": "CiRhNjZlZTBmMi1hYjY4LTQ5ZGYtYWU5OS1iMzNkNzUzN2E4MzEiZQokNjE3YzM5ZTQtNGQ2OS00MTlhLWJiYTYtZmJhZjlkMzVhZmIwKIgnMixBc0RFOHZhUUU4S3F3S1BrdTg0S3FRZENXMS1fNW1ab3Q4VjdfWFFiTllBZDoMSW5pdGlhbCBGb3Jr"
      },
      "BlockHash": "VruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgAM=",
      "ProducerSign": "5aac/iXJdxlNNa5ZuVFLitXqXpklNeA6/Zu5TAoqVR1s2KZOT+r9cLdvQoyl5iZNDkMyetZoK7Ag9+7mtziycwE="
    },
    "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
    "GroupName": "index_group",
    "OwnerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
    "SyncType": 1,
    "CipherKey": "a4f74bc8a97f3f8ebc51222713fce0aa94d4994c9214f17bfee9bd6afc52d2d2",
    "AppId": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
    "AppName": "dummy_app",
    "Hash": "22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9Jacs=",
    "Signature": "1/+/8VoMpIMSoJLJ6eGu+AaUEccvXg+zikL6jHGJMcsrTGsoFBMdxyDEl73JV8svSmuIA2YIrT0gVjTe2bFDQwE="
  },
  "seed_byts": "CtYCCiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAiLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMK2o/735hY/AFzqQARKNAQokYTY2ZWUwZjItYWI2OC00OWRmLWFlOTktYjMzZDc1MzdhODMxImUKJDYxN2MzOWU0LTRkNjktNDE5YS1iYmE2LWZiYWY5ZDM1YWZiMCiIJzIsQXNERTh2YVFFOEtxd0tQa3U4NEtxUWRDVzEtXzVtWm90OFY3X1hRYk5ZQWQ6DEluaXRpYWwgRm9ya0IgVruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgANKQeWmnP4lyXcZTTWuWblRS4rV6l6ZJTXgOv2buUwKKlUdbNimTk/q/XC3b0KMpeYmTQ5DMnrWaCuwIPfu5rc4snMBEiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAaC2luZGV4X2dyb3VwIixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSgBMkBhNGY3NGJjOGE5N2YzZjhlYmM1MTIyMjcxM2ZjZTBhYTk0ZDQ5OTRjOTIxNGYxN2JmZWU5YmQ2YWZjNTJkMmQyOiQ0YzBiZDVjNS0zNWI2LTQzYjQtOTJhNy1lMDY3YThlNzg2NWVCCWR1bW15X2FwcEog22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9JactSQdf/v/FaDKSDEqCSyenhrvgGlBHHL14Ps4pC+oxxiTHLK0xrKBQTHccgxJe9yVfLL0priANmCK09IFY03tmxQ0MB"
}

You can list the group just joined
curl -X GET -H 'Content-Type: application/json'  -d '{}'  http://127.0.0.1:8002/api/v1/groups
result:
{
  "groups": [
    {
      "group_id": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
      "group_name": "index_group",
      "owner_pubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
      "user_pubkey": "AkO8otfcqU5nYPyrvWLY3ypdglA5GXW-pYjYmTuJfOMU",
      "user_eth_addr": "0x78e348170C471F848B1A4cdC987a57e3046313e8",
      "consensus_type": "POA",
      "sync_type": "PRIVATE",
      "cipher_key": "a4f74bc8a97f3f8ebc51222713fce0aa94d4994c9214f17bfee9bd6afc52d2d2",
      "app_id": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
      "app_name": "dummy_app",
      "currt_top_block": 0,
      "last_updated": 1693425120202305215,
      "rex_syncer_status": "SYNCING",
      "rex_Syncer_result": null,
      "peers": null
    }
  ]
}

let's create your first post in this group
curl -X POST -H 'Content-Type: application/json'  -d '{"data":"xxxx"}'  http://127.0.0.1:8002/api/v1/group/617c39e4-4d69-419a-bba6-fbaf9d35afb0/content
result:
{
  "trx_id": "a3f32c29-acce-45e2-8510-3e0f5115b4a7"
}

check current group status
current_top_block increase to 1, As you can see, a new block is created to package your trx

curl -X GET -H 'Content-Type: application/json'  -d '{}'  http://127.0.0.1:8002/api/v1/groups
{
  "groups": [
    {
      "group_id": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
      "group_name": "index_group",
      "owner_pubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
      "user_pubkey": "AkO8otfcqU5nYPyrvWLY3ypdglA5GXW-pYjYmTuJfOMU",
      "user_eth_addr": "0x78e348170C471F848B1A4cdC987a57e3046313e8",
      "consensus_type": "POA",
      "sync_type": "PRIVATE",
      "cipher_key": "a4f74bc8a97f3f8ebc51222713fce0aa94d4994c9214f17bfee9bd6afc52d2d2",
      "app_id": "4c0bd5c5-35b6-43b4-92a7-e067a8e7865e",
      "app_name": "dummy_app",
      "currt_top_block": 1,     
      "last_updated": 1693516068931679416,
      "rex_syncer_status": "IDLE",
      "rex_Syncer_result": null,
      "peers": null
    }
  ]
}


you can check the block by using block_id

curl -X GET -H 'Content-Type: application/json'  -d '{}'  http://127.0.0.1:8002/api/v1/block/617c39e4-4d69-419a-bba6-fbaf9d35afb0/1

result:
{
  "block": {
    "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
    "BlockId": 1,
    "PrevHash": "VruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgAM=",
    "ProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
    "Trxs": [
      {
        "TrxId": "30bec1e2-d0ff-47a2-9381-0c21895307dc",
        "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
        "Data": "b6Fou5PQaUZkgrrkNsyCtM+HuXDk6Ix5YGZeYOwaTw==",
        "TimeStamp": "1693515861107843523",
        "Version": "2.1.0",
        "SenderPubkey": "AkO8otfcqU5nYPyrvWLY3ypdglA5GXW-pYjYmTuJfOMU",
        "Hash": "3OG+3Ffmr9ccgBf0+DD0Qay3P1QyP1QfpfO4HqIvd50=",
        "SenderSign": "ZXPg9ShGKFZjC363zzk3ph3geWVIPrmfLPlloc+LMnsqYwGe412ibcY/xiG1cnxqJ35SRkRe5LNKEzEn8WshAAE="
      }
    ],
    "TimeStamp": "1693515863897080926",
    "BlockHash": "Qr/1zUewJmOVRYY7BGVfe/ERSyPc5hHLycYbWqak42g=",
    "ProducerSign": "Q30IyiXblcjkzErnBWNHMMtEkVhUDM6R7Od1LW//UDdumVLjKGNPLpNuMh2fGMCjS7omgGNAjeVUBjqs5bUXkAA="
  },
  "status": "onchain"
}

you can check the trx by using trx_id
curl -X GET -H 'Content-Type: application/json'  -d '{}'  http://127.0.0.1:8002/api/v1/trx/617c39e4-4d69-419a-bba6-fbaf9d35afb0/30bec1e2-d0ff-47a2-9381-0c21895307dc
{
  "trx": {
    "TrxId": "30bec1e2-d0ff-47a2-9381-0c21895307dc",
    "GroupId": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
    "Data": "b6Fou5PQaUZkgrrkNsyCtM+HuXDk6Ix5YGZeYOwaTw==",
    "TimeStamp": "1693515861107843523",
    "Version": "2.1.0",
    "SenderPubkey": "AkO8otfcqU5nYPyrvWLY3ypdglA5GXW-pYjYmTuJfOMU",
    "Hash": "3OG+3Ffmr9ccgBf0+DD0Qay3P1QyP1QfpfO4HqIvd50=",
    "SenderSign": "ZXPg9ShGKFZjC363zzk3ph3geWVIPrmfLPlloc+LMnsqYwGe412ibcY/xiG1cnxqJ35SRkRe5LNKEzEn8WshAAE="
  },
  "status": "onchain"
}

group sync type
there 2 types of group sync types
"public sync type": 
  * any node who get the seed can join the group and request blocks in this group from other nodes
  * newly created block will be broadcast via group channel all online node will recive the new block

"private sync type": 
  * a node can join this group but it CAN NOT sync from other nodes, (sync request will be ignored by other nodes if the pubkey of requester is not in the syncer list)
  * newly created block will NOT be broadcast via group channel.
  * group owner can add/remove group sycners 

Let's start a user node "u1"
RUM_KSPASSWD=123 go run main.go rumlitenode --peername u1 --listen /ip4/127.0.0.1/tcp/7003 --apiport 8003 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm9w95mPtMLghqw6c2Zua7rX36zJAd7bMRonUvS7R9d4w2 --configdir config --datadir data --keystoredir u1keystore  --loglevel "debug"

u1 needs create a user sign pubkey with keyname "u1_test_app_sign_key"

u1 join the group just created
curl -X POST -H 'Content-Type: application/json' -d '{"seed":"CtYCCiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAiLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMK2o/735hY/AFzqQARKNAQokYTY2ZWUwZjItYWI2OC00OWRmLWFlOTktYjMzZDc1MzdhODMxImUKJDYxN2MzOWU0LTRkNjktNDE5YS1iYmE2LWZiYWY5ZDM1YWZiMCiIJzIsQXNERTh2YVFFOEtxd0tQa3U4NEtxUWRDVzEtXzVtWm90OFY3X1hRYk5ZQWQ6DEluaXRpYWwgRm9ya0IgVruDxry8tdHyKGjx6YTS3RDRrJ6o9jaX07f02UiRgANKQeWmnP4lyXcZTTWuWblRS4rV6l6ZJTXgOv2buUwKKlUdbNimTk/q/XC3b0KMpeYmTQ5DMnrWaCuwIPfu5rc4snMBEiQ2MTdjMzllNC00ZDY5LTQxOWEtYmJhNi1mYmFmOWQzNWFmYjAaC2luZGV4X2dyb3VwIixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSgBMkBhNGY3NGJjOGE5N2YzZjhlYmM1MTIyMjcxM2ZjZTBhYTk0ZDQ5OTRjOTIxNGYxN2JmZWU5YmQ2YWZjNTJkMmQyOiQ0YzBiZDVjNS0zNWI2LTQzYjQtOTJhNy1lMDY3YThlNzg2NWVCCWR1bW15X2FwcEog22Q2VX/VApu1HPdfeNHFtIOIA6wnvp2fsAxN+E9JactSQdf/v/FaDKSDEqCSyenhrvgGlBHHL14Ps4pC+oxxiTHLK0xrKBQTHccgxJe9yVfLL0priANmCK09IFY03tmxQ0MB", "user_sign_keyname":"u1_test_app_sign_key"}' http://127.0.0.1:8003/api/v2/group/joingroupbyseed

Since our group is "private sync" type, u1 can not get any block (exepct genesis block), top block id is always 0


Group Owner grant sync permisson for u1
curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"617c39e4-4d69-419a-bba6-fbaf9d35afb0", "syncer_pubkey":"AowfJhrIcD9H0X3-sHfANNB3hl8s3TQlHMj6eqFf2nwo", "action":"add", "memo":""}' http://127.0.0.1:8002/api/v2/group/updsyncer

result:
{
  "group_id": "617c39e4-4d69-419a-bba6-fbaf9d35afb0",
  "syncer_pubkey": "AowfJhrIcD9H0X3-sHfANNB3hl8s3TQlHMj6eqFf2nwo",
  "action": "ADD",
  "memo": "xxxx",
  "trx_id": "b3d49509-1525-47d0-b2a0-d746dd459720"
}

Now u1 can sync group blocks, verify by check top block id and trx content

------------------------------------------------------------------------------------------------------------------------------

Time to create a Cellar group
- Cellar group works as a service provider
- Cellar group has term of services

- When request service form a cellar group, node need execute cellar group contractr and provide proof (for example, payment recipt)
- Owner of cellar should verify the proof by themselves (RUM doesn't provide verify service)

A cellar group can provide 2 type of services
1. SYNC
  - after accept a SYNC request from a node, cellar group will check proof provided by the req, join the group and start sync all blocks till reach the block number listed in the request
  - SYNC request for the same group can be send multiple time, each time cellar receive the request, it will check proof, sync block till reach the block number in the request
  - when finish sync, an ARCHIVE type trx will be send to this group
  - STORAGE service requester should wait the ARCHIVE trx as a mark of sync with cellar finished then the group can be closed to save some local resources

2. BREW
  - after accept a BREW request from a node, cellar will 
    a. check proof
    b. join the provided group seed
    c. sync certain amount of blocks till reach the block number listed in the request
  - from that point, cellar will  work as the producer of this group (collect trxs and build block) and sign all blocks by using brewer key
  - when take over the group, an FORK type trx will be send to this group
  - BREW service requester should wait for the FORK trx till it can safely close the group locally

create a cellar group

curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"dummy_app_id", "app_name":"my_dummy_cellar", "group_name":"cellar_group","consensus_type":"poa", "sync_type":"private", "epoch_duration":1000, "owner_keyname":"my_test_app_owner_key", "producer_keyname":"my_test_app_producer_key", "cellar_"brew_service":{"term":"BREW FOR EVERYONE", "contract":""}, "sync_service":{"term":"SYNC FOR EVERYONE","contract":""} }' http://127.0.0.1:8002/api/v2/group/newseed | jq

parameters:
4 new parmeters are requested to create a new cellar group seed
  - BrewService   
	- SyncService 
	- BrewerKeyname 
	- SyncerKeyname 
  * all other parameters are as same as the parameters when create group seed
  * if brewer_keyname or "syncer_keyname" are not given, a new keyname (and keypair) will be created for brewer and syncer

 BrewService:
  - Term string : brew service term
  - Contract    : A PRS contract (executable or not) for brew service

 SyncService:
  - Term string : sync service term
  - Contract    : A PRS contract (executable or not) for sync service

  BrewerKeyname string : keyname of cellar group brewer, the pubkey will be use to sign all new blocks and FORK trx when brew service is accepted
  
  SyncerKeyname string : keyname of cellar group syncer, the pubkey will be use to sign ARCHIVE trx when sync service is accepted

  curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"dummy_app_id", "app_name":"my_dummy_cellar", "group_name":"cellar_group","consensus_type":"poa", "sync_type":"private", "epoch_duration":1000, "owner_keyname":"my_test_app_owner_key", "producer_keyname":"my_test_app_producer_key", "brew_service":{"term":"BREW FOR EVERYONE", "contract":""}, "sync_service":{"term":"SYNC FOR EVERYONE","contract":""}}' http://127.0.0.1:8002/api/v2/group/newseed | jq

result:
  {
    "group_id": "98fd8081-ed85-4806-9a60-b107a13a066d",
    "owner_keyname": "my_test_app_owner_key",
    "producer_sign_keyname": "98fd8081-ed85-4806-9a60-b107a13a066d_neoproducer_sign_keyname",
    "brewer_keyname": "98fd8081-ed85-4806-9a60-b107a13a066d_brewer_sign_keyname",
    "syncer_keyname": "98fd8081-ed85-4806-9a60-b107a13a066d_syncer_sign_keyname",
    "seed": {
      "GroupId": "98fd8081-ed85-4806-9a60-b107a13a066d",
      "GroupName": "cellar_group",
      "OwnerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
      "CipherKey": "e4d57865cd18223d3bed361a754e07bb2d05469aa5eaa925d41e55a2c6b923f4",
      "AppId": "dummy_app_id",
      "AppName": "my_dummy_cellar",
      "GenesisBlock": {
        "GroupId": "98fd8081-ed85-4806-9a60-b107a13a066d",
        "ProducerPubkey": "Az4MCHXOg3-jA-CWJ26lJpQlclKYwQ8aIUw2ZDObF6li",
        "TimeStamp": "1694449939617988775",
        "Consensus": {
          "Data": "CiRjMmVjM2ExMC0wNWQyLTRiZTYtODUzYi03NDMwOTFmZTQ4MmUiZQokOThmZDgwODEtZWQ4NS00ODA2LTlhNjAtYjEwN2ExM2EwNjZkKOgHMixBejRNQ0hYT2czLWpBLUNXSjI2bEpwUWxjbEtZd1E4YUlVdzJaRE9iRjZsaToMSW5pdGlhbCBGb3Jr"
        },
        "BlockHash": "Kw3Dnc1aIuUEdyeb+nDSAO+YacPFLmJxSUtzGBiZAHk=",
        "ProducerSign": "UnyZ2R1zyKondPWDDTYdzNPqzFNofYFz4Z1bydz/a/5FRTthN9dF7Le+ecL3S4Nqv4WHoQVJr3bDgGow4SypGQA="
      },
      "Services": [
        {
          "Service": "CixBeHZJRld1MlpkRFUwNGZKTGJTLXJmUkd5OXNkdTB4amhLQ1lhb2V5NmJKdRIsQWdudWQ5NmZGT0lfZ0MyaS1jQUU2LU1wOXhELXRqdHU5eklzOTZhbkhsckQaEUJSRVcgRk9SIEVWRVJZT05F"
        },
        {
          "Type": 1,
          "Service": "CixBZ251ZDk2ZkZPSV9nQzJpLWNBRTYtTXA5eEQtdGp0dTl6SXM5NmFuSGxyRBIRQlJFVyBGT1IgRVZFUllPTkU="
        }
      ],
      "Hash": "cUmhytj2uyAGBTZWxKR6QJdt/SMU0c5WXINWb4W9IGk=",
      "Signature": "OY6KOPi7H8eP0D4rM4Uz9LmQAjiBf7BE6YaSa+ndJrpL6lDZ7SSqMOlBNqjAzPLfHMpnKMSZeQf3CYL6wxEJDgA="
    },
    "seed_byts": "CiQ5OGZkODA4MS1lZDg1LTQ4MDYtOWE2MC1iMTA3YTEzYTA2NmQSDGNlbGxhcl9ncm91cBosQXE1ajkwN3hQel9xVjFzVEVRekIwUHhvazlENy12WENTSTlKR2JqVFowamUqQGU0ZDU3ODY1Y2QxODIyM2QzYmVkMzYxYTc1NGUwN2JiMmQwNTQ2OWFhNWVhYTkyNWQ0MWU1NWEyYzZiOTIzZjQyDGR1bW15X2FwcF9pZDoPbXlfZHVtbXlfY2VsbGFyQtYCCiQ5OGZkODA4MS1lZDg1LTQ4MDYtOWE2MC1iMTA3YTEzYTA2NmQiLEF6NE1DSFhPZzMtakEtQ1dKMjZsSnBRbGNsS1l3UThhSVV3MlpET2JGNmxpMKeB86Thp/nBFzqQARKNAQokYzJlYzNhMTAtMDVkMi00YmU2LTg1M2ItNzQzMDkxZmU0ODJlImUKJDk4ZmQ4MDgxLWVkODUtNDgwNi05YTYwLWIxMDdhMTNhMDY2ZCjoBzIsQXo0TUNIWE9nMy1qQS1DV0oyNmxKcFFsY2xLWXdROGFJVXcyWkRPYkY2bGk6DEluaXRpYWwgRm9ya0IgKw3Dnc1aIuUEdyeb+nDSAO+YacPFLmJxSUtzGBiZAHlKQVJ8mdkdc8iqJ3T1gw02HczT6sxTaH2Bc+GdW8nc/2v+RUU7YTfXRey3vnnC90uDar+Fh6EFSa92w4BqMOEsqRkASnESbwosQXh2SUZXdTJaZERVMDRmSkxiUy1yZlJHeTlzZHUweGpoS0NZYW9leTZiSnUSLEFnbnVkOTZmRk9JX2dDMmktY0FFNi1NcDl4RC10anR1OXpJczk2YW5IbHJEGhFCUkVXIEZPUiBFVkVSWU9ORUpFCAESQQosQWdudWQ5NmZGT0lfZ0MyaS1jQUU2LU1wOXhELXRqdHU5eklzOTZhbkhsckQSEUJSRVcgRk9SIEVWRVJZT05FUiBxSaHK2Pa7IAYFNlbEpHpAl239IxTRzlZcg1Zvhb0gaVpBOY6KOPi7H8eP0D4rM4Uz9LmQAjiBf7BE6YaSa+ndJrpL6lDZ7SSqMOlBNqjAzPLfHMpnKMSZeQf3CYL6wxEJDgA="
  }

Parameters:
  brewer_keyname : keyname of brewer
  syncer_keyname : keyname of syncer
  * all other parameters are as same as group seed


You can get the "plane text" version of a seed by using the following api

  curl -X POST -H 'Content-Type: application/json' -d '{"seed":"CiQ1ZWNmMWU5YS04YzI1LTQyMzktYmU1ZC1jMzM3NWE2YzQxMDMSDGNlbGxhcl9ncm91cBosQXE1ajkwN3hQel9xVjFzVEVRekIwUHhvazlENy12WENTSTlKR2JqVFowamUqQDVjOTQzYmU3MmZkZDlkODk2NTJlMTg0ZTdiN2RhZWYxMTNkNGEwOTM4Y2U5ZmE5OWJiMTU1MzI2M2Q0MTk0M2QyDGR1bW15X2FwcF9pZDoPbXlfZHVtbXlfY2VsbGFyQtYCCiQ1ZWNmMWU5YS04YzI1LTQyMzktYmU1ZC1jMzM3NWE2YzQxMDMiLEF1NHkwZ0FEUzFTN0Y2NmtCNXJxY1R3SU1LSHNYMmt5eGlJNmlVc20zeVpjMOa7yPewlPvBFzqQARKNAQokYWE0ZjUxNDQtNGNjZC00NTk2LWFjNmEtNjVmYWY4MTc4ZTQyImUKJDVlY2YxZTlhLThjMjUtNDIzOS1iZTVkLWMzMzc1YTZjNDEwMyjoBzIsQXU0eTBnQURTMVM3RjY2a0I1cnFjVHdJTUtIc1gya3l4aUk2aVVzbTN5WmM6DEluaXRpYWwgRm9ya0IgeVvC9zRV0jLldWdRhQaf8t0Adzmi4Ux+1HxjtVJjR+RKQZU+uN7AxjPu5cGuy7zSGks+PgbMq+dxUUMZdeqSOCFtLX5gX983bKywytktfVWyvdHtON5KqiTDcgS/x7RWmWIASnESbwosQXRTUVd2OWE0VDhMTlpIUG1TTjZlN2NUUXlycUhYSXlXb1B4N1BjbHdUV3ISLEF1RXlmMVBnQjBTZUJobUY1cWRKazROa1kycmdmNEZIWF9GTjVKWWxuNGJBGhFCUkVXIEZPUiBFVkVSWU9ORUpFCAESQQosQXVFeWYxUGdCMFNlQmhtRjVxZEprNE5rWTJyZ2Y0RkhYX0ZONUpZbG40YkESEUJSRVcgRk9SIEVWRVJZT05FUiAhy0ugVRfFEQOE/tt2ANEW+wgbfZS4L6CGZiznmsyhhFpBhRPQSZv4Rr6E9Les2yzUfcV3y8cMBxBQ0Zf9qt0NG1cfOFqYBOaigJ2H8NHZeDaKK++zxZWg+fThywjUAiKMwwE="}' http://127.0.0.1:8002/api/v2/group/parseseed

  result
  {
    "groupId": "5ecf1e9a-8c25-4239-be5d-c3375a6c4103",
    "groupName": "cellar_group",
    "ownerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
    "producerPubkey": "Au4y0gADS1S7F66kB5rqcTwIMKHsX2kyxiI6iUsm3yZc",
    "syncType": "PUBLIC",
    "cipherKey": "5c943be72fdd9d89652e184e7b7daef113d4a0938ce9fa99bb1553263d41943d",
    "appId": "dummy_app_id",
    "appName": "my_dummy_cellar",
    "consensusInfo": {
      "ConsensusId": "aa4f5144-4ccd-4596-ac6a-65faf8178e42",
      "ForkInfo": {
        "GroupId": "5ecf1e9a-8c25-4239-be5d-c3375a6c4103",
        "EpochDuration": 1000,
        "producers": [
          "Au4y0gADS1S7F66kB5rqcTwIMKHsX2kyxiI6iUsm3yZc"
        ],
        "Memo": "Initial Fork"
      }
    },
    "brewService": {
      "BrewerPubkey": "AtSQWv9a4T8LNZHPmSN6e7cTQyrqHXIyWoPx7PclwTWr",
      "SyncerPubkey": "AuEyf1PgB0SeBhmF5qdJk4NkY2rgf4FHX_FN5JYln4bA",
      "Term": "BREW FOR EVERYONE"
    },
    "syncService": {
      "SyncerPubkey": "AuEyf1PgB0SeBhmF5qdJk4NkY2rgf4FHX_FN5JYln4bA",
      "Term": "BREW FOR EVERYONE"
    },
    "genesisBlock": {
      "GroupId": "5ecf1e9a-8c25-4239-be5d-c3375a6c4103",
      "ProducerPubkey": "Au4y0gADS1S7F66kB5rqcTwIMKHsX2kyxiI6iUsm3yZc",
      "TimeStamp": "1694458069896011238",
      "Consensus": {
        "Data": "CiRhYTRmNTE0NC00Y2NkLTQ1OTYtYWM2YS02NWZhZjgxNzhlNDIiZQokNWVjZjFlOWEtOGMyNS00MjM5LWJlNWQtYzMzNzVhNmM0MTAzKOgHMixBdTR5MGdBRFMxUzdGNjZrQjVycWNUd0lNS0hzWDJreXhpSTZpVXNtM3laYzoMSW5pdGlhbCBGb3Jr"
      },
      "BlockHash": "eVvC9zRV0jLldWdRhQaf8t0Adzmi4Ux+1HxjtVJjR+Q=",
      "ProducerSign": "lT643sDGM+7lwa7LvNIaSz4+Bsyr53FRQxl16pI4IW0tfmBf3zdsrLDK2S19VbK90e043kqqJMNyBL/HtFaZYgA="
    },
    "hash": "IctLoFUXxREDhP7bdgDRFvsIG32UuC+ghmYs55rMoYQ=",
    "sign": "hRPQSZv4Rr6E9Les2yzUfcV3y8cMBxBQ0Zf9qt0NG1cfOFqYBOaigJ2H8NHZeDaKK++zxZWg+fThywjUAiKMwwE="
  }

You can verify the hash and signature of a seed by using the following api
  curl -X POST -H 'Content-Type: application/json' -d '{"seed":"CiQ1ZWNmMWU5YS04YzI1LTQyMzktYmU1ZC1jMzM3NWE2YzQxMDMSDGNlbGxhcl9ncm91cBosQXE1ajkwN3hQel9xVjFzVEVRekIwUHhvazlENy12WENTSTlKR2JqVFowamUqQDVjOTQzYmU3MmZkZDlkODk2NTJlMTg0ZTdiN2RhZWYxMTNkNGEwOTM4Y2U5ZmE5OWJiMTU1MzI2M2Q0MTk0M2QyDGR1bW15X2FwcF9pZDoPbXlfZHVtbXlfY2VsbGFyQtYCCiQ1ZWNmMWU5YS04YzI1LTQyMzktYmU1ZC1jMzM3NWE2YzQxMDMiLEF1NHkwZ0FEUzFTN0Y2NmtCNXJxY1R3SU1LSHNYMmt5eGlJNmlVc20zeVpjMOa7yPewlPvBFzqQARKNAQokYWE0ZjUxNDQtNGNjZC00NTk2LWFjNmEtNjVmYWY4MTc4ZTQyImUKJDVlY2YxZTlhLThjMjUtNDIzOS1iZTVkLWMzMzc1YTZjNDEwMyjoBzIsQXU0eTBnQURTMVM3RjY2a0I1cnFjVHdJTUtIc1gya3l4aUk2aVVzbTN5WmM6DEluaXRpYWwgRm9ya0IgeVvC9zRV0jLldWdRhQaf8t0Adzmi4Ux+1HxjtVJjR+RKQZU+uN7AxjPu5cGuy7zSGks+PgbMq+dxUUMZdeqSOCFtLX5gX983bKywytktfVWyvdHtON5KqiTDcgS/x7RWmWIASnESbwosQXRTUVd2OWE0VDhMTlpIUG1TTjZlN2NUUXlycUhYSXlXb1B4N1BjbHdUV3ISLEF1RXlmMVBnQjBTZUJobUY1cWRKazROa1kycmdmNEZIWF9GTjVKWWxuNGJBGhFCUkVXIEZPUiBFVkVSWU9ORUpFCAESQQosQXVFeWYxUGdCMFNlQmhtRjVxZEprNE5rWTJyZ2Y0RkhYX0ZONUpZbG40YkESEUJSRVcgRk9SIEVWRVJZT05FUiAhy0ugVRfFEQOE/tt2ANEW+wgbfZS4L6CGZiznmsyhhFpBhRPQSZv4Rr6E9Les2yzUfcV3y8cMBxBQ0Zf9qt0NG1cfOFqYBOaigJ2H8NHZeDaKK++zxZWg+fThywjUAiKMwwE="}' http://127.0.0.1:8002/api/v2/group/verifyseed

result:
  {
    "verified": true,
    "error": ""
  }



start bootstramp
  RUM_KSPASSWD=123 go run main.go bootstrapnode --listen /ip4/0.0.0.0/tcp/10666 --loglevel "debug"
    get id
    --- > bootstrap host created, ID:<16Uiu2HAm9w95mPtMLghqw6c2Zua7rX36zJAd7bMRonUvS7R9d4w2>,

start o1
  RUM_KSPASSWD=123 go run main.go rumlitenode --peername o1 --listen /ip4/127.0.0.1/tcp/7002 --apiport 8002 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm9w95mPtMLghqw6c2Zua7rX36zJAd7bMRonUvS7R9d4w2 --configdir config --datadir data --keystoredir o1keystore  --loglevel "debug"

start u1
  RUM_KSPASSWD=123 go run main.go rumlitenode --peername u1 --listen /ip4/127.0.0.1/tcp/7003 --apiport 8003 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm9w95mPtMLghqw6c2Zua7rX36zJAd7bMRonUvS7R9d4w2 --configdir config --datadir data --keystoredir u1keystore  --loglevel "debug"

o1 create keys for cellar group
  cellar owner :
    curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"o1_cellar_owner"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey

    {
      "key_alias": "91e8542f-237a-4dee-bc0c-7252c5177b6d",
      "key_name": "o1_cellar_owner",
      "pubkey": "AifJ9hx_BnZEadTTbPv_lUEbhIQ0myf9xOWFG_vNxBaR"
    }

  cellar producer
    curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"o1_cellar_producer"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey
    
    {
      "key_alias": "defeb806-d87a-4cec-8b6c-1f7cccf14955",
      "key_name": "o1_cellar_trx_sign",
      "pubkey": "AvbcnXkcfHdzr7cHtRsWuMwLL-1vDNpxDLhcQxjd1Acr"
    }

  cellar trx sign (use to sign trx in cellar group)

    curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"o1_cellar_trx_sign"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey

    {
      "key_alias": "defeb806-d87a-4cec-8b6c-1f7cccf14955",
      "key_name": "o1_cellar_trx_sign",
      "pubkey": "AvbcnXkcfHdzr7cHtRsWuMwLL-1vDNpxDLhcQxjd1Acr"
    }

  cellar brewer
    curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"o1_cellar_brewer"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey
    {
      "key_alias": "995799d1-115f-40d1-b6d0-40369e8b45c8",
      "key_name": "o1_cellar_brewer",
      "pubkey": "A_ZbfSDtLwjIzIAfGTfWaBCdg_wip0AP_1vmzkXUdkcA"
    }


  cellar syncer
    curl -X POST -H 'Content-Type: application/json' -d '{"key_name":"o1_cellar_syncer"}'  http://127.0.0.1:8002/api/v2/keystore/createsignkey
    {
      "key_alias": "1100ef72-5935-45f4-85ec-0adc566ed337",
      "key_name": "o1_cellar_syncer",
      "pubkey": "AlOtlrtI2r2HHVi-rDYv2XUlsBhQYRbrrY664WrkW3uT"
    }

  make sure all keys are created, list all keys
    curl -X GET -H 'Content-Type: application/json'  -d '{}' http://127.0.0.1:8002/api/v2/keystore/getallkeys

    {
    "keys_list": [
      {
        "pubkey": "AtA-xhMRKjTg2vf3UaNEGVje_Av13RLACksQSjvy6aF8",
        "key_name": "db646a74-3e10-4a73-9371-5682ef5c0a21_neoproducer_sign_keyname",
        "alias": []
      },
      {
        "pubkey": "A5FayGbpAPzPn4qwglQHqu20cLbb8QeMjoJfqa2pNTTt",
        "key_name": "default",
        "alias": []
      },
      {
        "pubkey": "A_ZbfSDtLwjIzIAfGTfWaBCdg_wip0AP_1vmzkXUdkcA",
        "key_name": "o1_cellar_brewer",
        "alias": [
          "995799d1-115f-40d1-b6d0-40369e8b45c8"
        ]
      },
      {
        "pubkey": "AifJ9hx_BnZEadTTbPv_lUEbhIQ0myf9xOWFG_vNxBaR",
        "key_name": "o1_cellar_owner",
        "alias": [
          "91e8542f-237a-4dee-bc0c-7252c5177b6d"
        ]
      },
      {
        "pubkey": "A2fLpK0H83X2ot0MrdjLuDHir5G2LPPZmWqdWEc_rNSI",
        "key_name": "o1_cellar_producer",
        "alias": [
          "d298ea65-5430-4227-a01d-370dc65dc6d4"
        ]
      },
      {
        "pubkey": "AlOtlrtI2r2HHVi-rDYv2XUlsBhQYRbrrY664WrkW3uT",
        "key_name": "o1_cellar_syncer",
        "alias": [
          "1100ef72-5935-45f4-85ec-0adc566ed337"
        ]
      },
      {
        "pubkey": "AvbcnXkcfHdzr7cHtRsWuMwLL-1vDNpxDLhcQxjd1Acr",
        "key_name": "o1_cellar_trx_sign",
        "alias": [
          "defeb806-d87a-4cec-8b6c-1f7cccf14955"
        ]
      }
    ]
  }

create cellar group seed

  curl -X POST -H 'Content-Type: application/json' -d '{"app_id":"o1_cellar_appid", "app_name":"o1_cellar", "group_name":"o1_cellar_group","consen
  sus_type":"poa", "sync_type":"private", "epoch_duration":1000, "owner_keyname":"o1_cellar_owner", "producer_keyname":"o1_cellar_producer", "brew_service":{"term":"BREW FOR EVERYONE", "contract":""}, "sync_service":{"term":"SYNC FOR EVERYONE","contract":""}, "brewer_keyname":"o1_cellar_brewer", "syncer_keyname":"o1_cellar_syncer"}' http://127.0.0.1:8002/api/v2/group/newseed | jq

  {
    "group_id": "5bf9db41-631c-4818-9a54-f85c1a503f84",
    "owner_keyname": "o1_cellar_owner",
    "producer_sign_keyname": "o1_cellar_producer",
    "brewer_keyname": "o1_cellar_brewer",
    "syncer_keyname": "o1_cellar_syncer",
    "seed_byts": "CiQ1YmY5ZGI0MS02MzFjLTQ4MTgtOWE1NC1mODVjMWE1MDNmODQSD28xX2NlbGxhcl9ncm91cBosQWlmSjloeF9CblpFYWRUVGJQdl9sVUViaElRMG15Zjl4T1dGR192TnhCYVIgASpANTA4ZmQ0NTI5NmFmZmE1Mjc3ZTViODAxZDEyYThjOWM5ZDMwMDAwYjdlMzc3M2IxZWE2ZThiY2JkYTc2OGEzYTIPbzFfY2VsbGFyX2FwcGlkOglvMV9jZWxsYXJC1gIKJDViZjlkYjQxLTYzMWMtNDgxOC05YTU0LWY4NWMxYTUwM2Y4NCIsQTJmTHBLMEg4M1gyb3QwTXJkakx1REhpcjVHMkxQUFptV3FkV0VjX3JOU0kwhLPDxZ6WkMIXOpABEo0BCiRiNjFlOGU1Zi0yNGZjLTQ1MjctODc4OS1jNjk1MGEwMTFhMzciZQokNWJmOWRiNDEtNjMxYy00ODE4LTlhNTQtZjg1YzFhNTAzZjg0KOgHMixBMmZMcEswSDgzWDJvdDBNcmRqTHVESGlyNUcyTFBQWm1XcWRXRWNfck5TSToMSW5pdGlhbCBGb3JrQiAczmWLUjOPK9Ditm2EytWFRY0f1kZks7wg/mYvULMn4EpBZbfMHkdDF5w5Ec0snG3Qv9zZYkeLxrpZT/5OX8inOzlHsfKgnaC6hvjbpKgo9Q2RBhOq8xc8drtUkpukfrWULQFKcRJvCixBX1piZlNEdEx3akl6SUFmR1RmV2FCQ2RnX3dpcDBBUF8xdm16a1hVZGtjQRIsQWxPdGxydEkycjJISFZpLXJEWXYyWFVsc0JoUVlSYnJyWTY2NFdya1czdVQaEUJSRVcgRk9SIEVWRVJZT05FSkUIARJBCixBbE90bHJ0STJyMkhIVmktckRZdjJYVWxzQmhRWVJicnJZNjY0V3JrVzN1VBIRQlJFVyBGT1IgRVZFUllPTkVSICjnwqAMA4JEwaqhPtE3m6y0AFbrzrMDAKfOau/WQsNKWkFaKu1fP3I0Lyg1QnGhzKEcuG30NU9Azb3XyKVOOtCt1EYcHOQUKJDlaudxHGat0VgvZlG/JMNv0O7JqKiajogLAQ=="
  }

verify seed is validhistory

  curl -X POST -H 'Content-Type: application/json' -d '{"seed":"CiQ1YmY5ZGI0MS02MzFjLTQ4MTgtOWE1NC1mODVjMWE1MDNmODQSD28xX2NlbGxhcl9ncm91cBosQWlmSjloeF9CblpFYWRUVGJQdl9sVUViaElRMG15Zjl4T1dGR192TnhCYVIgASpANTA4ZmQ0NTI5NmFmZmE1Mjc3ZTViODAxZDEyYThjOWM5ZDMwMDAwYjdlMzc3M2IxZWE2ZThiY2JkYTc2OGEzYTIPbzFfY2VsbGFyX2FwcGlkOglvMV9jZWxsYXJC1gIKJDViZjlkYjQxLTYzMWMtNDgxOC05YTU0LWY4NWMxYTUwM2Y4NCIsQTJmTHBLMEg4M1gyb3QwTXJkakx1REhpcjVHMkxQUFptV3FkV0VjX3JOU0kwhLPDxZ6WkMIXOpABEo0BCiRiNjFlOGU1Zi0yNGZjLTQ1MjctODc4OS1jNjk1MGEwMTFhMzciZQokNWJmOWRiNDEtNjMxYy00ODE4LTlhNTQtZjg1YzFhNTAzZjg0KOgHMixBMmZMcEswSDgzWDJvdDBNcmRqTHVESGlyNUcyTFBQWm1XcWRXRWNfck5TSToMSW5pdGlhbCBGb3JrQiAczmWLUjOPK9Ditm2EytWFRY0f1kZks7wg/mYvULMn4EpBZbfMHkdDF5w5Ec0snG3Qv9zZYkeLxrpZT/5OX8inOzlHsfKgnaC6hvjbpKgo9Q2RBhOq8xc8drtUkpukfrWULQFKcRJvCixBX1piZlNEdEx3akl6SUFmR1RmV2FCQ2RnX3dpcDBBUF8xdm16a1hVZGtjQRIsQWxPdGxydEkycjJISFZpLXJEWXYyWFVsc0JoUVlSYnJyWTY2NFdya1czdVQaEUJSRVcgRk9SIEVWRVJZT05FSkUIARJBCixBbE90bHJ0STJyMkhIVmktckRZdjJYVWxzQmhRWVJicnJZNjY0V3JrVzN1VBIRQlJFVyBGT1IgRVZFUllPTkVSICjnwqAMA4JEwaqhPtE3m6y0AFbrzrMDAKfOau/WQsNKWkFaKu1fP3I0Lyg1QnGhzKEcuG30NU9Azb3XyKVOOtCt1EYcHOQUKJDlaudxHGat0VgvZlG/JMNv0O7JqKiajogLAQ=="}' http://127.0.0.1:8002/api/v2/group/verifyseed

  {
    "verified": true,
    "error": ""
  }

parse seed to check the details

  curl -X POST -H 'Content-Type: application/json' -d '{"seed":"CiQ1YmY5ZGI0MS02MzFjLTQ4MTgtOWE1NC1mODVjMWE1MDNmODQSD28xX2NlbGxhcl9ncm91cBosQWlmSjloeF9CblpFYWRUVGJQdl9sVUViaElRMG15Zjl4T1dGR192TnhCYVIgASpANTA4ZmQ0NTI5NmFmZmE1Mjc3ZTViODAxZDEyYThjOWM5ZDMwMDAwYjdlMzc3M2IxZWE2ZThiY2JkYTc2OGEzYTIPbzFfY2VsbGFyX2FwcGlkOglvMV9jZWxsYXJC1gIKJDViZjlkYjQxLTYzMWMtNDgxOC05YTU0LWY4NWMxYTUwM2Y4NCIsQTJmTHBLMEg4M1gyb3QwTXJkakx1REhpcjVHMkxQUFptV3FkV0VjX3JOU0kwhLPDxZ6WkMIXOpABEo0BCiRiNjFlOGU1Zi0yNGZjLTQ1MjctODc4OS1jNjk1MGEwMTFhMzciZQokNWJmOWRiNDEtNjMxYy00ODE4LTlhNTQtZjg1YzFhNTAzZjg0KOgHMixBMmZMcEswSDgzWDJvdDBNcmRqTHVESGlyNUcyTFBQWm1XcWRXRWNfck5TSToMSW5pdGlhbCBGb3JrQiAczmWLUjOPK9Ditm2EytWFRY0f1kZks7wg/mYvULMn4EpBZbfMHkdDF5w5Ec0snG3Qv9zZYkeLxrpZT/5OX8inOzlHsfKgnaC6hvjbpKgo9Q2RBhOq8xc8drtUkpukfrWULQFKcRJvCixBX1piZlNEdEx3akl6SUFmR1RmV2FCQ2RnX3dpcDBBUF8xdm16a1hVZGtjQRIsQWxPdGxydEkycjJISFZpLXJEWXYyWFVsc0JoUVlSYnJyWTY2NFdya1czdVQaEUJSRVcgRk9SIEVWRVJZT05FSkUIARJBCixBbE90bHJ0STJyMkhIVmktckRZdjJYVWxzQmhRWVJicnJZNjY0V3JrVzN1VBIRQlJFVyBGT1IgRVZFUllPTkVSICjnwqAMA4JEwaqhPtE3m6y0AFbrzrMDAKfOau/WQsNKWkFaKu1fP3I0Lyg1QnGhzKEcuG30NU9Azb3XyKVOOtCt1EYcHOQUKJDlaudxHGat0VgvZlG/JMNv0O7JqKiajogLAQ=="}' http://127.0.0.1:8002/api/v2/group/parseseed

  {
    "groupId": "5bf9db41-631c-4818-9a54-f85c1a503f84",
    "groupName": "o1_cellar_group",
    "ownerPubkey": "AifJ9hx_BnZEadTTbPv_lUEbhIQ0myf9xOWFG_vNxBaR",
    "producerPubkey": "A2fLpK0H83X2ot0MrdjLuDHir5G2LPPZmWqdWEc_rNSI",
    "syncType": "PRIVATE",
    "cipherKey": "508fd45296affa5277e5b801d12a8c9c9d30000b7e3773b1ea6e8bcbda768a3a",
    "appId": "o1_cellar_appid",
    "appName": "o1_cellar",
    "consensusInfo": {
      "ConsensusId": "b61e8e5f-24fc-4527-8789-c6950a011a37",
      "ForkInfo": {
        "GroupId": "5bf9db41-631c-4818-9a54-f85c1a503f84",
        "EpochDuration": 1000,
        "producers": [
          "A2fLpK0H83X2ot0MrdjLuDHir5G2LPPZmWqdWEc_rNSI"
        ],
        "Memo": "Initial Fork"
      }
    },
    "brewService": {
      "BrewerPubkey": "A_ZbfSDtLwjIzIAfGTfWaBCdg_wip0AP_1vmzkXUdkcA",
      "SyncerPubkey": "AlOtlrtI2r2HHVi-rDYv2XUlsBhQYRbrrY664WrkW3uT",
      "Term": "BREW FOR EVERYONE"
    },
    "syncService": {
      "SyncerPubkey": "AlOtlrtI2r2HHVi-rDYv2XUlsBhQYRbrrY664WrkW3uT",
      "Term": "BREW FOR EVERYONE"
    },
    "genesisBlock": {
      "GroupId": "5bf9db41-631c-4818-9a54-f85c1a503f84",
      "ProducerPubkey": "A2fLpK0H83X2ot0MrdjLuDHir5G2LPPZmWqdWEc_rNSI",
      "TimeStamp": "1694550492655442308",
      "Consensus": {
        "Data": "CiRiNjFlOGU1Zi0yNGZjLTQ1MjctODc4OS1jNjk1MGEwMTFhMzciZQokNWJmOWRiNDEtNjMxYy00ODE4LTlhNTQtZjg1YzFhNTAzZjg0KOgHMixBMmZMcEswSDgzWDJvdDBNcmRqTHVESGlyNUcyTFBQWm1XcWRXRWNfck5TSToMSW5pdGlhbCBGb3Jr"
      },
      "BlockHash": "HM5li1IzjyvQ4rZthMrVhUWNH9ZGZLO8IP5mL1CzJ+A=",
      "ProducerSign": "ZbfMHkdDF5w5Ec0snG3Qv9zZYkeLxrpZT/5OX8inOzlHsfKgnaC6hvjbpKgo9Q2RBhOq8xc8drtUkpukfrWULQE="
    },
    "hash": "KOfCoAwDgkTBqqE+0TebrLQAVuvOswMAp85q79ZCw0o=",
    "sign": "WirtXz9yNC8oNUJxocyhHLht9DVPQM2918ilTjrQrdRGHBzkFCiQ5WrncRxmrdFYL2ZRvyTDb9Duyaiomo6ICwE="
  }

o1 join the new cellar group











===== TO BE MODIFIED =====
酒窖（cellar）
酒窖其实也是一个group，同步类型可以是public或者private，producer可以是一个或者多个（一旦确定则不可更改，除非停机fork）
酒窖提供2种服务
  - Storage, 只同注册的组的block
  - Brew, 提供producer签名服务
酒窖会根据放入其中的Seed的服务类型提供同步或者签名服务
酒窖中的所有组会保持打开状态，以随时给不同业务提供block同步或者出块服务
一个酒窖本身不能放入其他酒窖
一个酒窖可以同意其他酒窖加入自己并同步酒窖group本身的block，加入的酒窖也同样需要给酒窖里的seed提供服务

============================================================================================================================

节点，酒窖和种子的互动过程
1. 节点A创建了一个group seed Group_A
2. 节点A在本地调用JoinGroupBySeed加入 Group_A
3. 节点A将一些内容切片，并以POST的形式存入 Group_A
4. 节点A获取酒窖B的种子
5. 节点A加入酒窖B（但是不要同步酒窖中的Block）
6. 节点A向酒窖B发送类型为SYNC_REQ的trx，包括如下内容
  a. Group_A的种子
  b. 请求同步的块数
  c. 支付凭证（或钱包地址）
7. 酒窖B在接到这个请求后
  a. 检查支付凭证
  b. 加入这个组
8. 酒窖B开始同步 Group_A
  a. 酒窖B在Group_A 发送trx，SYNC_RESP（类型为 START)，以标识开始同步
9. 酒窖B在同步Group_A的过程中，每隔一段会写入一个SYNC_RESP trx（类型为PROGRESS)，以标识同步的进度
9. 当酒窖B完成同步 Group_A时，写入SYNC_RESP
  a.如果SYNC_REQ里标明了块数，则当完成指定block_id时，写入DONE
  b.如果为连续同步，则当同步到ON_TOP时，写入ON_TOP
10. 节点A在完成同步后，可以按照app要求处理本地的Group_A，例如close Group_A以节省资源
11. 酒窖B中的Group_A保持打开，为其他人提供Block


节点可能提供的酒窖API
	- 创建一个酒窖（公开/私有）
	- 删除一个酒窖
	- 列出所有酒窖
	- 列出某个酒窖的所有组
	- 列出某个酒窖的所有申请
	- 批准/拒绝某个种子的加入申请
	- 列出一个酒窖里所有group的状态