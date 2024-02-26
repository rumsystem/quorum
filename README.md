# RUM: The internet alternatives

### An open source peer-to-peer application infrastructure to offer the internet alternatives in a decentralized and privacy oriented way.

## Summary

The internet has become too centralized, and RUM is trying to find an alternative way to build online applications.

In the RUM system users are organized into groups, and each group will share a blockchain.

Group’s data will be encrypted and sync with all related users as events. All data will be stored in the group-shared-blockchain and eventually be consistent.

Applications can replay the blockchain locally and render the results on the user interface and interact with users.

There are no centralized server providers to store or process data.

![rum-architecture](arch_info_forppt.png)

## Concepts

**Peer**: A user with a pair of keys and connects to the peer-to-peer network.

**Group**: Multi peers or single peer can be organized into groups. A group can represent any online applications, a twitter-like SNS, a reddit-like community or a personal cloud notebook.

**Group Owner**: The creator of a group is the group owner, who will record any valid events transactions in this group and produce new blocks onto old blocks.

Group owners have more privileges, including authorizing other peers (as producers) to produce new blocks, or denying some peers to send events in this group.

**Producer**: Any peers with a public IP address and open ports (including port forwarding/UPnP) can become a producer, who can help the group owner to produce new blocks without reading encrypted events/messages. Producers may receive crypto incentives as reward from the group owner or users, depending on the consensus.

**Event/Message**: Any activity from users is regarded as an event, for example, creating a post, updating avatars, and replying to a post. Event transactions will be broadcast to the group owner and all producers, waiting to be recorded into the new block. RUM uses a subset of Activity Vocabulary Core Types to represent event types.

**Blockchain**: Events transactions will be recorded into blocks, then be linked together to become a blockchain.

**Gossip Network**: There are no centralized servers in the RUM network, the network is only constructed by peers. All event transactions/messages will be passed along to their neighbours peers until target peers receive it eventually.

**Bootstrap**: Bootstrap node is an address book which can help your peers to discover others peers through the DHT-KAD protocol. You can use AddPeer api to add any normal peers which you trust to bootstrap. However, bootstrap node/DHT is not the only way -- a peer will discover others by peer-exchange protocol in the gossip network.

## Why

The internet is broken.

Technology monopoly, especially social network monopoly, has become a worldwide problem. In the United States and the EU, technology companies have caused a lot of disputes over privacy and user data ownership issues. Criticism of traditional Internet platforms has long been a mainstream voice in various countries.

Almost all online services are running on Client-Server architecture, which means all users' activities and data will be stored in centralized servers. User's data always be controlled by service providers not themself, this made the internet centralized by nature.

People urgently need an innovative model that no longer adopts the "privacy for advertising" mode: a new type of network system where data is owned by users and has a special way to distribute traffic without being controlled by the enterprise.

RUM uses a different approach to rebuild the online service. Users will control their own data and interact with other related users in the peer-to-peer network.

## How (video, WIP)

## Ecosystem (WIP)

