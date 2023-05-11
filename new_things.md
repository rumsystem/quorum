# 3 types of node

## "bootstrapnode"
bootstrapnode is the old "bootstrap" mode

## "fullnode"

fullnode is a fully functional rum node.

a full node can:
- create new group
- join group created by other owner as a "user"
- access all group functions by using rum apis
- apply ALL TYPE OF TRXS, that means only full node can get contents of the group

a full node can not:
- announce itself as a "producer" when join other group
- work as a "producer" when join other groups

When create a new group
- the creator is the "owenr" of the new group
- the creator will announce itself as the first "producer" of the group
- the creator will work as the first "producer" of the group

A typical rum user should create and only create fullnode unless you want to provide resources (network/cpu/storage) and workas a "producer" for group created by other users
  
## "producernode"
a producer node is a node dedicated to provide withness service(and resources) as a "producer" for a group

a producer node can :
- join other group
- announce itself as a "producer"
- after select by group owner, a producer node will start work as a group producer
- quit group 

a producer node can not:
- create it own group
- send "POST" type trx 
- get content of the a group

only group producers should consider start a "producer node"

# Group Owner
- group creator becomes the "owner" of the group
- the fullnode creator run will "host" the group instance and provide api for other node to use 
- group owner will announce and work as the first group producer
- owner should share this group by provide group seed to other people
- owner has full control of the group, can change the following config
  - auth 
  - consensus
  - app

## RexSync
RUM use rex sync protocl to get new blocks from other connected nodes, when join a group or back from offline status, a node will start RexSync automatically and ask for any missing blocks start from it top block.  All nodes has responsibility to give blocks when asked by other nodes in this group. 

## Consensus
1 or Some nodes will work as "producer" of the group, they have responsibility to:
- collect trxs send by group users
- save and sort those trxs in local buffer
- select a bench of trxs from local buffer and propose them in an "epoch", if no trxs to propose, a producer will propose "EMPTY" message for this epoch
- all producers will following HBBFT protocal to make an agreemnet for each "epoch"
- when agreement is made, all producers should have same set of trxs selected to be packaged inside a new block 
- all producers will generate a new block to package all selected trxs, sign it and broadcast it to group
- all producers will move to next epoch and start again till new agreement is made

