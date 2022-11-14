# New node type

## There are 3 types you can choose when start a node

- "bootstrapnode"
  Start a bootstrap node, bootstrap node is the old "bootstrap" mode when start a node

- "fullnode"
  a full node can create group and work as the first producer of the group created by this node
  a full node can join group created by other owner (as a user)
  a full node can NOT announce itself as a "producer" when join group created by other owner (can only announced as user)
  a group run on full node will apply ALL trxs received in a produced block

  create a group when run as "full node"
    a "producer" and a "user" will be created , this producer act as the first "producer" for this group, you can involve more producers after the group is created (follow the previous announce/approve process)
  
  join a group when run as "full node"
    only a "user" will be created, when run in full node, after join a group created by other owner, you CAN NOT ANNOUNCE youself as a "producer"
  

- "producernode"
  Start a producer only node
  a producer node can NOT create it own group
  a producer node can join group created by other owner (as a producer)
  a producer node can NOT "post" to a group it joined
  a producer node can announce itself as a "producer" when join group created by other owner
  a group run on producer node will save the block to local db but WILL NOT apply POST type trxs, that means you can NOT get content of a group from a group fun on producer node

# How to setup and test new consensus 

## start boot strap node
go run main.go bootstrapnode --listen /ip4/0.0.0.0/tcp/10666

## start node owner (fullnode)
RUM_KSPASSWD=123 go run main.go fullnode --peername owner --listen /ip4/127.0.0.1/tcp/7002 --apiport 8002 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm68RHVt6NedGSiKmkRhadaxoKSRuawnEnmi7jznN7fwLm --configdir config --datadir data --keystoredir ownerkeystore  --jsontracer ownertracer.json --debug=true

## Start node producer1 (producernode)
RUM_KSPASSWD=123 go run main.go producernode --peername p1 --listen /ip4/127.0.0.1/tcp/7003 --apiport 8003 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm68RHVt6NedGSiKmkRhadaxoKSRuawnEnmi7jznN7fwLm --configdir config --datadir data --keystoredir p1keystore  --jsontracer p1tracer.json --debug=true

## Start node producer2 (producernode)
RUM_KSPASSWD=123 go run main.go producernode --peername p2 --listen /ip4/127.0.0.1/tcp/7004 --apiport 8004 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm68RHVt6NedGSiKmkRhadaxoKSRuawnEnmi7jznN7fwLm --configdir config --datadir data --keystoredir p2keystore  --jsontracer p2tracer.json --debug=true

## Start node user1 (fullnode)
RUM_KSPASSWD=123 go run main.go fullnode --peername n1 --listen /ip4/127.0.0.1/tcp/7005 --apiport 8005 --peer /ip4/127.0.0.1/tcp/10666/p2p/16Uiu2HAm68RHVt6NedGSiKmkRhadaxoKSRuawnEnmi7jznN7fwLm --configdir config --datadir data --keystoredir n1keystore  --jsontracer u1tracer.json --debug=true


## Owner create a new group
 curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' http://127.0.0.1:8002/api/v1/group

## owner, producer1, producer2, u1 join the group just created

## owner post content to group
 curl -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"df24e848-63d3-424f-96e4-c0a8e8413b71","type":"Group"}}'  http://127.0.0.1:8002/api/v1/group/content
{
  "trx_id": "9251dae7-a5d9-4d41-a223-0e9d75a5a42a"
}

## a new block will be created immediately

## Check block (on all node) and trx (only on fullnode)
curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:8002/api/v1/block/b41da0dc-c19d-4ed3-9528-46d86bd6df4e/1

and the result looks like

{
  "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
  "Epoch": 1,
  "PrevEpochHash": "rW3zht7+tQ7fO51eIo0CFiDuOUCv0niPA/j3Pl27o/s=",
  "Trxs": [
    {
      "TrxId": "95b81a71-7d4c-466b-b3f2-59a422906a94",
      "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
      "Data": "e/6XjpssR203P6PFwouw0mJu7gFuMUTCPiaLZJ58Wk8czXT+4Kfvy6M8Yx8DOqD5AzwGq8kDxECIGGHi5bnBxOtOELYQAif9/930huzwRtkpxXc64f/Byc+FBsnytIwMhJuGp/N0iZ5veRy3i5ylHA==",
      "TimeStamp": "1662663644859650912",
      "Version": "1.0.0",
      "Expired": 1662663674859650982,
      "SenderPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
      "SenderSign": "Lph8LVcLavf0HBauxiIPjGnaxVRfTeFVysuq4vM3gOlWt3pxJX1qqF50Y/3P6hp7rSTmnbbns0Ud3oDM0BJ/iAE="
    }
  ],
  "EpochHash": "S+TOY4fqHIhJU24STYCB+Vj6+uR6pOvOwaYjx/e7AK8=",
  "TimeStamp": "1662663644863062170",
  "BlockHash": "1Wh6/WtMvc8GvO7bsclZ+oqQhYpErNagZxqk4ffFoV8=",
  "BookkeepingPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
  "BookkeepingSignature": "3YapwnIMqjQPnPxFaWfxqy9hdJ2mPoU0xJVYu39f9dVJ7cTwrYrHJQWVfSn21WHSv5LNIr+WLAZwJ1uR/OCSogA="
}

curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:8002/api/v1/trx/b41da0dc-c19d-4ed3-9528-46d86bd6df4e/95b81a71-7d4c-466b-b3f2-59a422906a94 | jq

and the result looks like

