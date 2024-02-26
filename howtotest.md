# How to test producer_sync branch?

## Test environment

| role  | name for short  | working mode |
| ---- | ---- | ---- |
| Bootstrap node      | "bootstrap" | bootstrapnode |
| Owner node          | "owner"   | fullnode |
| Producer node 1     | "p1"      | producernode |
| Producer node 2     | "p2"      | producernode |
| Producer node 3     | "p3"      | producernode |
| User node 1         | "u1"      | fullnode |

**Tips**: it's ok to have all 6 nodes/roles running on one computer (API port should be different)

## build quorum binary

use the head version of producer_sync branch

```sh
make linux
```

check the version and get help:

```sh
./quorum version
./quorum --help
```

## Start all 6 nodes

### 1. start "bootstrap" node:

```sh
./quorum bootstrapnode \
    --keystoredir=mynodes/bootstrapnode/keystore \
    --keystorename=default \
    --keystorepwd=myboot123 \
    --configdir=mynodes/bootstrapnode/config \
    --datadir=mynodes/bootstrapnode/data \
    --listen=/ip4/0.0.0.0/tcp/50510 \
    --listen=/ip4/0.0.0.0/tcp/50511/ws \
    --apiport=50513 \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=3 \
    --log-max-size=10 \
    --logfile=mynodes/bootstrapnode/logs/quorum.log \
    --loglevel=debug
```

check the node status:

```sh
curl http://127.0.0.1:50513/api/v1/node 

{"node_id":"16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP","node_status":"NODE_ONLINE","node_type":"bootstrap"}

```

so, your bootstrap peer is `/ip4/127.0.0.1/tcp/50510/p2p/16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP` , it will be used in other 5 nodes as param peer.

### 2. start fullnode "owner" as group owner

```sh
./quorum fullnode \
    --peername=peer \
    --keystoredir=mynodes/fullnode_owner/keystore \
    --keystorename=default \
    --keystorepwd=myowner456 \
    --configdir=mynodes/fullnode_owner/config \
    --datadir=mynodes/fullnode_owner/data \
    --certdir=mynodes/fullnode_owner/certs \
    --jsontracer=mynodes/fullnode_owner/logs/tracer.json \
    --peer=/ip4/127.0.0.1/tcp/50510/p2p/16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP \
    --listen=/ip4/0.0.0.0/tcp/50610 \
    --listen=/ip4/0.0.0.0/tcp/50611/ws \
    --apiport=50613 \
    --autoack=true \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=3 \
    --log-max-size=10 \
    --logfile=mynodes/fullnode_owner/logs/quorum.log \
    --loglevel=debug
```

### 3. start fullnode "n1" as group user:

```sh
./quorum fullnode \
    --peername=peer \
    --keystoredir=mynodes/fullnode_user/keystore \
    --keystorename=default \
    --keystorepwd=myuser456 \
    --configdir=mynodes/fullnode_user/config \
    --datadir=mynodes/fullnode_user/data \
    --certdir=mynodes/fullnode_user/certs \
    --jsontracer=mynodes/fullnode_user/logs/tracer.json \
    --peer=/ip4/127.0.0.1/tcp/50510/p2p/16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP \
    --listen=/ip4/0.0.0.0/tcp/50620 \
    --listen=/ip4/0.0.0.0/tcp/50621/ws \
    --apiport=50623 \
    --autoack=true \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=3 \
    --log-max-size=10 \
    --logfile=mynodes/fullnode_user/logs/quorum.log \
    --loglevel=debug
```

### 4. start producer node "p1":

```sh
./quorum producernode \
    --peername=peer \
    --keystoredir=mynodes/producernode1/keystore \
    --keystorename=default \
    --keystorepwd=myproducer111 \
    --configdir=mynodes/producernode1/config \
    --datadir=mynodes/producernode1/data \
    --certdir=mynodes/producernode1/certs \
    --peer=/ip4/127.0.0.1/tcp/50510/p2p/16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP \
    --listen=/ip4/0.0.0.0/tcp/50710 \
    --listen=/ip4/0.0.0.0/tcp/50711/ws \
    --apiport=50713 \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=3 \
    --log-max-size=10 \
    --logfile=mynodes/producernode1/logs/quorum.log \
    --loglevel=debug

```