RUM implement HBBFT(https://eprint.iacr.org/2016/199.pdf) protocol to make consensus among multiple producers when make new blocks
- N is num of total producer node
- f is num of failable producer node
- HBBFT request:
f * 3 < N
e.g. 2 producers, failable node f = 0 (0 * 3 < 2)
3 producers, failable node f = 0 (0 * 3 < 3)
4 producers, failable node f = 1 (1 * 3 < 4)
...
10 producers, failable node f = 3 (3 * 3 < 10)       

- Update HBBFT config
owner can change group HBBFT consensus setting
- owner can assign new producers for this group (by approve some announced producers) 
- owner can change the epoch duration
- after new HBBFT config is assign, a "negotiation" process will start among all selected producers 
- owner will give new epoch to start with
- a "conculation" should be made by all selected producers, they should sign and send their proof of agree with the assignment
- after the new agreement is agree by enough producers, all producers will work by following the new consensus config

## a full node join a group (except his own group)
full node become a "user" of the group, it will:
- create this group locally
- save seed and genesis block 
- try connect with other nodes in this group
- start RexSync to get all valid blocks for this group and save all received blocks to local db
- when receive new block, apply all valid trxs in the blocks
- provide apis to access all group functions, like 
  - send new trx
  - get group content
  - get group config
  - get app config
  - get group info
  - get group consensus info
  - get block 
  - get trx
  ...

## a producer node join a group
after join a group, a producer node will:
- create this group locally 
- save group seed and gensis block 
- try connect with other node in this group 
- start RexSync to get all valid blocks from other nodes and save all block locally, only certain type of trxs will be applied.
a producer node need "announce" itself as a "producer" to the group
- call announce API to "register" itself as a group producer
- group user can list all "announced" producers 
- till now, a producer is NOT a "active" producer, it needs to be selected and approved by group owenr

after select by the group owner, the producer node start work as a active group producer

## How consensus was made inside a group
- After join a group by useing group seed. a node can create/sign and send trxs to the group
- some node will work as a "producer" of the gorup
- Producers will collect all trxs sent to this group and buffer them locally 
- Producers will use HBBFT protocol to make consensus
  - Consensus are made round by round
  - Each round of consensus is called an "epoch"
  - If consensus was made, epoch will increased
  - If no consensus was made for this round, producer will retry by using the same epoch number till consensus was made
- By following epoch rhythm, a producers will pickup some trxs to propose
- If a producer has no trx to propose, it will propose a "EMPTY" mark
- Each producer will pickup trxs independently (although they may select the same trxs) to propose in an epoch
- after the HBBFT protocl is finished, a consensus for this Epoch will be made, for example:
```
{
    Epoch : 100
    Trxs proposed by producer A {TX1, TX2, TX3}
    Trxs proposed by producer B {TX1, TX7, TX8}
    Trxs proposed by producer C {TX4, TX2, TX3}
    Trxs proposed by producer D {"EMPTY"}
}
```
- by following HBBFT protocol, all producers will have the same result after consensus is made
- After that ALL active producers will 
    1. organizd their result and make a trxs list to package into a new block
    2. Make new block to package all Trxs, for example:
    ```
    {
        BlockId        : 1
        Trxs           : [TX1, TX2, TX3, TX4, TX7, TX8]
        Previsou Block : 0
        ProducerPubkey : {ProducerPubkey}
        Hash           : {hash}
        Sign           : {producer_sign}
    }
    ```
    3. broadcast the newly built block to group
    4. remove packaged trxs from trx cache
    5. increase current Epoch number
    6. wait till next trx propose epoch 
    7. repeat the same process till node quit or remove from the producer list by owner
- Even if there is only 1 producer, the same conseusus procedure will be followed. 

## change consensus process
- Only owner has the authority to change "consensu" of a group
    - Change producers of the group
    - Change trx propose intervel (epoch time)
- A producer needs to announce itself as a "PRODUCER" before approved by the owner by using announce API
- Quorum fullnode provide several API to handle this task, more detail will be given by using examples
- Change consensus on a running p2p network is a challenge in critual contion, a "consensus agreement signature" process is implemented 
  1. All producers requested by owner will join the agreement sigunature process
  2. Owner will give an ordinal includes:
    - Agreement tick window, in this time frame, all producers need "sign" the agreement all together
    - Agreement tick retry cnt
  3. The change consensus request is initialed by group owner, owner will broadcast a CHANGE_CONSENSUS_REQ to all producers
  ```
  {
      req_id                : Req_ID
      producer_list         : [new_group_producer_list]
      trx_propose_interval  : time interval for each epoch
      agreement_tick_length : time for each round of agreemnet
      agreement_tick_cnt    : retry time
  }
  ```
  4. Owner will give a trx_id as return value of the API
  5. After change conseusus finished, owner will send the result as a CONSENSUS type Trx by using this trx_id
  6. Node user can monitor if this trx is packaged successful as an evidence of change consensus finished
    
- when receive req from owner, all producers in request list will create and sign a "proof" 
  1. A proof consist of 2 parts:
  ```
  {
      req  : original_req
      resp : response from producer
  }
  ```
  a response has the following items
  ```
  {
      resp_id        : resp_id
      req            : the original req
      producerPubKey : producer pubkey
      Hash           : hash
      Sign           : producer sign 
  }
  ```
- After create proof, all producer start make consensus agreement by using HBBFT
- Within retry cnt, in a agreement time frame, all producer should propose its own proof
- Just like a trx, when consensus agreement finished, all particapicated producers should all have the same result includes all necessary proofs
```
{
    owner_proof,
    producer_1_proof,
    producer_2_proof,
    ...
}
```
- If a agreement was made, all producer stop current consensus and start next round epoch by using new consensus configuration
- Owner send a CONSENSUS type trx (user the trx_id in return value when call change consensus API)to tell all other nodes in this group consensus config has been changed. 
- All user node, after receive and verify this CONSENSUS Trx, will apply it locally and update all consensus related config (for example, update producer pool)

## How to test
- A local bootstrap node should be launched 

### Single producer
- Launch a Full_Node o1
- Create a new（or join a previous created group) at o1

