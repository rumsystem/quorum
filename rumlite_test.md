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

Time to create a Cella
- Cella has a seed
- Cella works as a service provider
- Cella has term of services
- When request service form a cella, node need fulfill the term of service and provide proof (for example, payment recipt)
- Owner of cella should verify the proof by themselves (RUM doesn't provide verify service)

A cella can provide 2 type of services
1. STORAGE
  - after accept a STORAGE request from a node, cella will check proof, join the group and start sync all blocks till reach the block number listed in the request
  - Storage request for the same group can be send multiple time, each time cella receive the request, it will check proof, sync block till reach the block number in the request
  - when finish sync, an ARCHIVE type trx will be send to this group
  - STORAGE service requester should wait the ARCHIVE trx as a mark of sync with cella finished then the group can be closed to save some local resources

2. BREW
  - after accept a BREW request from a node, cella will 
    a. check proof
    b. join the provided group seed
    c. sync certain mount of blocks till reach the block number listed in the request
  - from that point, cella will  work as the producer of this group (collect trxs and build block) and sign all blocks by using brewer key
  - when take over the group, an FORK type trx will be send to this group
  - BREW service requester should wait for the ARCHIVE trx


create a cella seed
curl -X POST -H 'Content-Type: application/json' -d '{"cella_name":"dummy_cella", "epoch_duration":1000, "owner_keyname":"my_test_app_owner_key", "producer_keyname":"my_test_app_producer_key", "brewer_keyname":"my_brewer_key", "brew_service_term":"BREW FOR EVERYONE", "storage_service_term":"STORAGE FOR EVERYONE"}' http://127.0.0.1:8002/api/v2/cella/newseed | jq

parameters:
cella_name: name of the cella
brewer_keyname : brewer keyname
brew_service_term : term of brew service
storage_service_term : term of storage service

* all other parameters are as same as the parameters when create group seed

result:
{
  "cella_id": "bc42ba3a-8972-4af5-b3a9-21b03e719280",
  "owner_keyname": "my_test_app_owner_key",
  "brewer_keyname": "bc42ba3a-8972-4af5-b3a9-21b03e719280_brewer_sign_keyname",
  "producer_keyname": "my_test_app_producer_key",
  "seed": {
    "CellaId": "bc42ba3a-8972-4af5-b3a9-21b03e719280",
    "CellaName": "dummy_cella",
    "CellaOwnerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
    "CellaBrewerPubkey": "A8uxWZPMrH216FZhVOjQj7kZcnfaVtyUoJGNjE0tFfYd",
    "CellaProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
    "ServiceTerms": [
      {
        "Type": 1,
        "Term": "ChRTVE9SQUdFIEZPUiBFVkVSWU9ORQ=="
      },
      {
        "Term": "ChFCUkVXIEZPUiBFVkVSWU9ORQ=="
      }
    ],
    "Group": {
      "GenesisBlock": {
        "GroupId": "30dcc230-e468-4825-82dc-56e06199d5d9",
        "ProducerPubkey": "AsDE8vaQE8KqwKPku84KqQdCW1-_5mZot8V7_XQbNYAd",
        "TimeStamp": "1694026288131981653",
        "Consensus": {
          "Data": "CiRkNGMzNTJiZi02ZmE1LTRlNmUtOGQzOC1iMDIyMTVlNmY5ZWMiZQokMzBkY2MyMzAtZTQ2OC00ODI1LTgyZGMtNTZlMDYxOTlkNWQ5KOgHMixBc0RFOHZhUUU4S3F3S1BrdTg0S3FRZENXMS1fNW1ab3Q4VjdfWFFiTllBZDoMSW5pdGlhbCBGb3Jr"
        },
        "BlockHash": "cXLFqIbp270vmPAFnGNzt5Tlso9W00+fKsxSxsXuXYk=",
        "ProducerSign": "SAO4J3WL9W2JYwdpkHdggmLg0TuW3s/rIoJYpGtYqzMAH4jFP6NIQmqYhuOksmYpeCZLDhBKW51rMIsLyhLDXQE="
      },
      "GroupId": "30dcc230-e468-4825-82dc-56e06199d5d9",
      "GroupName": "dummy_cella_group",
      "OwnerPubkey": "Aq5j907xPz_qV1sTEQzB0Pxok9D7-vXCSI9JGbjTZ0je",
      "SyncType": 1,
      "CipherKey": "333e594f5afb800b910345af216f38106af02e686a220c205b72deb94b9f9da5",
      "Hash": "vMqI5l+Cheh9+U22klpj4Oy499kV33gQC6QvArlmxYQ=",
      "Signature": "/wwXm8qbsdEzLxNiBaKSXY95apf+hNd3x56IxiYGM0p709YFonLlqAMF2huaP6ofy3O+b7rPmhxt7AIkqw6EtAE="
    },
    "Hash": "tMOZqdMdVGz/SltXUwlrTH/nV76dLfZrXDY6U/k610Q=",
    "Signature": "6B+uzAZZTHWofGwsE+pV9umeSf+INsuAdkF+rOkTjL9JBLPsSdO/feIO60DJTL0xFkUW6CAIJSQ/pAN+FXjFvQA="
  },
  "seed_byts": "CiRiYzQyYmEzYS04OTcyLTRhZjUtYjNhOS0yMWIwM2U3MTkyODASC2R1bW15X2NlbGxhGixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSIsQTh1eFdaUE1ySDIxNkZaaFZPalFqN2taY25mYVZ0eVVvSkdOakUwdEZmWWQqLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMhoIARIWChRTVE9SQUdFIEZPUiBFVkVSWU9ORTIVEhMKEUJSRVcgRk9SIEVWRVJZT05FOukECtYCCiQzMGRjYzIzMC1lNDY4LTQ4MjUtODJkYy01NmUwNjE5OWQ1ZDkiLEFzREU4dmFRRThLcXdLUGt1ODRLcVFkQ1cxLV81bVpvdDhWN19YUWJOWUFkMNWCn8Lw/ZjBFzqQARKNAQokZDRjMzUyYmYtNmZhNS00ZTZlLThkMzgtYjAyMjE1ZTZmOWVjImUKJDMwZGNjMjMwLWU0NjgtNDgyNS04MmRjLTU2ZTA2MTk5ZDVkOSjoBzIsQXNERTh2YVFFOEtxd0tQa3U4NEtxUWRDVzEtXzVtWm90OFY3X1hRYk5ZQWQ6DEluaXRpYWwgRm9ya0IgcXLFqIbp270vmPAFnGNzt5Tlso9W00+fKsxSxsXuXYlKQUgDuCd1i/VtiWMHaZB3YIJi4NE7lt7P6yKCWKRrWKszAB+IxT+jSEJqmIbjpLJmKXgmSw4QSludazCLC8oSw10BEiQzMGRjYzIzMC1lNDY4LTQ4MjUtODJkYy01NmUwNjE5OWQ1ZDkaEWR1bW15X2NlbGxhX2dyb3VwIixBcTVqOTA3eFB6X3FWMXNURVF6QjBQeG9rOUQ3LXZYQ1NJOUpHYmpUWjBqZSgBMkAzMzNlNTk0ZjVhZmI4MDBiOTEwMzQ1YWYyMTZmMzgxMDZhZjAyZTY4NmEyMjBjMjA1YjcyZGViOTRiOWY5ZGE1SiC8yojmX4KF6H35TbaSWmPg7Lj32RXfeBALpC8CuWbFhFJB/wwXm8qbsdEzLxNiBaKSXY95apf+hNd3x56IxiYGM0p709YFonLlqAMF2huaP6ofy3O+b7rPmhxt7AIkqw6EtAFCILTDmanTHVRs/0pbV1MJa0x/51e+nS32a1w2OlP5OtdESkHoH67MBllMdah8bCwT6lX26Z5J/4g2y4B2QX6s6ROMv0kEs+xJ07994g7rQMlMvTEWRRboIAglJD+kA34VeMW9AA=="
}

cella_id  : cella id
seed      : cella seed
seed_byts : cella seed byts 



























===== TO BE MODIFIED =====
酒窖（cella）
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