{
  "TrxId": "95b81a71-7d4c-466b-b3f2-59a422906a94",
  "Type": "POST",
  "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
  "Data": "e/6XjpssR203P6PFwouw0mJu7gFuMUTCPiaLZJ58Wk8czXT+4Kfvy6M8Yx8DOqD5AzwGq8kDxECIGGHi5bnBxOtOELYQAif9/930huzwRtkpxXc64f/Byc+FBsnytIwMhJuGp/N0iZ5veRy3i5ylHA==",
  "TimeStamp": "1662663644859650912",
  "Version": "1.0.0",
  "Expired": "1662663674859650982",
  "ResendCount": "0",
  "Nonce": "0",
  "SenderPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
  "SenderSign": "Lph8LVcLavf0HBauxiIPjGnaxVRfTeFVysuq4vM3gOlWt3pxJX1qqF50Y/3P6hp7rSTmnbbns0Ud3oDM0BJ/iAE=",
  "StorageType": "CHAIN"
}

## Producer1 and producer2 announce as producer
curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"b41da0dc-c19d-4ed3-9528-46d86bd6df4e", "action":"add", "type":"producer", "memo":"producer p1, realiable and cheap, online 24hr"}' http://127.0.0.1:8003/api/v1/group/announce | jq

curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"b41da0dc-c19d-4ed3-9528-46d86bd6df4e", "action":"add", "type":"producer", "memo":"producer p2, realiable and cheap, online 24hr"}' http://127.0.0.1:8004/api/v1/group/announce | jq

## new block will be created, trxs will be applied to all node, all 4 node should have the same "announced" producer list

## Owner promote (add) p1 and p2 and new producers
 curl -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":'[\"AgBnNJKDTw1l39nZenbbueCM6wFu9_GDwPR7T1lAuZxO\",\"AmSNt3fIEfxueMHF2MQJqWxFqUV7m4jmWK-dWupKEkbg\"]',"group_id":"b41da0dc-c19d-4ed3-9528-46d86bd6df4e", "action":"add"}' http://127.0.0.1:8002/api/v1/group/producer | jq

result:
 {
  "group_id": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
  "Producers": [
    {
      "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
      "ProducerPubkey": "AgBnNJKDTw1l39nZenbbueCM6wFu9_GDwPR7T1lAuZxO",
      "GroupOwnerPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
      "GroupOwnerSign": "524df0c49dddd74ee32127dad228a6acf3819230f594787306b3a5bcc354377f27a3361272125609b8262947e38945db51b3c2552f539322ea0e1ed063da5f9201",
      "TimeStamp": "1662663806828170050"
    },
    {
      "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
      "ProducerPubkey": "AmSNt3fIEfxueMHF2MQJqWxFqUV7m4jmWK-dWupKEkbg",
      "GroupOwnerPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
      "GroupOwnerSign": "9f3c1acf388c105d8d401ec56588fb9a511dbd6c672dda8bf7bc26783a67312107f75dffe07d9313be62734c960f4f77f6cb011e15b42e15db711138e444254101",
      "TimeStamp": "1662663806828230670"
    }
  ],
  "trx_id": "509469a4-b326-4ae1-8781-5097126b74ea",
  "memo": "",
  "action": ""
}

## p1 and p2 will be promoted to producers, all 3 producers (p1, p2, owner) now work together

## Owner or u1 post new content

## all 3 producers will make agreement and create new block, all 4 nodes will get the newly produced block and 2 full nodes will apply the POST trx
 
** The block created by 3 producers are "actually different"
{
  "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
  "Epoch": 1,
  "PrevEpochHash": "rW3zht7+tQ7fO51eIo0CFiDuOUCv0niPA/j3Pl27o/s=",
  "Trxs": [
    {
      "TrxId": "95b81a71-7d4c-466b-b3f2-59a422906a94",
      "GroupId": "b41da0dc-c19d-4ed3-9528-46d86bd6df4e",
      "Data": "e/6XjpssR203P6PFwouw0mJu7gFuMUTCPiaLZJ58Wk8czXT+4Kfvy6M8Yx8DOqD5AzwGq8kDxECIGGHi5bnBxOtOELYQAif9/930huzwRtkpxXc64f/Byc+FBsnytIwMhJuGp/N0iZ5veRy3i5ylHA==",
      "TimeStamp": "1662663644859650912",
      "Version": "1.0.0",
      "Expired": 1662663674859650982,
      "SenderPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
      "SenderSign": "Lph8LVcLavf0HBauxiIPjGnaxVRfTeFVysuq4vM3gOlWt3pxJX1qqF50Y/3P6hp7rSTmnbbns0Ud3oDM0BJ/iAE="
    }
  ],
  "EpochHash": "S+TOY4fqHIhJU24STYCB+Vj6+uR6pOvOwaYjx/e7AK8=",
  "TimeStamp": "1662663644863062170",
  "BlockHash": "1Wh6/WtMvc8GvO7bsclZ+oqQhYpErNagZxqk4ffFoV8=",
  "BookkeepingPubkey": "A8PjBP60vtkERRhSKJuK73muBv8nj0QQ3fgpmErO_rI9",
  "BookkeepingSignature": "3YapwnIMqjQPnPxFaWfxqy9hdJ2mPoU0xJVYu39f9dVJ7cTwrYrHJQWVfSn21WHSv5LNIr+WLAZwJ1uR/OCSogA="
}

The following items will be same for all blocks produced by different Producers
- GroupID
- Epoch
- PrevEpochHash
- Trxs
- EpochHahs (hash only for the 4 items above)

The following items will NOT be the same for blocks produced by different producers
- BlochHash (hash include every items in a block)
- BookkeepingPubkey ( who "bookkeeping" this block, same as the usersignpubkey for this producer in group)
- BookkeepingSignature ( signature of the block "bookkeeping")


## Block_Id is eliminated and "epoch" number are used to mark different blocks