```
curl -X POST -H 'Content-Type: application/json' -d '{"seed": "rum://seed?v=1&e=0&n=0&c=jlHZ0yV07L7LjT03TILSL2ILwnfpsAMz44AyBy1MvS4&g=zZkc4n27Qi6fXKD1UuTYlg&k=AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B&s=6t5Ds-keygP8JwPG0i6V05RmNNu8IaRYU0-UTrVaMaRj2_ekgYf1rhUux5Z9_jwSEAyMAFY0hTmw7OyGVmJTAAA&t=F1eyqmr-x78&a=my_test_group&y=test_app&u=http%3A%2F%2F127.0.0.1%3A8002%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyJjZDk5MWNlMi03ZGJiLTQyMmUtOWY1Yy1hMGY1NTJlNGQ4OTYiXSwiZXhwIjoxODM5Njg5NDMxLCJuYW1lIjoiYWxsb3ctY2Q5OTFjZTItN2RiYi00MjJlLTlmNWMtYTBmNTUyZTRkODk2Iiwicm9sZSI6Im5vZGUifQ.trc-OrMabglMDgZTSTMGcqOLqIPIsgsIr1_ddo5SVS0"}' http://127.0.0.1:8002/api/v2/group/join
```
Result
```
{
    "group_id": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
    "group_name": "my_test_group",
    "owner_pubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
    "user_pubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
    "user_encryptpubkey": "age1zd7ehw55e780ll8m2nuyz9275k5hzpu6h0mr2vl5dxt6ymsdsutqnusd0w",
    "consensus_type": "poa",
    "encryption_type": "public",
    "cipher_key": "8e51d9d32574ecbecb8d3d374c82d22f620bc277e9b00333e38032072d4cbd2e",
    "app_key": "test_app",
    "signature": "e76aa89d58ce569458d3635b135bfb495a273c9977ce1a40d738f74f9904a477267cd7700ad7d5a5647d0565ef9c7e43cbb152a0c93a8b5642249f639d7eae1e01"
}
```

- After create a new group, owner will announce itself as the first producer:
```
curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:8002/api/v1/group/cd991ce2-7dbb-422e-9f5c-a0f552e4d896/announced/producers
```
Result:
```
{
    "producers": [
        {
        "GroupId": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
        "Content": {
            "Type": 1,
            "SignPubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
            "EncryptPubkey": "age1zd7ehw55e780ll8m2nuyz9275k5hzpu6h0mr2vl5dxt6ymsdsutqnusd0w",
            "Memo": "owner announce as the first group producer"
        },
        "AnnouncerPubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
        "Hash": "X4nqkzgOrQqbQpgSrWqOt26EktAPenfO5w4rQrwADcg=",
        "Signature": "564nJHVf5dPHbbGsO7zSZ00tsxrNCyR8JC5cGNv332c8/I6eHpenbRuMDV3sOnoX47v96C7SjBRsyevxDFPXcwE="
        }
    ]
}
```