### 5. start producer node "p2":

```sh
./quorum producernode \
    --peername=peer \
    --keystoredir=mynodes/producernode2/keystore \
    --keystorename=default \
    --keystorepwd=myproducer222 \
    --configdir=mynodes/producernode2/config \
    --datadir=mynodes/producernode2/data \
    --certdir=mynodes/producernode2/certs \
    --peer=/ip4/127.0.0.1/tcp/50510/p2p/16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP \
    --listen=/ip4/0.0.0.0/tcp/50720 \
    --listen=/ip4/0.0.0.0/tcp/50721/ws \
    --apiport=50723 \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=3 \
    --log-max-size=10 \
    --logfile=mynodes/producernode2/logs/quorum.log \
    --loglevel=debug
    
```
### 5. start producer node "p3":

```sh
./quorum producernode \
    --peername=peer \
    --keystoredir=mynodes/producernode2/keystore \
    --keystorename=default \
    --keystorepwd=myproducer333 \
    --configdir=mynodes/producernode3/config \
    --datadir=mynodes/producernode3/data \
    --certdir=mynodes/producernode3/certs \
    --peer=/ip4/127.0.0.1/tcp/50510/p2p/16Uiu2HAkxfXvruFGQUAURi83RrkHmATjc44obJ7YN3AnZvGNA8NP \
    --listen=/ip4/0.0.0.0/tcp/50720 \
    --listen=/ip4/0.0.0.0/tcp/50721/ws \
    --apiport=50733 \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=3 \
    --log-max-size=10 \
    --logfile=mynodes/producernode3/logs/quorum.log \
    --loglevel=debug
    
```


### 6. check node status

check the nodes by api: `/node`, all the nodes status should be NODE_ONLINE.

```sh
# bootstrap
curl http://127.0.0.1:50513/api/v1/node 
# owner
curl http://127.0.0.1:50613/api/v1/node 
# user
curl http://127.0.0.1:50623/api/v1/node 
# producer1
curl http://127.0.0.1:50713/api/v1/node 
# producer2
curl http://127.0.0.1:50723/api/v1/node 
# producer3
curl http://127.0.0.1:50733/api/v1/node 
```

### 7. check the network of nodes

```sh
# bootstrap 
curl http://127.0.0.1:50513/api/v1/network 
# owner
curl http://127.0.0.1:50613/api/v1/network 
# user
curl http://127.0.0.1:50623/api/v1/network 
# producer1
curl http://127.0.0.1:50713/api/v1/network 
# producer2
curl http://127.0.0.1:50723/api/v1/network 
# producer3
curl http://127.0.0.1:50733/api/v1/network 


```

## create group and join it

### 1. owner create group 

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"group_timeline"}' http://127.0.0.1:50613/api/v1/group

