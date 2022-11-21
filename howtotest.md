# Test environment

role                  name      type
================================================
Bootstrap node      | "bootstrap| bootstrapnode
Owner node          | "owner"   | fullnode
Producer node 1     | "p1"      | producernode
Producer node 2     | "p2"      | producernode
User node 1         | "u1"      | fullnode

#Roles and node type explain

When start a node, a "working mode" should be given, there are 3 types of node
1. bootstrapnode
2. fullnode
3. producernode 

The following rules needs to be followed
1. only fullnode can create a new group
2. producer node can only join group created by other fullnode
3. producer node can only work as a "producer", therefor API like posttogroup is not provided
4. producer node only apply "producer" related trxs, such as appconfig, chainconfig, announce, user, producer, trx with POST type will *NOT* be applied
5. all "normal" group user (user like to use the data and app provided by the group) should run a node as "fullnode" to join the group

A producer node needs to:
1. Start node as a "producernode"
2. Join a group by using group seed
3. Wait block sync finished
4. Self announce as a "producer" by using announce API
5. Wait group owner approve
6. Start work as a producer, handle trx, finish BFT consensus with other producer, build and broadcast new block

What will happen after a fullnode (not for owner) offline and back
1. Start epoch(block) sync with all producers
2. Till get a "BLOCK_NOT_FOUND" response, then finish epoch sync
3. work normally

what will happen after a producer node (include owner) offline and back
1. Start consensus sync with all other producers
2. If consensus sync successful, it means the chain still has enough producers and work normally
3. If needed(chain epoch is large than local epoch), then start epoch sync
4. Till get a "BLOCK_NOT_FOUND" response, then finish epoch sync
5. work normally

#How to test

## Start all 5 nodes

## owner create group 

## p1, p2 and n1 join the Group

## owner or n1 send a POST

## verify a new block is added to ALL nodes and can be get by using block API

## verify trx is ONLY applied at node owner and n1 and can be get by using trx API

## p1 and p2 announce as group producer

e.g 

curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"7c32cde8-bd01-417b-b671-5956ec525fed", "action":"add", "type":"producer", "memo":"producer p1, realiable and cheap, online 24hr"}' http://127.0.0.1:8003/api/v1/group/announce | jq

## verify producers announced 

## owner approve p1 and p2

e.g 
 curl -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":["A8fJLRCgX5ROCbkrhX4bx4yw11Q4yfyQxhCnG_87BJN_","AhCUlCfHYt19mjoyu4W3iMOQAZ2JNJdTo0WR1KWz1QNl"] ,"group_id":"7c32cde8-bd01-417b-b671-5956ec525fed", "action":"add"}' http://127.0.0.1:8002/api/v1/group/producer/false

## owner or n1 send a POST

## verify a new block is added to ALL nodes and can be get by using block API

## verify trx is ONLY applied at node owner and n1 and can be get by using trx API

## owner node offline 

## n1 send a POST

## verify a new block is added to ALL nodes and can be get by using block API

## verify trx is ONLY applied at node owner and n1 and can be get by using trx API

## start owner node again

## owner node should sync the missing block

## verify chain still working

chain should be able to recovery from total failure

## close owner, p1, p2

## Start owner, owenr should start consensus sync 

## Start p1,  p1 should start consensus sync and make agreement with owner and back to idle

## chain should ready for work

## Start p2, p2 should start consensus sync and back to idle

## Chain is fully recovered