- as the first producer, owner will add a consensus proof automatically
```
curl -X GET -H 'Content-Type: application/json' -d {} http://127.0.0.1:8002/api/v1/group/cd991ce2-7dbb-422e-9f5c-a0f552e4d896/consensus/proof/last
```
Result: 
```
{
    "result": "SUCCESS",
    "req": {
        "ReqId": "5129cd9a-2d05-497e-bd9c-01604108bdcf",
        "GroupId": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
        "ProducerPubkeyList": [
        "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B"
        ],
        "TrxEpochTickLenInMs": 1000,
        "SenderPubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
        "MsgHash": "jNqN0HPUnhjCKYeKfLA3HbPhFMSPt2+aR9xRYOYYCic=",
        "SenderSign": "rQoEs/uk+RSgbwWlTAuV9SrFFeoqZ/PDIincxKgxCR9IdfQB2bslxX5bNyFmi6BT7IZy72wFfNbK4Q5yhHoVZwA="
    },
    "repononsed_producer": [
        "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B"
    ]
}
```
Parameters:
```
Result: Change Consensus Result, can be
  - SUCCESS
  - TIMEOUT
  - FAILED

Req:
  - ReqId   : req_id
  - GroupId : group_id
  - ProducerPubkeyList : which announced producer(s) is(are) requested  to work as a producer 
  - TrxEpochTickLenInMs : for how long an epoch (propose trx) will be start
  - SenderPubkey : should be owner 
  - MsgHash
  - senderSign   : should be siged by owner

ResponsedProducers:
  - which producer(s) is (are) responsed for this request
```

- quorum provids the following APIs to let user check consensus proof
  - Get the latest consensus proof
```
curl -X GET -H 'Content-Type: application/json' -d{} http://127.0.0.1:8002/api/v1/group/:group_id/consensus/proof/last
```

  - Get the consensus change proof history
```
curl -X GET -H 'Content-Type: application/json' -d{} http://127.0.0.1:8002/api/v1/group/:group_id/consensus/proof/history
  ```

  - Get proof by using req_id
```
curl -X GET -H 'Content-Type: application/json' -d{} http://127.0.0.1:8002/api/v1/group/:group_id/consensus/proof/:req_id
```

- User can retrive current consensus info by using the following API
```
curl -X GET -H 'Content-Type: application/json' -d{} http://127.0.0.1:8002/api/v1/group/cd991ce2-7dbb-422e-9f5c-a0f552e4d896/consensus/
```
Result: 
```
{
"producers": [
    {
    "GroupId": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
    "ProducerPubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
    "Memo": "Owner Registated as the first group producer"
    }
],
"trx_epoch_interval": 1000,
"proof_req_id": "11aaff04-f566-4008-99e3-3789a3a5b7b6",
"curr_epoch": 2,
"curr_block_id": 0,
"last_update": 1682364270159316000
}
```

Parameters
```
- Producers          : Current group producer
- trx_epoch_interval : current Epoch time duration        
- proof_req_id       : req_id of the proof that make this consensus configuration
- curr_epoch         : current epoch
- curr_block_id      : current block id
- last_update        : last update
```

- DON'T misunderstanding the consensus info with group info, group info looks like:
```
curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/groups | jq
```
result 
```
{
"groups": [
    {
    "group_id": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
    "group_name": "my_test_group",
    "owner_pubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
    "user_pubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
    "user_eth_addr": "0x38FE32733fD9855367a0148B4C7B00d99535FDb5",
    "consensus_type": "POA",
    "encryption_type": "PUBLIC",
    "cipher_key": "8e51d9d32574ecbecb8d3d374c82d22f620bc277e9b00333e38032072d4cbd2e",
    "app_key": "test_app",
    "currt_top_block": 0,
    "last_updated": 1682357738877044000,
    "rex_syncer_status": "IDLE",
    "rex_Syncer_result": null
    }
]
}

** item current_epoch has been moving from group info api to consensus info api
```    

- o1 start change consensus:
```
curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"cd991ce2-7dbb-422e-9f5c-a0f552e4d896","start_from_epoch":10000, "trx_epoch_tick":3000, "agreement_tick_length":10000, "agreement_tick_count":10, "producer_pubkey":["AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B"]}'  http://127.0.0.1:8002/api/v1/group/updconsensus
```
Result:
```
{
  "group_id": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
  "Producers": [
      "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B"
  ],
  "start_from_epoch": 10000,
  "trx_epoch_tick": 3000,
  "trx_id": "133c71b9-9f5a-464c-accd-eb2e33ae10f5",
  "failable_producers": 0,
  "memo": ""
}

```