The [RUM tokens](https://etherscan.io/token/0x72313959c0346016bfba17fa29dcea109f3aa348) can be mined by a non-GPU proof of X algorithm with less energy consuming. After the mainnet launch, the chances of winning the reward is linked to the user's contribution to the network, including the data storage and network traffic.

### Why Do We Need RUM Tokens?

1. Anti-spam

Blocking or blacklists does not work on a decentralized system, so we must use an economic measure to stop abuses and spam. Any RUM network operations will need a small amount of RUM. This is the origin of the Hashcash and POW being invented, as well as the Bitcoin.

2. Resources contribution incentive

A peer-to-peer system must provide economic incentive to minimize the Free-rider problem. Peers share their computing and network resources to help keep the RUM network secure and robust, to get rewards they deserve from the network.

## Getting Started

**TL;DR**:  Try [rum-app](https://rumsystem.net/apps), a cross platform RUM GUI application.

### Build:

<span id="build_quorum"></span>

Build the quorum binary by running the command: `make linux` or  `make buildall`. You can find the binary file in the `dist` dir.

*or*

Build the Docker image by running the command: `sudo docker build -t quorum .`

### Build API Document:

Read deploy version [https://rumsystem.github.io/quorum-api/](https://rumsystem.github.io/quorum-api/)

*or*

Read the [RUM Development Tutorial](./Tutorial.md).

*or*

Running and then open browser with <http://localhost:1323/index.html>.

```sh
# only the first time you build need to run above.
export PATH=$(go env GOPATH)/bin:$PATH
# check GOPATH in PATH,
# if not in run the command above.
make gen-doc
# due to project continue developing sometime it may build failure
# when this happen you can check for a older head or read the deploy version below
make serve-doc
```

### Run a RUM peer

<span id="run_a_peer"></span>

Try [rum-app](https://rumsystem.net/apps), a cross platform RUM GUI application.

*or*

Run the [quorum binary](#build_quorum):

```sh
./quorum fullnode \
    --listen=/ip4/0.0.0.0/tcp/7000 \
    --listen=/ip4/0.0.0.0/tcp/9000/ws \
    --apiport=8000 \
    --peer=/ip4/94.23.17.189/tcp/62777/p2p/16Uiu2HAm5waftP3s4oE1EzGF2SyWeK726P5B8BSgFJqSiz6xScGz \
    --configdir=rum/config \
    --datadir=rum/data \
    --keystoredir=rum/keystore \
    --keystorepwd=yourpassword \
    --loglevel=debug \
    --enabledevnetwork=false 
```

OPTIONS:

```sh
   --listen      required. a multiaddress for the peer service listening
   --apihost     required. http api listening host, a domain or a public ip address
   --peer        required. a bootstrap peer multiaddress. Any online peer could be used for bootstrap, you can use the RUM testing bootstrap server for testing.
   --configdir   optional. a directory for config files. The `peer` of `peerConfig` must same as peername `peer`, eg: if `mypeer2Config`, peername must be `mypeer2`.
   --datadir     optional. all data storage location. The `peer` of `peerData` must same as peername `peer`, eg: if `mypeer2Data`, peername must be `mypeer2`.
   --keystoredir optional. a directory to store private keys. All key files are password protected, and it\'s very important to keep backups of all your keys.
   --keystorepwd optional. password for all keystores. Or using the system environment variable `RUM_KSPASSWD`.
```

See more by running ```./quorum help``` and ```./quorum fullnode help```.

*or*

Run RUM inside Docker:

```
mkdir -p dockerdata/data
mkdir -p dockerdata/certs
mkdir -p dockerdata/config
mkdir -p dockerdata/keystore

docker run --user 1001 \
    -v $(pwd)/dockerdata/data:/data \
    -v $(pwd)/dockerdata/certs:/certs \
    -v $(pwd)/dockerdata/config:/config \
    -v $(pwd)/dockerdata/keystore:/keystore \
    -p 127.0.0.1:8002:8002 \
    -p 8000:8000 \
    -p 8001:8001 \
    -e RUM_KSPASSWD='myverysecretpassword' \
    quorum fullnode --listen /ip4/0.0.0.0/tcp/8000 --listen /ip4/0.0.0.0/tcp/8001/ws  --apiport 8002 --peer /ip4/94.23.17.189/tcp/62777/p2p/16Uiu2HAm5waftP3s4oE1EzGF2SyWeK726P5B8BSgFJqSiz6xScGz 
```

### Example: a private decentralized forum

The main purpose of RUM is to connect groups of people without any centralized server. We start from a simple scenario of a private decentralized forum for a group of friends.

The [rum-app](https://rumsystem.net/apps) will help you create/join/post/view with a nice GUI.

The following shows how to create/join group and post/view content with [quorum binary](#build_quorum) and command line.

1. [Run RUM peer](#run_a_peer) on each friend’s computer, so we have peerA, peerB, peerC...

2. PeerA will create the group, and A will become the owner of the group.

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"ourforum","consensus_type":"poa","encryption_type":"public","app_key":"group_bbs"}' http://127.0.0.1:8000/api/v1/group
```

The response is the group info with group_id and seed-url-string.

```json
{
 "seed": "rum://seed?v=1&e=0&n=0&b=0FzDZxaTRGaMm2h8ojtCAw&c=B10xEenG4-fsa-SRVHHRR-aoBNUQdvkGRHvDVjGXo5Q&g=YznMyEplTMGp6NpP8KMwUw&k=Apot4fO05OsKpEsfzwPlmZTIqsR6QKntRUpCXiEi16rx&s=7cbjKQlDnTzjTAL6YrVMW269B7NC3BE9RiwaZ3GOS_kVHn9SBfPq3KhdxO7_ieretm78tO96-USC9P1LmR1hFgE&t=FwmcaM33XGg&a=ourforum&y=group_bbs&u=http%3A%2F%2F127.0.0.1%3A8000%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI2MzM5Y2NjOC00YTY1LTRjYzEtYTllOC1kYTRmZjBhMzMwNTMiXSwiZXhwIjoxODE3NzA5OTEzLCJuYW1lIjoiYWxsb3ctNjMzOWNjYzgtNGE2NS00Y2MxLWE5ZTgtZGE0ZmYwYTMzMDUzIiwicm9sZSI6Im5vZGUifQ.o1AvtOnkpSIDOJ_tI4By2yU3dCC6tOdW7znurzyFLSg",
 "group_id": "6339ccc8-4a65-4cc1-a9e8-da4ff0a33053"
}
```

> [API: create group](./Tutorial.md#api-create-group)

3. Share the group seed with your friends, so they can join your group with the seed.

4. Join the group with Peer B, C...

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"seed": "rum://seed?v=1&e=0&n=0&b=0FzDZxaTRGaMm2h8ojtCAw&c=B10xEenG4-fsa-SRVHHRR-aoBNUQdvkGRHvDVjGXo5Q&g=YznMyEplTMGp6NpP8KMwUw&k=Apot4fO05OsKpEsfzwPlmZTIqsR6QKntRUpCXiEi16rx&s=7cbjKQlDnTzjTAL6YrVMW269B7NC3BE9RiwaZ3GOS_kVHn9SBfPq3KhdxO7_ieretm78tO96-USC9P1LmR1hFgE&t=FwmcaM33XGg&a=ourforum&y=group_bbs&u=http%3A%2F%2F127.0.0.1%3A8000%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI2MzM5Y2NjOC00YTY1LTRjYzEtYTllOC1kYTRmZjBhMzMwNTMiXSwiZXhwIjoxODE3NzA5OTEzLCJuYW1lIjoiYWxsb3ctNjMzOWNjYzgtNGE2NS00Y2MxLWE5ZTgtZGE0ZmYwYTMzMDUzIiwicm9sZSI6Im5vZGUifQ.o1AvtOnkpSIDOJ_tI4By2yU3dCC6tOdW7znurzyFLSg"}' http://127.0.0.1:8001/api/v2/group/join
```

> [API: join group](./Tutorial.md#api-join-group)

5. Check the group status

```bash
curl http://127.0.0.1:8000/api/v1/groups
```

Response:

```json
{
    "groups": [
        {
            "group_id": "01014e95-303e-4955-b06e-bf185556a729",
            "group_name": "ourforum",
            "owner_pubkey": "CAISIQPAeFZ8rgsENE12HgYwH+3N/aKsRN4fnPEUzEIY7ZyiAQ==",
            "user_pubkey": "CAISIQPAeFZ8rgsENE12HgYwH+3N/aKsRN4fnPEUzEIY7ZyiAQ==",
            "consensus_type": "POA",
            "encryption_type": "PUBLIC",
            "cipher_key": "accb6a4faf34734c418683a9c62bb61209dc79380b69dab20b5042694009dfda",
            "app_key": "group_bbs",
            "last_updated": 1633022375303983600,
            "highest_height": 0,
            "highest_block_id": [
                "989ffea1-083e-46b0-be02-3bad3de7d2e1"
            ],
            "group_status": "IDLE"
        }
    ]
}
```

> [API: groups](./Tutorial.md#api-get-groups)

6. "group_status": "IDLE"  means the group is ready to use. Check the group_status on PeerB, C... make sure group_status is IDLE on every peer.

> [API: start sync](./Tutorial.md#api-post-startsync)

7. It's time to create your first forum post.

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"The Future Will Be Decentralized","name":"My First Post!"},"target":{"id":"01014e95-303e-4955-b06e-bf185556a729","type":"Group"}}' http://127.0.0.1:8000/api/v1/group/content
```

Response:

```json
{
    "trx_id": "0ad70ee3-b6de-4a19-9b3d-f02c037e6a52"
}
```

> [API: post content](./Tutorial.md#api-post-content)

8. Waiting about 10s to sync the blockchain, then check the groups status again.

```bash
curl http://127.0.0.1:8000/api/v1/groups
```

Response:

```json
{
    "groups": [
        {
            "group_id": "01014e95-303e-4955-b06e-bf185556a729",
            "group_name": "ourforum",
            "owner_pubkey": "CAISIQPAeFZ8rgsENE12HgYwH+3N/aKsRN4fnPEUzEIY7ZyiAQ==",
            "user_pubkey": "CAISIQPAeFZ8rgsENE12HgYwH+3N/aKsRN4fnPEUzEIY7ZyiAQ==",
            "consensus_type": "POA",
            "encryption_type": "PUBLIC",
            "cipher_key": "accb6a4faf34734c418683a9c62bb61209dc79380b69dab20b5042694009dfda",
            "app_key": "group_bbs",
            "last_updated": 1633024842663874300,
            "highest_height": 1,
            "highest_block_id": [
                "a835ea5f-ece1-4ba4-94f3-782470dff8c6"
            ],
            "group_status": "IDLE"
        }
    ]
}
```

You will find the group’s highest_height becomes 1, and the highest_block_id also changed.

Check the group_status on PeerB, C ... , All peers should have the same highest_height and highest_block_id which means that all peers have been synchronized successfully.

9. View the posts.

```bash
curl -X POST -H 'Content-Type: application/json' -d '{"senders":[]}' "http://localhost:8000/app/api/v1/group/01014e95-303e-4955-b06e-bf185556a729/content?num=20&reverse=false"
```

Response:

```json
[
    {
        "TrxId": "0ad70ee3-b6de-4a19-9b3d-f02c037e6a52",
        "Publisher": "CAISIQNc7wg3VLZCbKHetaqbZdUro/IUSy33ypWPoI4J24L6gw==",
        "Content": {
            "type": "Note",
            "content": "The Future Will Be Decentralized",
            "name": "My First Post!"
        },
        "TypeUrl": "quorum.pb.Object",
        "TimeStamp": 1633024832659417600
    }
]
```

Congratulations, You have a fully decentralized forum now. Every peer can view the forum posts from their peers.All the data belongs to you and your friends, there is no other service provider or centralized storage.

10. Next:

Add more producers to prevent outages.

> [API: producers](./Tutorial.md#test-producers)

---

### Run a RUM peer on server

1. Build the quorum binary by running the command: `make linux` or  `make buildall`. You can find the binary file in the `dist` dir.
2. Add a shell script to run the peer:

Using the system environment variable `RUM_KSPASSWD` or add the param `keystorepwd`.

```sh
export RUM_KSPASSWD=your_very_secret_password 
```

```bash
# run fullnode of quorum
peername=my_first_peer

./quorum fullnode \
    --keystorepwd=your_very_secret_password \
    --keystoredir=keystore/$peername \
    --configdir=config/$peername \
    --datadir=data/$peername \
    --peer=/ip4/101.42.141.118/tcp/62777/p2p/16Uiu2HAm9uziCEHprbzJoBdG9uktUQSYuFY58eW7o5Dz7rKhRn2j \
    --listen=/ip4/0.0.0.0/tcp/60137 \
    --listen=/ip4/0.0.0.0/tcp/60135/ws \
    --apiport=60136 \
    --log-compress=true \
    --log-max-age=7 \
    --log-max-backups=100 \
    --log-max-size=10 \
    --logfile=logs/$peername/quorum.log \
    --loglevel=debug \
    --enabledevnetwork=false 
```

[view OPTIONS](#run_a_peer)

3. Run the shell script.

Tips: You can use our public bootstrap peer ```/ip4/94.23.17.189/tcp/62777/p2p/16Uiu2HAm5waftP3s4oE1EzGF2SyWeK726P5B8BSgFJqSiz6xScGz``` or any other online peers as bootstrap. 

You can also run a bootstrapnode by using the following command: `./quorum bootstrapnode --help`.
