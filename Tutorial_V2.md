## There are 3 types you can choose when start a node

### "bootstrapnode"
  Start a bootstrap node, bootstrap node is the old "bootstrap" mode

### "fullnode"
  A typical rum user should create and only create fullnode unless you want to provide resources (network/cpu/storage) as a "producer" for group created by other user
  - a full node can create group and work as the first producer of the group created by this node
  - a full node can join group created by other owner (as a user)
  - a full node can NOT announce itself as a "producer" when join group created by other owner (can only announced as user)
  - a group run on full node will apply ALL type of trxs received in a produced block
  
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
      * broadcast the new block to all user in this group
  - a producer WILL NOT apply POST type trxs, that means you can NOT get group content from producer node
  - a producer WILL apply all other type trxs

### Block Sync
  RUM use rex sync protocl to get new blocks from other connected nodes, when join a group or back from offline status, a node will start the res sync automatically and ask for any missing blocks from it top block.  All nodes has responsibility to give blocks when asked by other nodes in this group. 

## HBBFT consensus
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
       ... 
       100 producers, failable node f = 33 (33 * 3 < 100)
- HBBFT works on epoch
  all producers will collect trxs send by group user 
  all producers will work together with same epoch to generate consensus
  all producers will "propose" some trxs to this epoch
  HBBFT consensus will generate an "agreement" on which trxs should be packaged to a new block in this round of epoch
  if no trx was proposed in this epoch, next epoch will be carry on
  if some trxs were chosen in this epoch, all producers will generate a new block to package all those trxs

- Update HBBFT config
  owner can change group HBBFT consensus setting
  - owner can assign new producers for this group (by approve some announced producers) 
  - owner can change the epoch duration
  - after new HBBFT config is assign, a "negotiation" process will start among all selected producers 
    - owner will give new epoch to start with
    - a "conculation" should be made by all selected producers, they should sign the contract to agree with the assignment
    - after the new agreement is agree by enough producers, all producers will work by following the new consensus config

## What will happen after a full node create a group
1. this fullnode will "host" the instance of the group just created (memory/storage/network)
2. the user who create this group will become the "owner" of this group
3. the user who create this group will become the first and only producer of this group
4. after distrube the "seed" of the group, other full nodes (or producer nodes) can join this group
5. as the first producer, group owner will collect all trxs in this group, sign and produce block (by using HBBFT)

## What will happen after a full node join a group (by using group seed)
1. the full node become a "user" of the group
2. user node will start rex sync to get all valid blocks and save all received blocks to local db
3. user node will apply all valid trxs in the blocks synced back in a decidated sequence
4. user node provide apis to send new trx (if allowed by owner) to this group, producer will make new block to inclusive this trx
5. user node provide apis to retrieve different kinds of group info like blocks/trxs/content/appconfg/...

## What will happen after a producer node join a group
1. the producer node become "potential" producer of the group
2. producer node will start rex sync to get all blocks 
3. producer node will save all block to local db and only apply producer related trxs
4. producer node need "announce" itself as a "producer" to the group
6. after select by the group owner, a producer become an "alive" producer
7. alive producer will join group consensus process and producer block

<<<<<<< HEAD
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
=======
>>>>>>> consensus_2_main