Parameters:
```
API : /api/v1/group/updconsensus
    Parameters :
    group_id              : groupid
    start_from_epoch      : producers should start produce by using this epoch number
    trx_epoch_tick        : producers should follow this epoch interval when propose trx
    agreement_tick_Length : lenght (in ms) for each agreement round
    agreement_tick_count  : make agreement retry count 
    producer_pubkey       : producers pubkey list
```
    
Result break down:
```
- producers "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B" will be the new group producer
- Epoch should start from 10000
- Trx propose interval should be 10000ms (10s)
- trx with id "133c71b9-9f5a-464c-accd-eb2e33ae10f5" will be send by owner after change consensus done
```
            
a trx_id will be given back to API caller
  - node user should try to get this trx, if a trx with this trx_id is packaged, that means the change consensus agreement is done successful.

```
curl -X GET -H 'Content-Type: application/json' -d{} http://127.0.0.1:8003/api/v1/trx/cd991ce2-7dbb-422e-9f5c-a0f552e4d896/133c71b9-9f5a-464c-accd-eb2e33ae10f5
```
Result:
```
  {
      "TrxId": "133c71b9-9f5a-464c-accd-eb2e33ae10f5",
      "Type": 2,
      "GroupId": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
      "Data": TRX_DATA,
      "TimeStamp": "1682358865812642130",
      "Version": "2.0.0",
      "Expired": 1682358895812642420,
      "SenderPubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
      "SenderSign": "1vCT4vWFamp6KYLeSf2LqOTA2fyJOkFvF5RZ2EsaWNY8f7m23AeHVnJi1LZSkk3+nBVl3ZhyV9O0sZ5DnoL50AA="
  }
```
- Check consensus now
```
curl -X GET -H 'Content-Type: application/json' -d{} http://127.0.0.1:8002/api/v1/group/cd991ce2-7dbb-422e-9f5c-a0f552e4d896/consensus/
```
Result:
```
{
"producers": [
    {
    "GroupId": "cd991ce2-7dbb-422e-9f5c-a0f552e4d896",
    "ProducerPubkey": "AqdrgpUpRj41BBZmpcfu8VahwZ9IXx8yJl0iaCPW3b7B",
    "ProofTrxId": "a3549534-61f7-428b-8574-f7a7505ea5c1"
    }
],
"trx_epoch_interval": 10000,
"proof_req_id": "a62e45b4-bf47-40b3-b33f-a28bf8407fab",
"curr_epoch": 10003,
"curr_block_id": 1,
"last_update": 1682364196836412069
}
```

- trx_epoch_interval has been changed to 10000
- current epoch is set to 10000
    
- You can try consensus proof related api to get the updated proof
- consensus history will return 2 proofs (original proof and the proof for the update just finished)
- last consensus will return the new proof instead of original proof

### Single producer with multiple user node
- You can add several fullnode as user to the same group and repeat the previous test
- All user node should get the same:
  - consensus info 
  - consensus history/last/req_id should be the same
  - all node should be able to get the CONSENSUS trx with given trx_id

### Multi producers with users
- create full node as o1 
- create several producer nodes as p1, p2... (as group producer)
- create several full node as u1, u2... (as group user)
- o1 create a new group
- all producer nodes join the group
- all user nodes join the group
- verify the group works ok, all apis works ok
- pickup some producrers node to announc itself as group producer
  - verify result
    - group announced producer list is updated 
    - all blocks are created by o1 successul
    - all blocks are saved by all group nodes
    - all trxs are applied
- o1 start change consensus
  try the following test and verify result
  - chagne group producer to p1
  - change group producer to o1, p1, p2
  - change group epoch duration
  - change group epoch number
  - ...

  verify:
  - all selected producers will make agreement to start new consensus
  - all change consensus proof and history are saved by all group nodes
  - all nodes should have same consensus info
  - all change consensus trxs are created and applied

### RexSync test
TBD