```

returns:

```log
{
  "seed": "rum://seed?v=1\u0026e=0\u0026n=0\u0026c=_sCLLReWZ3vE8hFs9-EBA4x71uidvH_hU0WQVFplAoM\u0026g=eFjiGAnBQwOh3ZoWNqqabg\u0026k=A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF\u0026s=IMbZvWwgz7moq5sohajEwXMCecGw_mc-5xp__qKVbmFLT2Ri7TYOaMtOICVCo_nTrdk_yxbNB72sQBS3sxJ1hgE\u0026t=FywI0LkbS5Q\u0026a=my_test_group\u0026y=group_timeline\u0026u=http%3A%2F%2F127.0.0.1%3A50613%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI3ODU4ZTIxOC0wOWMxLTQzMDMtYTFkZC05YTE2MzZhYTlhNmUiXSwiZXhwIjoxODI3Mzk5MjU1LCJuYW1lIjoiYWxsb3ctNzg1OGUyMTgtMDljMS00MzAzLWExZGQtOWExNjM2YWE5YTZlIiwicm9sZSI6Im5vZGUifQ.ng2FAlbvnUz7M3lLoxpJA-rI8Dbui849DJAgxo5Acg4",
  "group_id": "7858e218-09c1-4303-a1dd-9a1636aa9a6e"
}
```

the seed-url is used for others nodes to join.

### 2. p1, p2, p3 and u1 join the Group with seed url

```bash
# usernode join the group
curl -X POST -H 'Content-Type: application/json' -d '{"seed":"rum://seed?v=1\u0026e=0\u0026n=0\u0026c=_sCLLReWZ3vE8hFs9-EBA4x71uidvH_hU0WQVFplAoM\u0026g=eFjiGAnBQwOh3ZoWNqqabg\u0026k=A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF\u0026s=IMbZvWwgz7moq5sohajEwXMCecGw_mc-5xp__qKVbmFLT2Ri7TYOaMtOICVCo_nTrdk_yxbNB72sQBS3sxJ1hgE\u0026t=FywI0LkbS5Q\u0026a=my_test_group\u0026y=group_timeline\u0026u=http%3A%2F%2F127.0.0.1%3A50613%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI3ODU4ZTIxOC0wOWMxLTQzMDMtYTFkZC05YTE2MzZhYTlhNmUiXSwiZXhwIjoxODI3Mzk5MjU1LCJuYW1lIjoiYWxsb3ctNzg1OGUyMTgtMDljMS00MzAzLWExZGQtOWExNjM2YWE5YTZlIiwicm9sZSI6Im5vZGUifQ.ng2FAlbvnUz7M3lLoxpJA-rI8Dbui849DJAgxo5Acg4"}' http://127.0.0.1:50623/api/v2/group/join

# producernode 1 join the group
curl -X POST -H 'Content-Type: application/json' -d '{"seed":"rum://seed?v=1\u0026e=0\u0026n=0\u0026c=_sCLLReWZ3vE8hFs9-EBA4x71uidvH_hU0WQVFplAoM\u0026g=eFjiGAnBQwOh3ZoWNqqabg\u0026k=A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF\u0026s=IMbZvWwgz7moq5sohajEwXMCecGw_mc-5xp__qKVbmFLT2Ri7TYOaMtOICVCo_nTrdk_yxbNB72sQBS3sxJ1hgE\u0026t=FywI0LkbS5Q\u0026a=my_test_group\u0026y=group_timeline\u0026u=http%3A%2F%2F127.0.0.1%3A50613%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI3ODU4ZTIxOC0wOWMxLTQzMDMtYTFkZC05YTE2MzZhYTlhNmUiXSwiZXhwIjoxODI3Mzk5MjU1LCJuYW1lIjoiYWxsb3ctNzg1OGUyMTgtMDljMS00MzAzLWExZGQtOWExNjM2YWE5YTZlIiwicm9sZSI6Im5vZGUifQ.ng2FAlbvnUz7M3lLoxpJA-rI8Dbui849DJAgxo5Acg4"}' http://127.0.0.1:50713/api/v2/group/join

# producernode 2 join the group
curl -X POST -H 'Content-Type: application/json' -d '{"seed":"rum://seed?v=1\u0026e=0\u0026n=0\u0026c=_sCLLReWZ3vE8hFs9-EBA4x71uidvH_hU0WQVFplAoM\u0026g=eFjiGAnBQwOh3ZoWNqqabg\u0026k=A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF\u0026s=IMbZvWwgz7moq5sohajEwXMCecGw_mc-5xp__qKVbmFLT2Ri7TYOaMtOICVCo_nTrdk_yxbNB72sQBS3sxJ1hgE\u0026t=FywI0LkbS5Q\u0026a=my_test_group\u0026y=group_timeline\u0026u=http%3A%2F%2F127.0.0.1%3A50613%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI3ODU4ZTIxOC0wOWMxLTQzMDMtYTFkZC05YTE2MzZhYTlhNmUiXSwiZXhwIjoxODI3Mzk5MjU1LCJuYW1lIjoiYWxsb3ctNzg1OGUyMTgtMDljMS00MzAzLWExZGQtOWExNjM2YWE5YTZlIiwicm9sZSI6Im5vZGUifQ.ng2FAlbvnUz7M3lLoxpJA-rI8Dbui849DJAgxo5Acg4"}' http://127.0.0.1:50723/api/v2/group/join

