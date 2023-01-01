## There are 3 types you can choose when start a node

### "bootstrapnode"
  Start a bootstrap node, bootstrap node is the old "bootstrap" mode

### "fullnode"
  a typical rum user should create and only create fullnode unless you want to provide resources (network/cpu/storage) as a "producer" for group created by other user
  - a full node can create group and work as the first producer of the group created by this node
  - a full node can join group created by other owner (as a user)
  - a full node can NOT announce itself as a "producer" when join group created by other owner (can only announced as user)
  - a group run on full node will apply ALL trxs received in a produced block
  
### "producernode"
  Start a producer only node, the only propose to create a producer node is to provide withness service(and resources) as a "producer"
  There are some "rules" applied to producer node
  - a producer node can NOT create it own group
  - a producer node can ONLY join group created by other owner (as a producer)
  - a producer node can NOT send "POST" type trx  to a group it joined
  - a producer node can announce itself as a "producer" when join group created by other owner
  - after success announce itself as producer and approved by group owner, a producer node will join the rum consensus process:
      * collect trxs send by other fullnode user in this group
      * sort trx and sign to create a new block (based on epoch)
      * save the block to local db
      * provide block when other node request it (by sync)
  - a producer WILL NOT apply POST type trxs, that means you can NOT get group content from producer node
  - a producer WILL apply all other type trxs

## HBBFT consensus
RUM implement RBC part of the HBBFT protocol.
- N is num of total producer node (owner included)
- f is num of failable producer node (owner included)
- HBBFT request:
   f * 3 < N
  e.g. 2 producers, failable node f = 0 (0 * 3 < 2)
       3 producers, failable node f = 0 (0 * 3 < 3)
       4 producers, failable node f = 1 (1 * 3 < 4)
       ...
       10 producers, failable node f = 3 (3 * 3 < 10)
       ... 
       100 producers, failable node f = 33 (33 * 3 < 100)

## What will happen after a full node create a group
1. the fullnode become "owner" of the group
2. after distrube the "seed" of the group, other full nodes (or producer nodes) can join this group
3. without add other producers, the owner node is the only producer in this group when syncing
4. group owner will collect all trxs send in this group, sign and producer block
5. group owner will broadcast and save the block in local db
6. when asked by other node, group owner will provide the requested block (epoch)

## What will happen after a full node join a group (by using group seed)
1. the full node become "user" of the group
2. user node will start sync with owner(or other nodes)
3. user node will receive block provided by other users in this group
4. user node will check and save the valid block to local db
5. user node will apply all valid trxs in the blocks synced back in a decidated sequence
6. after initial sync finished, user node will receive broadcast for new blocks produced(send by producers)
7. rum user can send variable kinds of trx through API provided by group full node
8. rum user can retrieve blocks/trxs/content/appconfg/... through API provided by group full node

## What will happen after a producer node join a group
1. the producer node become "potential" producer of the group
2. producer node will start sync with owner(or other nodes)
3. producer node will receive block provided by other users in this group when syncing
4. after sync done, a producer node need "announce" itself to owner
5. owner can approve or declain a producer announcement request
6. after approved by the group owner, a producer become an "alive" producer
7. alive producer will join group consensus process and producer block

## What will happen after a fullnode (except owner) offline and back
1. Start epoch(block) sync with all producers
2. Till get a "BLOCK_NOT_FOUND" response, then finish epoch sync
3. work normally

## What will happen after a producer node (owner consider as a producer) offline and back
1. Start consensus sync with all other producers
2. If consensus sync successful, it means the chain still has enough producers and work normally
3. If needed(chain epoch is large than local epoch), then start epoch sync
4. Till get a "BLOCK_NOT_FOUND" response, then finish epoch sync
5. work normally

## number of producers
in current implementation, if there are not enough "alive" or "reachable" producers to finish consensus, the group (chain) is consider "death"
TBD:
owner can send a "sudo" trx (bypass all consensus) to fix the chain (e.g remove all dead producers)

## How to set up test env and test
 Check howtotest.md
