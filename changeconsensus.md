# New change consensus procedure
## After create a new group
- The create becomes the "owner" of the group. 
- The owner will announce itself as a "producer"
- The owner will work as the first and the only "producer" of the group

## How producer make consensus
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
- after the HBBFT protocl is finished, a consensus for this Epoch will be made, for example
    {
        Epoch : 100
        Trxs proposed by producer A {TX1, TX2, TX3}
        Trxs proposed by producer B {TX1, TX7, TX8}
        Trxs proposed by producer C {TX4, TX2, TX3}
        Trxs proposed by producer D {"EMPTY"}
    }
- Proofed by HBBFT protocol, all producers will have the same result after consensus is made
- After that ALL active producers will 
    1. organizd their result and make a trxs list to package into a new block
    2. Make new block to package all Trxs, for example 
    {
        BlockId        : 1
        Trxs           : [TX1, TX2, TX3, TX4, TX7, TX8]
        Previsou Block : 0
        ProducerPubkey : {ProducerPubkey}
        Hash           : {hash}
        Sign           : {producer_sign}
    }

    3. broadcast the newly built block to group user
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
        {
            req_id                : Req_ID
            producer_list         : [new_group_producer_list]
            trx_propose_interval  : time interval for each epoch
            agreement_tick_length : time for each round of agreemnet
            agreement_tick_cnt    : retry time
        }
    4. Owner will give a trx_id as return value of the API
    5. After change conseusus finished, owner will send the result as a CONSENSUS type Trx by using this trx_id
    6. Node user can monitor if this trx is packaged successful as an evidence of change consensus finished
    
- when receive req from owner, all producers in request list will create and sign a "proof" 
    1. A proof consist of 2 parts:
        {
            req  : original_req
            resp : response from producer
        }
        a response has the following items
        {
            resp_id        : resp_id
            req            : the original req
            producerPubKey : producer pubkey
            Hash           : hash
            Sign           : producer sign 
        }
- After create proof, all producer start make consensus by using HBBFT
- Within retry cnt, in a agreement time frame, all producer should make consensus of proofs from each other
    {
        owenr_proof,
        producer_1_proof,
        producer_2_proof,
        ...
    }
- If agreement was made, all producer stop current consensus and start next round epoch by using new consensus parameters
- Owner send a CONSENSUS type trx to tell all other nodes in this group consensus config has been changed. 