# producernode 2 join the group
curl -X POST -H 'Content-Type: application/json' -d '{"seed":"rum://seed?v=1\u0026e=0\u0026n=0\u0026c=_sCLLReWZ3vE8hFs9-EBA4x71uidvH_hU0WQVFplAoM\u0026g=eFjiGAnBQwOh3ZoWNqqabg\u0026k=A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF\u0026s=IMbZvWwgz7moq5sohajEwXMCecGw_mc-5xp__qKVbmFLT2Ri7TYOaMtOICVCo_nTrdk_yxbNB72sQBS3sxJ1hgE\u0026t=FywI0LkbS5Q\u0026a=my_test_group\u0026y=group_timeline\u0026u=http%3A%2F%2F127.0.0.1%3A50613%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI3ODU4ZTIxOC0wOWMxLTQzMDMtYTFkZC05YTE2MzZhYTlhNmUiXSwiZXhwIjoxODI3Mzk5MjU1LCJuYW1lIjoiYWxsb3ctNzg1OGUyMTgtMDljMS00MzAzLWExZGQtOWExNjM2YWE5YTZlIiwicm9sZSI6Im5vZGUifQ.ng2FAlbvnUz7M3lLoxpJA-rI8Dbui849DJAgxo5Acg4"}' http://127.0.0.1:50733/api/v2/group/join
```

### 3. check the result

check the result with api: `/groups`

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50623/api/v1/groups
curl http://127.0.0.1:50713/api/v1/groups
curl http://127.0.0.1:50723/api/v1/groups
curl http://127.0.0.1:50733/api/v1/groups

```

returns: 

```
{"groups":[{"group_id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","group_name":"my_test_group","owner_pubkey":"A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF","user_pubkey":"AyeWNbBuM_gXIz5833lkH-0sJ6vTxgMxJbvdPvNPArl5","user_eth_addr":"0x8296B71193e8fC7706b29Ccf1aD19e49a577b125","consensus_type":"POA","encryption_type":"PUBLIC","cipher_key":"fec08b2d1796677bc4f2116cf7e101038c7bd6e89dbc7fe1534590545a650283","app_key":"group_timeline","epoch":0,"last_updated":1669719254399404948,"group_status":"BLOCK_SYNCING"}]}

```

**TIP**: the value of epoch is import, it should be changed with the following steps.

## owner work as the only producer: 

### 1. owner or u1 send a POST

```bash
curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the owner node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50613/api/v1/group/content/false

curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the user node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50623/api/v1/group/content/false

```

returns:

```bash
{
  "trx_id": "c712d274-e158-4ff2-86ef-7bb77cc76a52"
}
{
  "trx_id": "d24853e1-67d7-4cf4-9d4b-a41c14d00d8b"
}

```


### 2. verify a new block is added to ALL nodes and can be get by using block API

check epoch value by /groups api

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50623/api/v1/groups
curl http://127.0.0.1:50713/api/v1/groups
curl http://127.0.0.1:50723/api/v1/groups
curl http://127.0.0.1:50733/api/v1/groups

```

returns:

```sh
{
  "groups": [
    {
      "group_id": "7858e218-09c1-4303-a1dd-9a1636aa9a6e",
      "group_name": "my_test_group",
      "owner_pubkey": "A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF",
      "user_pubkey": "A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF",
      "user_eth_addr": "0xBD90f2964D12a20aA33751020ab2aeC987a1184c",
      "consensus_type": "POA",
      "encryption_type": "PUBLIC",
      "cipher_key": "fec08b2d1796677bc4f2116cf7e101038c7bd6e89dbc7fe1534590545a650283",
      "app_key": "group_timeline",
      "epoch": 1,
      "last_updated": 1669719619,
      "group_status": "IDLE"
    }
  ]
}
```

check one block by /block api with the value of epoch

`api/v1/block/:group_id/:epoch `


```sh
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/1 
```

returns:

```sh
{
  "GroupId": "7858e218-09c1-4303-a1dd-9a1636aa9a6e",
  "Epoch": 1,
  "PrevEpochHash": "pjDj6AdP1O0Olh7b9DVNuTN1yREYWKLBz+CfyHta9K0=",
  "Trxs": [
    {
      "TrxId": "c712d274-e158-4ff2-86ef-7bb77cc76a52",
      "GroupId": "7858e218-09c1-4303-a1dd-9a1636aa9a6e",
      "Data": "IsHuc/RRwz6Qqg2bEMlA6aS5F/oraX9oFUuvyM+8lGTy5QoN95JZ80YZpEcOLou+tAB8NwZwkzY/zDnOaullkzZLci2klG7Az1umPibKL3Uf+tLSfpxPJBO00RvF90zS",
      "TimeStamp": "1669719619538026628",
      "Version": "2.0.0",
      "Expired": 1669719649538026708,
      "SenderPubkey": "A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF",
      "SenderSign": "EWLt3Y7hPS11JyuturM2ZTZ5/nZJyQxOiybGSSqmkIMibXThLQdcYK3hvY5JTPKQCcv1ZnM09qHq0FgpVdIKGwA=",
      "SudoTrx": true
    }
  ],
  "EpochHash": "QmVuGCZWgWYMbACaXKReWjb107f7eQuu+F6M10AM3x4=",
  "TimeStamp": "1669719619560270815",
  "BlockHash": "MANgzBkjCEMABbwuLCEBVWfU0nT0Mn0hVHzXxx+xKtQ=",
  "BookkeepingPubkey": "A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF",
  "BookkeepingSignature": "1uDLcZcJrYzrwB++rbBlK7Yoi4NK35Wt53ojaZpfTEljUhocuGrASMK+KqXcMcbwvJvFggsB8Nicb06c5+RXPgE="
}

```


### 3. verify trx is ONLY applied at node owner and u1 and can be get by using trx API

`api/v1/trx/:group_id/:trx_id `


node with trx applied returns:

```sh

curl http://127.0.0.1:50613/api/v1/trx/7858e218-09c1-4303-a1dd-9a1636aa9a6e/c712d274-e158-4ff2-86ef-7bb77cc76a52


{"TrxId":"c712d274-e158-4ff2-86ef-7bb77cc76a52","Type":"POST","GroupId":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","Data":"IsHuc/RRwz6Qqg2bEMlA6aS5F/oraX9oFUuvyM+8lGTy5QoN95JZ80YZpEcOLou+tAB8NwZwkzY/zDnOaullkzZLci2klG7Az1umPibKL3Uf+tLSfpxPJBO00RvF90zS","TimeStamp":"1669719619538026628","Version":"2.0.0","Expired":"1669719649538026708","ResendCount":"0","Nonce":"0","SenderPubkey":"A9MSO6MW0m-m-h7GQlH6fk34jsxNOzoUayJlws3lRTqF","SenderSign":"EWLt3Y7hPS11JyuturM2ZTZ5/nZJyQxOiybGSSqmkIMibXThLQdcYK3hvY5JTPKQCcv1ZnM09qHq0FgpVdIKGwA=","StorageType":"CHAIN","SudoTrx":true}

```

node with trx not applied returns:

```sh
curl http://127.0.0.1:50613/api/v1/trx/7858e218-09c1-4303-a1dd-9a1636aa9a6e/d24853e1-67d7-4cf4-9d4b-a41c14d00d8b

{"TrxId":"","Type":"POST","GroupId":"","Data":"","TimeStamp":"0","Version":"","Expired":"0","ResendCount":"0","Nonce":"0","SenderPubkey":"","SenderSign":"","StorageType":"CHAIN","SudoTrx":false}

```


## add 3 producers nodes 

### 1. p1 p2 and p3 announce as group producer

p1 e.g:

```sh

curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e", "action":"add", "type":"producer", "memo":"producer p1, realiable and cheap, online 24hr"}' http://127.0.0.1:50713/api/v1/group/announce 

{"group_id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","sign_pubkey":"AyeWNbBuM_gXIz5833lkH-0sJ6vTxgMxJbvdPvNPArl5","encrypt_pubkey":"","type":"AS_PRODUCER","action":"ADD","sign":"2140ff057e56395b9ea658df450d4c4a14c468f24fb337fb2383828caca920f76e73176f25ccfece4725965cdede29b761b6b02add837980de56fa045a00c8ca01","trx_id":"a02ec961-aee6-4a3d-9e3a-1e72aa32de63"}

```

### 2. verify producers announced

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/2
```

### 3. owner approve p1 and p2 as producers

```sh
curl -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":["AqSEeptDka8_5jy9Hmen8HJImNPFDxVdkWUOHo3q0UTW","AyeWNbBuM_gXIz5833lkH-0sJ6vTxgMxJbvdPvNPArl5", "pubkey_of_p3"] ,"group_id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e", "action":"add"}' http://127.0.0.1:50613/api/v1/group/producer/false

```

check the approve trx is applied:

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/3
```

### 4. owner or u1 send a POST

```bash
curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the owner node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50613/api/v1/group/content/true

curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the user node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50623/api/v1/group/content/false

```


### 5. verify a new block is added to ALL nodes and can be get by using block API

check the epoch value changed to 4 (e.g.):

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4
```


```sh
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4 
curl http://127.0.0.1:50623/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4 
curl http://127.0.0.1:50713/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4 
curl http://127.0.0.1:50723/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4
curl http://127.0.0.1:50733/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4  
```

### 6. verify trx is ONLY applied at node owner and u1 and can be get by using trx API

`curl http://127.0.0.1:50613/api/v1/trx/:group_id/:trx_id`


## owner node offline and recovery

### 1. stop the owner node

### 2. u1 send a POST

```sh
curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the user node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50623/api/v1/group/content/false

```

### 3. verify a new block is added to ALL nodes and can be get by using block API

check the epoch value changed to 4 (e.g.):

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/4
```

### 4. verify trx is ONLY applied at node owner and u1 and can be get by using trx API

e.g: `curl http://127.0.0.1:50613/api/v1/trx/:group_id/:trx_id`


### 5. owner node quit

### 6. u1 send several trxs to producer some new blocks

### 7. verify all 4 nodes (p1, p2, p3, n1) have same blocks

### 8. start owner node again

check the node staus is NODE_ONLINE:

```sh
curl http://127.0.0.1:50613/api/v1/node
```

### 6. owner node should start sync and get all missing blocks

check the value of group epoch of nodes:

```sh
curl http://127.0.0.1:50613/api/v1/groups
curl http://127.0.0.1:50623/api/v1/groups
curl http://127.0.0.1:50713/api/v1/groups
curl http://127.0.0.1:50723/api/v1/groups
curl http://127.0.0.1:50733/api/v1/groups
```

check the block api of owner node:

```sh
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/5
```

### verify chain still working


------------------------ DONT TEST BELOW SCENES------------------------------

chain should be able to recovery from total failure

owner and u1 post to group, and the group epoch is changed to add 1.

```bash
curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the owner node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50613/api/v1/group/content/true

curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"I am the user node!"},"target":{"id":"7858e218-09c1-4303-a1dd-9a1636aa9a6e","type":"Group"}}'  http://127.0.0.1:50623/api/v1/group/content/false

```

check the block:

```
curl http://127.0.0.1:50613/api/v1/block/7858e218-09c1-4303-a1dd-9a1636aa9a6e/:epoch 
```

## all producers offline and recovery

### 1. close owner, p1, p2 (all the producers of the group)

### 2. Start owner

owenr should start consensus sync 

### 3. Start p1

p1 should start consensus sync and make agreement with owner and back to idle

chain should ready for work

### 4. Start p2

p2 should start consensus sync and back to idle

### 5. Chain is fully recovered


