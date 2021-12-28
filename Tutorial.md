# RUM Development Tutorial

![Main Test](https://github.com/rumsystem/quorum/actions/workflows/maintest.yml/badge.svg)

<span id="top"></span>

## Menu

Start:

- [Quick Start](#quick-start)
- [Env Prepare](#prepare)
  - [API docs](#docs-api)

You can try:

- [Node](#test-node)
  - [Get node info](#api-get-node)
- [Group](#test-group)
  - [Owner node create a group](#api-post-group)
  - [User node join a group](#api-post-group-join)
  - [List all groups](#api-get-groups)
  - [User node clear a group*](#api-post-group-clear)
  - [User node leave a group](#api-post-group-leave)
  - [Owner node del a group](#api-post-group-del)
- [Network and Sync](#test-network)
  - [Get network info](#api-get-network)
  - [Start sync](#api-post-startsync)
- [Content](#test-content)
  - [Get group content](#api-get-group-content)
    - Get all content
    - Request content with senders filter
  - [Post content to group](#api-post-content)
    - Only text content
    - Content with images
    - Like/Dislike
  - [Update user profile of a group](#api-group-profile) 
- [Block](#test-block)
  - [Get block info](#api-get-block)
- [Trx](#test-trx)
  - [About trx](#about-trx)
  - [Get trx info](#api-get-trx)
- [Producers](#test-producers)
  - [About producers](#about-producers)
  - [Announce producer](#api-post-announce-producer)
  - [Get announced producers](#api-get-announced-producers)
  - [Add producer](#api-post-producer-add)
  - [Get producers](#api-get-producers)
  - [Owner remove producer](#api-post-producer-remove)
- [DeniedList](#test-deniedlist)
  - [Get deniedlist](#api-get-deniedlist)
  - [Add deniedlist](#api-post-deniedlist-add)
  - [Del deniedlist](#api-post-deniedlist-del)
- [Group Config](#test-group-config)
  - [Add group config](#api-post-group-config-add)
  - [Get group config keylist](#api-get-group-config-keylist)
  - [Get group config keyname](#api-get-group-config-keyname)
  - [Add group schema](#api-post-group-schema)
  - [Get group schema](#api-get-group-schema)
- [Private Group](#test-private-group)
  - [Announce user](#api-post-announce-user)
  - [Get announced users](#api-get-announced-users)
  - [Owner approve a user](#api-post-group-user)

Common params:

- [Params](#param-list)
  - [group_id](#param-group_id)
  - [group_name](#param-group_name)
  - [trx_id](#param-trx_id)
  - [block_id](#param-block_id)
  - [node_id](#param-node_id)
  - [peer_id](#param-peer_id)
  - [owner_pubkey](#param-owner_pubkey)
  - [user_pubkey](#param-user_pubkey)
  - [group_status](#param-group_status)
  - [app_key](#param-app_key)

<span id="quick-start"></span>

# Quick Start

1. 安装 RUM

[下载安装包](https://docs.prsdev.club/#/rum-app/test)

[自行编译最新包](https://github.com/rumsystem/quorum)

2.、启用 RUM 服务

3. 采用你擅长的语言，与 RUM 服务建立连接，

```python
"""
:param PORT: int，本地已经启动的 Rum 服务的 端口号，该端口号基本上是固定的，不会变动
:param HOST: str，本地已经启动的 Rum 服务的 host，通常是 127.0.0.1
:param CACERT: str，本地 Rum 的 server.crt 文件的绝对路径
"""

import requests

url = f"https://{HOST}:{PORT}/api/v1"
session = requests.Session()
session.verify = CACERT
session.headers.update({
    "USER-AGENT": "asiagirls-py-bot",
    "Content-Type": "application/json"})
session.get(f"{url}/node")

```

请在您自行创建的种子网络中测试，不要往公开的种子网络中发布测试信息。

[>>> back top](#top)

<span id="prepare"></span>

# Env Prepare

## Run testing

```go test cmd/main* -v```

## Generate API Docs

```go run cmd/docs.go```

Open url ```http://localhost:1323/swagger/index.html``` in the browser.

<span id="docs-api"></span>

[>>> back top](#top)

## Setup local test network

1. Download and install go (ver 1.15.2)

2. Clone quorum project from ```https://github.com/rumsystem/quorum.git```

3. 3 local nodes will be created

- `bootstrap node` (bootstrap)
- `owner node`   (group owner node)
- `user node`    (group user node)

follow steps below.

4. cd to quorum souce code path and create `config/` dir

```bash
mkdir -p config
```

5. start `bootstrap node`

```bash
go run cmd/main.go -bootstrap -listen /ip4/0.0.0.0/tcp/10666 -logtostderr=true
```

output:

```bash
I0420 14:58:47.719592     332 keys.go:47] Load keys from config
I0420 14:58:47.781916     332 main.go:64] Host created, ID:<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR>, Address:<[/ip4/172.28.230.210/tcp/10666 /ip4/127.0.0.1/tcp/10666]>
```

Record `<HOST_ID>`, for example:
`QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR`

5. Start `owner node`

```bash
go run cmd/main.go -peername owner -listen /ip4/127.0.0.1/tcp/7002 -apilisten :8002 -peer /ip4/127.0.0.1/tcp/10666/p2p/<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR> -configdir config -datadir data -keystoredir ownerkeystore  -jsontracer ownertracer.json -debug true
```

- For the first time, user will be asked to input a password for the node, if not given, a password will be created for the node
- After a password is created, next time user will be asked to input the password to open node.
- env RUM_KSPASSWD can be used to input node password, like:

```bash
RUM_KSPASSWD=<node_passwor> go run cmd/main.go...
```

6. Start `user node`

```bash
go run cmd/main.go -peername user -listen /ip4/127.0.0.1/tcp/7003 -apilisten :8003 -peer /ip4/127.0.0.1/tcp/10666/p2p/<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR> -configdir config -datadir data -keystoredir ownerkeystore  -jsontracer usertracer.json -debug true
```

[>>> back top](#top)

<span id="test-node"></span>

# Node

## Get Node Info

<span id="api-get-node"></span>

**API**: ```*/api/v1/node```

- Method: GET
- Usage : get node info
- Params : none

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8003/api/v1/node
```

API return value:

```json
{
    "node_id": "16Uiu2HAkytdk8dhP8Z1JWvsM7qYPSLpHxLCfEWkSomqn7Tj6iC2d",
    "node_publickey": "CAISIQJCVubdxsT/FKvnBT9r68W4Nmh0/2it7KY+dA7x25NtYg==",
    "node_status": "NODE_ONLINE",
    "node_type": "peer",
    "node_version": "1.0.0 - 99bbd8e65105c72b5ca57e94ae5be117eaf05f0d",
    "peers": {
        "/quorum/nevis/meshsub/1.1.0": [
            "16Uiu2HAmM4jFjs5EjakvGgJkHS6Lg9jS6miNYPgJ3pMUvXGWXeTc"
        ]
    }
}
```

| Param | Description |
| --- | --- |
| "node_publickey" | 组创建者的 pubkey |
| "node_status" | "NODE_ONLINE" or "NODE_OFFLINE" |
| "node_version" | 节点的协议版本号 |
| "peers" | dict |

[>>> back top](#top)

<span id="test-group"></span>

# Group

<span id="api-post-group"></span>

## Owner node create a group

**API**: ```*/api/v1/group```

- Method: POST
- Usage : Owner create a group
- Params:
    - group_name
    - [consensus_type](#param-consensus_type)
    - [encryption_type](#param-encryption_type)
    - app_key

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' https://127.0.0.1:8002/api/v1/group
```

```json
{
    "group_name": "my_test_group",
    "consensus_type": "poa",
    "encryption_type": "public",
    "app_key": "test_app"
}
```

API return value:

```json
{
    "genesis_block": {
        "BlockId": "80e3dbd6-24de-46cd-9290-ed2ae93ec3ac",
        "GroupId": "c0020941-e648-40c9-92dc-682645acd17e",
        "ProducerPubKey": "CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg==",
        "Hash": "LOZa0CLITIpuQqpvXb6LyXV9z+2rSoU4JwBq0BCXttc=",
        "Signature": "MEQCICAXCicQ6f4hRNSoJR89DF3a6AKpe6ZgLXsjXqH9H3jxAiA8dpukcriwEu8amouh2ZEKA2peXr3ctKQwxI3R6+nrfg==",
        "Timestamp": 1632503907836381400
    },
    "group_id": "c0020941-e648-40c9-92dc-682645acd17e",
    "group_name": "my_test_group",
    "owner_pubkey": "CAISIQLW2nWw+IhoJbTUmoq2ioT5plvvw/QmSeK2uBy090/3hg==",
    "owner_encryptpubkey": "age18kngxt6lkxqulldvxu8xs2ey77rrzwjhqpdey527ad4gkn3euu9sj3ah5j",
    "consensus_type": "poa",
    "encryption_type": "public",
    "cipher_key": "8e9bd83f84cf1408484d24f486861947a1db3fbe6eb3c61e31af55a4803aedc1",
    "app_key": "test_app",
    "signature": "304502206897c3c67247cba2e8d5991501b3fd471fcca06f15915efdcd814b9e99c9a48a022100aa3024eb5663da6cbbde150132a4ff52c6c6aeeb49e0c039b4c28e72b071382f"
}
```

Params:

* genesis_block       //genesis block, the first block of the group
* owner_encryptpubkey //owner encryption key(age)
* [consensus_type](#param-consensus_type)
* [encryption_type](#param-encryption_type)
* cipher_key          //aes key <sup>[1]</sup>
* signature           //owner signature

<sup>[1]</sup> neglect group encryption type (`public` or `private`), all trx except "POST" will be encrypted by `cipher_key`

returned json string from API call is the "`seed`" of the newly created `group`.

other nodes can use the seed to [join the group](#api-post-group-join).

[>>> back top](#top)

<span id="api-post-group-join"></span>

## User node join a group

**API**: ```*/api/v1/group/join```

- Method: GET
- Usage : User node join a group
- Params: the `seed` of `group` above

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"genesis_block":{"BlockId":"36ac6e22-80a1-4d54-abbb-8bd2c55ef8cf","GroupId":"eae3f0db-a034-4c5f-a25f-b1177390ec4d","ProducerPubKey":"CAISIQMJIG4do9g8PBixH432YXVQmD7Ilqp7DzbGxgLJHbRoFA==","Hash":"fDGwAPJbHHG0GpKLQZnRolK9FUO5nSIod/iprwQQn8g=","Signature":"MEYCIQDo5uge+saujb0WR6ZreISDYWpRzY6PQ3f5ly7vtHHgkQIhAKcuwDT2fIpBDx/7lQU6mIBQKJuQeI0Zbw3W7kHfBO28","Timestamp":1631804384241781200},"group_id":"eae3f0db-a034-4c5f-a25f-b1177390ec4d","group_name":"my_test_group","owner_pubkey":"CAISIQMJIG4do9g8PBixH432YXVQmD7Ilqp7DzbGxgLJHbRoFA==","owner_encryptpubkey":"age1lx3zh5sc5cureh484t5tm2036lhrzdnh96rfaft6echs9cqsefss4yn886","consensus_type":"poa","encryption_type":"public","cipher_key":"3994c4224da17ad50504c78458f37249149477c7bc643f3fe78e44033c17874a","signature":"30450220591361918948140c8ad1736cde3831f326470f2d3c5105a0b63867c7b216857c0221008921422c6e1974834d5610d4c6ad1a9dd0394ac464dfc12659cde41d75172d14"}' https://127.0.0.1:8003/api/v1/group/join
```

API return value:

```json
{
    "group_id": "ac0eea7c-2f3c-4c67-80b3-136e46b924a8",
    "group_name": "my_test_group",
    "owner_pubkey": "CAISIQOeAkTcYYWVTSH80dl2edMA4kI27g9/C6WAnTR01Ae+Pw==",
    "user_pubkey": "CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA==",
    "user_encryptpubkey": "age1774tul0j5wy5y39saeg6enyst4gru2dwp7sjwgd4w9ahl6fkusxq3f8dcm",
    "consensus_type": "poa",
    "encryption_type": "public",
    "cipher_key": "076a3cee50f3951744fbe6d973a853171139689fb48554b89f7765c0c6cbf15a",
    "signature": "3045022100a819a627237e0bb0de1e69e3b29119efbf8677173f7e4d3a20830fc366c5bfd702200ad71e34b53da3ac5bcf3f8a46f1964b058ef36c2687d3b8effe4baec2acd2a6"
}
```

API return value:

- "user_encryptpubkey" 本节点在组内的加密公钥 ** 
- [consensus_type](#param-consensus_type) 
- [encryption_type](#param-encryption_type)
- "cipher_key"  组内协议对称加密密钥(aes) 
- "signature"  signature by group owner 

* [如果组类型为 PRIVATE，则该加密公钥需要用其他协议进行组内广播](#test-private-group)

节点 B 加入组后，开始自动同步(SYNCING)，同步完成后状态变为（IDLE)

[>>> back top](#top)

<span id="api-get-groups"></span>

## List all groups

**API**: ```*/api/v1/groups```

- Method: GET
- Usage : List all groups
- Params : none

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/groups
```

- Method: GET
- Params: none

API return value:

```json
{
    "groups": [
        {
            "group_id": "90387012-431e-495e-b0a1-8d8060f6a296",
            "group_name": "my_test_group",
            "owner_pubkey": "CAISIQP67zriZHvC+OWv1X8QzFIwm8CKIM+5KRx1FsUSHQoKxg==",
            "user_pubkey": "CAISIQP67zriZHvC+OWv1X8QzFIwm8CKIM+5KRx1FsUSHQoKxg==",
            "consensus_type": "POA",
            "encryption_type": "PUBLIC",
            "cipher_key": "f4ee312ef7331a2897b547da0387d56a7fe3ea5796e0b628f892786d1e7ec15d",
            "app_key": "test_app",
            "last_updated": 1631725187659332400,
            "highest_height": 0,
            "highest_block_id": "a865ae03-d8ce-40fc-abf6-ea6f6132c35a",
            "group_status": "IDLE"
        }
    ]
}
```

- Params:
    * [consensus_type](#param-consensus_type) 
    * [encryption_type](#param-encryption_type)
    * cipher_key
    * last_updated
    * highest_height      <sup>[2]</sup>
    * highest_block_id    <sup>[3]</sup>

<sup>[2]</sup> Heighty of the "highest" block in this group

<sup>[3]</sup> block_id of the "highest" block in this group

[>>> back top](#top)

<span id="api-post-group-clear"></span>

## User node clear a group *

**删除组 API 已废除，所有节点只能“离开”一个组，不管是不是自己创建的**

**API**: ```*/api/v1/group/clear```

- Method: POST
- Usage : User node clear a group
- Params :
  - [group_id](#param-group_id)

删除一个组的全部内容，包括如下内容

    - block
    - trx
    - announced
    - scheam
    - denied_list
    - post
    - producer

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"13a25432-b791-4d17-a52f-f69266fc3f18"}' https://127.0.0.1:8002/api/v1/group/clear | jq
```

API return value:

```json
{
    "group_id": "13a25432-b791-4d17-a52f-f69266fc3f18",
    "signature": "30450221009634af1636bf7374453cd73088ff992d9020777eb617795e3c93ea5d5008f56d022035342a852e87afa87b5e038147dedf10bb847f60808ec78a470b92dfbff91504"
}
```

**目前前端在离开组时需一起调用该 API，清除所有组相关的数据，警告用户“如果离开组，所有数据将被删除，再加入需要重新同步”即可**

[>>> back top](#top)

<span id="api-post-group-leave"></span>

## User node leave a group

**API**: ```*/api/v1/group/leave```

- Method: POST
- Usage : User node leave a group
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' https://127.0.0.1:8002/api/v1/group/leave
```

**Params**:

```json
{
    "group_id": "846011a8-1c58-4a35-b70f-83195c3bc2e8"
}
```

API return value:

```json
{
    "group_id": "846011a8-1c58-4a35-b70f-83195c3bc2e8",
    "signature": "304402201818acb8f1358b65aecd0343a48f0fe79c89c3f2852fa809dd6b9315a20740e4022026d0ca3b981ee2a3701930b62d7f5ddcf959a3ba50d926c31f6c143ef91f024a"
}
```

| Param | Description |
| --- | --- |
| "signature" | signature by group owner |

[>>> back top](#top)

<span id="api-post-group-del"></span>

## Owner node del a group

**API**:  ```*/api/v1/group```

- Method: DELETE
- Usage : Owner node del a group
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
 curl -k -X DELETE -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' https://127.0.0.1:8003/api/v1/group

```

API return value:

```json
{
    "group_id": "846011a8-1c58-4a35-b70f-83195c3bc2e8",
    "owner_pubkey": "CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB",
    "signature": "owner_signature"
}
```

[>>> back top](#top)

<span id="test-network"></span>

# Network and Sync

<span id="api-get-network"></span>

## Get network info

**API**:  ```*/api/v1/network```

- Method: GET
- Usage : Get network info
- Params : none

**Example**:

```bash
curl -k http://localhost:8002/api/v1/network
```

API return value:

```json
{
    "groups": [
        {
            "GroupId": "997ce496-661b-457b-8c6a-f57f6d9862d0",
            "GroupName": "pb_group_1",
            "Peers": [
                "16Uiu2HAkuXLC2hZTRbWToCNztyWB39KDi8g66ou3YrSzeTbsWsFG"
            ]
        }
    ],
    "node": {
        "addrs": [
            "/ip4/192.168.20.17/tcp/7002",
            "/ip4/127.0.0.1/tcp/7002",
            "/ip4/107.159.4.35/tcp/65185"
        ],
        "ethaddr": "0x4daD72e78c3537a8852ca7b3d1742Dd42c30441A",
        "nat_enabled": true,
        "nat_type": "Public",
        "peerid": "16Uiu2HAm8XVpfQrJYaeL7XtrHC3FvfKt2QW7P8R3MBenYyHxu8Kk"
    }
}
```

这里需要注意， nat_type 和 addrs 都会改变，开始的时候没有公网地址，类型是 Unknown 之后会变成 Private，再过一段时间反向链接成功的话，就变成 Public，同时 Addrs 里面出现公网地址。

[>>> back top](#top)

<span id="api-post-startsync"></span>

## Start sync

**API**: ```*/api/v1/group/{group_id}/startsync```

- Method: POST
- Usage : Start sync for a group
- Params :
  - [group_id](#param-group_id)

客户端可以手动触发某个组和组内其他节点同步块

**Example**:

```bash
curl -X POST -H 'Content-Type: application/json' -d '' http://<IP_ADDR>/api/v1/group/<GROUP_ID>/startsync
```

API return value:

| status_code | result | Description|
| --- | --- | --- |
| 200 | ```{"GroupId":<GROUP_ID>,"Error":""}```| GROUP_ID 的组正常开始同步，同时组的状态会变为 SYNCING|
| 400 | ```{"error":"GROUP_ALREADY_IN_SYNCING"}```| GROUP_ID 的组当前正在同步中|

[>>> back top](#top)

<span id="test-content"></span>

# Content

<span id="api-get-group-content"></span>

## Get group content

**API**: ```*/api/v1/group/{group_id}/content```

- Method: GET
- Usage : Get content of a group, all or with senders filter
- Params :
  - [group_id](#param-group_id)

**Example**:

#### Requst all content

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8003/api/v1/group/c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55/content
```

#### Request content with senders filter

```bash
curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[ "CAISIQP8dKlMcBXzqKrnQSDLiSGWH+bRsUCmzX42D9F41CPzag=="]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20"
```

```json
{
    "start": 0,
    "num": 10
}
```

API return value: a list of trxs. 

Note, Person-profile or Like/Dislike Trx.

Note Trx:

- "type" of "Content": "Note"
- "TypeUrl": "quorum.pb.Object"

```json
{
    "TrxId": "da2aaf30-39a8-4fe4-a0a0-44ceb71ac013",
    "Publisher": "CAISIQOlA37+ghb05D5ZAKExjsto/H7eeCmkagcZ+BY/pjSOKw==",
    "Content": {
        "type": "Note",
        "content": "simple note by aa",
        "name": "A simple Node id1"
    },
    "TypeUrl": "quorum.pb.Object",
    "TimeStamp": 1629748212762123400
}
```

Person Profile Trx:

- "type" of "Content": any of "name"(string),"image"(dict) or "wallet"(list)
- "TypeUrl": "quorum.pb.Person"

```json
{
    "TrxId": "7d5e4f23-42c5-4466-9ae3-ce701dfff2ec",
    "Publisher": "CAISIQNK024r4gdSjIK3HoQlPbmIhDNqElIoL/6nQiYFv3rTtw==",
    "Content": {
        "name": "Lucy",
        "image": {
            "mediaType": "image/png",
            "content": "there will be bytes content of images、"
        },
        "wallet": [
            {
                "id": "bae95683-eabb-212f-9588-12dadffd0323",
                "type": "mixin",
                "name": "mixin messenger"
            }
        ]
    },
    "TypeUrl": "quorum.pb.Person",
    "TimeStamp": 1637574058426424900
}
```

Like/Dislike Trx:

- "type" of "Content": "Like" or "Dislike"
- "TypeUrl": "quorum.pb.Object"

```json

{
    "TrxId": "65de2397-2f35-4a07-9af2-35a920b79882",
    "Publisher": "CAISIQMbTGdEDACml0BOcBXpWM6FOLDgH7u9VapHJ+wDMZSObw==",
    "Content": {
        "id": "02c23edc-be7d-4a32-bbae-fb8e179e9c5b",
        "type": "Like"
    },
    "TypeUrl": "quorum.pb.Object",
    "TimeStamp": 1639980884426949600
}
```

Params
- "TrxId",[trx_id](#param-trx_id) 
- "Publisher" ,发布者 
- "Content", dict, 内容 
- "TypeURL", string, Type 
- "TimeStamp" ,int64 

[>>> back top](#top)

<span id="api-post-content"></span>

## Post content to group

nodeA can be `owner node` or `user node`.

**API**: ```*/api/v1/group/content```

- Method: POST
- Usage : Post content to group
- Params :
  - type
  - object
  - target
- API return value:
  - [trx_id](#param-trx_id)

**Example**:

### Note only content:

```bash
curl -k -X POST -H 'Content-Type: application/json'  -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55","type":"Group"}}'  https://127.0.0.1:8002/api/v1/group/content
```

```json
{
    "type": "Add",
    "object": {
        "type": "Note",
        "content": "Good Morning!\nHave a nice day.",
        "name": ""
    },
    "target": {
        "id": "c60ed78e-df15-4408-9b5b-f87158cf0bda",
        "type": "Group"
    }
}
```

### content with images

1~4 images , total size less than 200 mb

```json
{
    "type": "Add",
    "object": {
        "type": "Note",
        "content": "Good Morning!\nHave a nice day.",
        "name": "",
        "image": [
            {
                "mediaType": "image/png",
                "content": "this is pic content by base64.b64encode(pic-content-bytes).decode(\"utf-8\")",
                "name": "pic-name init by uuid like f\"{uuid.uuid4()}-{datetime.now().isoformat()}\""
            }
        ]
    },
    "target": {
        "id": "d87b93a3-a537-473c-8445-609157f8dab0",
        "type": "Group"
    }
}
```

[>>> back top](#top)

### Like/Dislike

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Like","object":{"id":"578e65d0-9b61-4937-8e7c-f00e2b262753"}, "target":{"id":"c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/content
```

**Params**:

- "id" in "object" is [trx_id](#param-trx_id)

```json
{
    "type": "Like",
    "object": {
        "id": "578e65d0-9b61-4937-8e7c-f00e2b262753"
    },
    "target": {
        "id": "c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55",
        "type": "Group"
    }
}
```

[>>> back top](#top)

<span id="api-group-profile"></span>

## Update user profile of a group

any group has its own profile to set.

**API**: ```*/api/v1/group/profile```

- Method: POST
- Usage : update profile of a group
- Params :
  - type
  - object
  - target
- API return value:
  - [trx_id](#param-trx_id)

**Example**:


**Params**:

```json
{
    "type": "Update",
    "person": {
        "name": "nickname",
        "image": {
            "mediaType": "image/png",
            "content": "there will be bytes content of images"
        },
        "wallet": [
            {
                "id": "bae95683-eabb-211f-9588-12dadffd0323",
                "type": "mixin",
                "name": "mixin messenger"
            }
        ]
    },
    "target": {
        "id": "c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55",
        "type": "Group"
    }
}
```

[>>> back top](#top)

<span id="test-block"></span>

# Block

<span id="api-get-block"></span>

## Get block info

**API**: ```*/api/v1/block/{group_id}/{block_id}```

- Method: GET
- Usage : Get block info
- Params :
  - [group_id](#param-group_id)
  - [block_id](#param-block_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8003/api/v1/block/<GROUP_ID>/<BLOCK_ID>
```

API return value:

```json
{
    "BlockId": "aa75447f-b621-4424-a723-9d4bf1d9fff9",
    "GroupId": "a4b634c2-ceb7-4e60-9584-a221aa7b6855",
    "PrevBlockId": "78bffd23-2dba-408b-b88e-ed3f5f005411",
    "PreviousHash": "ZXh7C2Fnp4J8ny96Udo2Nr3Z50zu+KdA4BcEiw7cF4s=",
    "Trxs": [
        {
            "TrxId": "820d6b65-99b8-4b96-afb1-0b639a76e1f3",
            "GroupId": "a4b634c2-ceb7-4e60-9584-a221aa7b6855",
            "Data": "CiR0eXBlLmdvb2dsZWFwaXMuY29tL3F1b3J1bS5wYi5PYmplY3QSKxIETm90ZTIRc2ltcGxlIG5vdGUgYnkgYWFCEEEgc2ltcGxlIE5vZGUgaWQ=",
            "TimeStamp": 1631817240704625000,
            "Version": "ver 0.01",
            "Expired": 1631817540704625200,
            "SenderPubkey": "CAISIQJHvBByFpoeT6SBvE+w3FTs5zRTq19hi7GP0fTVkj00hw==",
            "SenderSign": "MEUCIBwTg4UzSub5IUl4NVEZmMmkG8Kx2XMZCHIThoLdAtBoAiEAoCM5f/vYbUVIqdgS40vVueb954duzIjrzMDzHmE8h6s="
        }
    ],
    "ProducerPubKey": "CAISIQJHvBByFpoeT6SBvE+w3FTs5zRTq19hi7GP0fTVkj00hw==",
    "Hash": "RnChfYe3rBsO5swKoSDV5K8spV+NL5kaJ3aH1w/73lU=",
    "Signature": "MEYCIQC9Rnj381tjLmo8XwW0kpOCQb5o62QN78L4a6QsXIA37gIhALVClUs9UB32f7wQTUmoVg58uLr6r3apGkNyKh1uek4i",
    "Timestamp": 1631817245705639200
}
```

"Trxs" is a list. one block can have a number of trxs.

[>>> back top](#top)

<span id="test-trx"></span>

# Trx

<span id="about-trx"></span>

## About Trx

Trx 生命周期，加密和出块过程

**Trx 种类**

所有链上操作均是 Trx，客户端相关的 Trx 有 5 种

|Type|Description |More about|
|---|---|---|
| POST | user 发送组内信息(POST Object)|[Post content](#api-post-content)|
| ANNOUNCE | user 宣布自己在组内的公钥|[Anounce user](#api-post-announce-user)|
| AUTH | Owner 调整组内权限|
| SCHEMA | Owner 管理组内数据 schema|[Group config](#test-group-config)|
| PRODUCER|  Owner 管理组内 producer|[Producers](#test-producers)|

<span id="about-trx"></span>

**Trx 加密类型**

为了确保后加入的用户能正确使用组内功能，根据 trx 类型，进行如下形式的加密：

- POST trx
  - 强加密组： 每个发送节点都要根据自己手中的组内成员名单（公钥名单），对 POST 的内容进行非对称加密，然后发送，收到 trx 的节点使用自己的公钥对 trx 进行解密 [private group](#test-private-group)
  - 弱加密组： 每个发送节点都用 seed 中的对称加密字串对收发的 trx 数据进行对称加密

- other trx
  - 所有其他的链相关的协议均使用弱加密组策略（用 seed 中的对称加密字串进行收发）

**出块流程/共识策略**

一个 Trx 被 push 到链上后，根据 group 的不同共识类型，将被采取不同形式对待。

[consensus type](#param-consensus_type)

**Trx 状态判断**

同其他链相似，Trx 的发送没有重试机制，客户端应自己保存并判断一个 Trx 的状态，具体过程如下

1. [发送一个 trx ](#api-post-content)时，获取 [trx_id](#param-trx_id)

2. 将这个 trx 标记为“发送中”

3. [查询组内的内容](#api-get-group-content)

4. 设置一个超时，目前建议是 30 秒，不断查询，直到相同 [trx_id](#param-trx_id) 的内容出现在返回结果中，即可认为 trx 发送成功（被包含在块中）

5. 如果超时被触发，没有查到结果，即认为发送 trx 失败，客户端可以自行处理重发

* AUTH 相关的 trx 处理方式相同（[黑名单](#test-deniedlist)）

[>>> back top](#top)

<span id="api-get-trx"></span>

## Get trx info

**API**: ```*/api/v1/trx/{group_id}/{trx_id}```

- Method: GET
- Usage : Get trx info
- Params :
  - [group_id](#param-group_id)
  - [trx_id](#param-trx_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d https://127.0.0.1:8003/api/v1/trx/<GROUP_ID>/<TRX_ID>
```

API return value:

```json
{
    "TrxId": "c63d7c8e-d56d-432c-aae3-7d0d9dc34c31",
    "GroupId": "3bb7a3be-d145-44af-94cf-e64b992ff8f0",
    "Data": "rx5hmlGgIgnQSm5tT75KY96UaIauDALPvPLjRRe2qiwJhc8VI3wwpsm2M3Y4bYCXGhpjWVDc3D5pHr+cnhuUqWZWQUZJ8FkGYG+bHnz0t4z2//6xo+3+GrCogphT+vJHPCld3womShSLEo4G3VTBbBzaPOnSg1T31OuI8wRsKoslI1owKiWC4r5VwhXHmLq8RW+HFpIy7PqZXxr+8Hsojawrs0B9CbJ3wf7TWubUlw5JhpAXGbbBBw6nLyGM7MnL0+Q3nUi1mX9dgGWOEwwxvO66SYhB",
    "TimeStamp": "1639570707554262200",
    "Version": "1.0.0",
    "Expired": 1639571007554262200,
    "SenderPubkey": "CAISIQKwLxW1uBoZHMbss9QTdVLb8lfBhvMQ3ucnm9afGnVmpQ==",
    "SenderSign": "MEQCIGKc0MyiusNFWZEc+ZMXzk/eev7Sdouii4zAeSIGCqnMAiAz+LMXWck1NIJLB8U7mGmetzYGuTYPKxifH7sF1cMwZg=="
}
```

* "Data" 是加密的，([encryption type](#param-encryption_type)由组类型决定)
* 客户端应通过[获取 Content 的 API](#api-get-group-content) 来获取解密之后的内容

[>>> back top](#top)

<span id="test-producers"></span>

# Producers

<span id="about-producers"></span>

## About producers

Producer 作为组内“生产者”存在，可以代替 Owner 出块，组内有其他 Producer 之后，Owenr 可以不用保持随时在线，在 Owner 下线的时间内，Producer 将代替 Owner 执行收集 Trx 并出块的任务

关于 Producer 的内容，如具体的共识算法、Producer 的收益等内容，请参考 RUM 设计文档

Owner 作为组内第一个 Producer 存在，有其它 Producer 存在时，如果 Owner 在线，也将作为一个 Producer 存在

有 Producer 存在的流程如下：

1. [Owner 创建组](#api-post-group)

2. Owner 作为 Producer 存在，负责出块

3. 其他 Producer 获得组的 seed，[加入组](#api-post-group-join)，完成同步

4. Producer 用[Announce API](#api-post-announce)将自己作为 Producer 的意愿告知 Owner

5. 其他节点（包括 Owner 节点）[查看所有 Announce 过的 Producer](#api-get-announced-producers)

6. [Owner 批准某个 producer](#api-post-producer-add)

* 请注意，Owner 只可以选择在组内[Announce 过自己的 Producer](#api-post-announce-producer)，并且 producer 的状态应该为“ADD”，没有 Announce 过的 producer 是不可以添加的

7. [查看组内目前的实际批准的 producers](#api-get-announced-producers)

8. [查看 Announce Producer 状态](#api-get-announced-producers)，可以看出，经过 Owner 批准，该 Producer 的状态（result)变为 APPROVED

9. [Owenr 删除组内 Producer](#api-post-producer-remove)
    * Owner 可以随时删除一个 Producer, 不管 Producer 是否 Announce 离开
    * 在实际环境中，Producer 完全可以不 Announce Remove 而直接离开，Owner 需要注意到并及时将该 Producer 从 Producer 列表中删除

[>>> back top](#top)

<span id="api-post-announce-producer"></span>

## Announce producer

**API**: ```*/api/v1/group/announce```

- Method: POST
- Usage : Announce producer
- Params :
  - [group_id](#param-group_id)
  - action
  - type
  - memo

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add", "type":"producer", "memo":"producer p1, realiable and cheap, online 24hr"}' https://127.0.0.1:8005/api/v1/group/announce | jq
```

**Params**:

```json
{
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "action": "add",
    "type": "producer",
    "memo": "producer p1, realiable and cheap, online 24hr"
}
```

| Param | Description |
| --- | --- |
| "action" | add or remove |
| "type" | string | producer |
| "memo" | memo |

API return value:

```json
{
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "sign_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
    "encrypt_pubkey": "",
    "type": "AS_PRODUCER",
    "action": "ADD",
    "sign": "3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5",
    "trx_id": "2e86c7fb-908e-4528-8f87-d3548e0137ab"
}
```

| Param | Description |
| --- | --- |
| "sign_pubkey" | producer 在本组的签名 pubkey |
| "encrypt_pubkey" | 没有使用 |
| "type" | AS_PRODUCER |
| "action" | ADD |
| "sign" | producer 的签名 |

[>>> back top](#top)

<span id="api-get-announced-producers"></span>

## Get announced producers

**API**: ```*/api/v1/group/{group_id}/announced/producers```

- Method: GET
- Usage : Get announced producers
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8002/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/announced/producers
```

API return value:

```json
[
    {
        "AnnouncedPubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
        "AnnouncerSign": "3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5",
        "Result": "ANNOUCNED",
        "Action": "Add",
        "TimeStamp": 1634756064250457600
    }
]
```

* ACTION 可以有 2 种状态，“ADD”表示 Producer 正常，“REMOVE”表示 Producer 已经 announce 自己离开改组

```json
[
    {
        "AnnouncedPubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
        "AnnouncerSign": "3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5",
        "Result": "APPROVED",
        "Action": "ADD",
        "TimeStamp": 1634756064250457600
    }
]
```

**Params**:

| Param | Description |
| --- | --- |
| "AnnouncedPubkey" | producer pubkey |
| "AnnouncerSign" | producer 的签名 |
| "Result" | ANNOUNCED or APPROVED，producer 刚 Announce 完毕的状态是 ANNOUNCED |
| "Action" | "ADD" or "REMOVE" |
| "TimeStamp" | timestamp |

[>>> back top](#top)

<span id="api-post-producer-add"></span>

## Add producer

**API**: ```*/api/v1/group/producer```

- Method: POST
- Usage : Add producer
- Params :
  - producer_pubkey
  - [group_id](#param-group_id)
  - action

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==","group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add"}' https://127.0.0.1:8002/api/v1/group/producer | jq
```

**Params**:

```json
{
    "producer_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "action": "add"
}
```

| Param | Description |
| --- | --- |
| "action" | "add" // add or remove |
| "producer_pubkey" | producer public key |
| "memo" | optional |

API return value:

```json
{
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "producer_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
    "owner_pubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
    "sign": "304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea",
    "trx_id": "6bff5556-4dc9-4cb6-a595-2181aaebdc26",
    "memo": "",
    "action": "ADD"
}
```

| Param | Description |
| --- | --- |
| "producer_pubkey" | publikc key for producer just added |
| "sign" | string | 签名|
| "action" | Add or REMOVE |
| "memo" | memo |

<span id="api-get-producers"></span>

[>>> back top](#top)

## Get producers

**API**: ```*/api/v1/group/{group_id}/producers```

- Method: GET
- Usage : Get producers
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8005/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/producers | jq
```

API return value:

```json
[
    {
        "ProducerPubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
        "OwnerPubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
        "OwnerSign": "3046022100e29a892a9e66f9a736a7d9672db7bd9e2431b8bcff6d407723303a14bc53c66e022100ecf61ce2ff95109fb6504094104afca7074a7c96ac79733cab98cef0e5f85baf",
        "TimeStamp": 1634755122424178000,
        "BlockProduced": 3
    },
    {
        "ProducerPubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
        "OwnerPubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
        "OwnerSign": "304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea",
        "TimeStamp": 1634756661280204800,
        "BlockProduced": 0
    }
]
```

| Param | Description |
| --- | --- |
| "ProducerPubkey" | Producer Pubkey|
| "OwnerPubkey" | Owner Pubkey |
| "OwnerSign" | Owner 签名 |
| "TimeStamp" | Timestamp|
| "BlockProduced" | 该 Producer 目前实际生产的区块数 |

* 注意，如果 ProducerPubkey 和 OwnerPubkey 相同，则说明这是 Owner，上例可以看出，Owner 目前共生产了 3 个区块，Producer `<CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ>` 还没有生产区块

[>>> back top](#top)

<span id="api-post-producer-remove"></span>

## Owner remove producer

**API**: ```*/api/v1/group/producer```

- Method: POST
- Usage : Owner remove producer
- Params :
  - producer_pubkey
  - [group_id](#param-group_id)
  - action

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==","group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"remove"}' https://127.0.0.1:8002/api/v1/group/producer | jq
```

```json
{
    "producer_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "action": "remove"
}
```

[>>> back top](#top)

<span id="test-deniedlist"></span>

# DeniedList

<span id="api-get-deniedlist"></span>

## Get deniedlist

**API**: ```*/api/v1/group/{group_id}/deniedlist```

- Method: GET
- Usage : Get deniedlist
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' https://127.0.0.1:8002/api/v1/group/:group_id/deniedlist
```

说明：获取一个节点禁止访问名单列表

API return value:

```json
[
    {
        "GroupId": "f4273294-2792-4141-80ba-687ce706bc5b",
        "PeerId": "QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY111",
        "GroupOwnerPubkey": "CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==",
        "GroupOwnerSign": "3046022100c2c07b0b03ea5a624dbe07b2cb30ad08a5282a017b873c8defbec9656ae4f8da022100a3659f8410151c811ee331de9cbdf719ec9db33170a95dddfe2c443ace36f3c3",
        "TimeStamp": 1632514808574721034,
        "Action": "add",
        "Memo": ""
    }
]
```

数组，包含该组已经被 Owner 屏幕的用户 id 列表

| Param | Description |
| --- | --- |
| "GroupId" |
| "PeerId" | 被屏蔽的用户 id |
| "GroupOwnerPubkey" | public key of group owner (ecdsa) |
| "GroupOwnerSign" | 组拥有者的签名（可通过 pubkey 验证） |
| "Timestamp" | 操作执行的时间戳 |
| "Acition" | "add" |
| "memo" |

[>>> back top](#top)

<span id="api-post-deniedlist-add"></span>

## Add deniedlist

**API**: ```*/api/v1/group/deniedlist```

- Method: POST
- Usage : Add deniedlist
- Params :
  - [peer_id](#param-peer_id)
  - [group_id](#param-group_id)
  - action

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","group_id":"f4273294-2792-4141-80ba-687ce706bc5b", "action":"add"}' https://127.0.0.1:8002/api/v1/group/deniedlist
```

**Params**:

```json
{
    "peer_id": "QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m",
    "group_id": "f4273294-2792-4141-80ba-687ce706bc5b",
    "action": "add"
}
```

| Param | Description |
| --- | --- |
| "action" | "add" |
| "memo" | memo |

说明：只有创建该组的节点才能执行此操作，也就是需要 group_owner 的权限，添加后会通过 block 广播至组中其他节点

注意：黑名单操作分为 2 种情况

1. 被组屏蔽的节点发出的 trx 会被 producer 或拒绝拒绝，因此无法向节点中发布内容，但是因为新块是通过广播发送的，此时该节点仍可以获得组中得新块（也即只要节点不退出，仍然可以看到新内容)

2. 被组屏蔽的节点如果退出并再次打开，此时发送的 ASK_NEXT 请求将被 Owner 或 Producer 拒绝，因此无法获取节点中最新的块

API return value:

```json
{
    "group_id": "f4273294-2792-4141-80ba-687ce706bc5b",
    "peer_id": "QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m",
    "owner_pubkey": "CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==",
    "sign": "30460221008d7480261a3a33f552b268429a08f8b0ede03b4ddc8014d470d84e707a80d600022100b1616d4662f3e7f0c7594381e425e0c26cf25b66a2cef9437d320cccb0871e5b",
    "trx_id": "2f434ac3-c2a8-494a-9c58-d03a8b51dab5",
    "action": "add",
    "memo": ""
}
```

**Params**:

| Param | Description |
| --- | --- |
| "sign" | 组拥有者的签名（可通过 pubkey 验证） |
| "memo" | "Add" |

[>>> back top](#top)

<span id="api-post-deniedlist-del"></span>

## Del deniedlist

**API**: ```*/api/v1/group/deniedlist```

- Method: POST
- Usage : Del deniedlist
- Params :
  - [peer_id](#param-peer_id)
  - [group_id](#param-group_id)
  - action

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY222","group_id":"f4273294-2792-4141-80ba-687ce706bc5b", "action":"del"}' http://127.0.0.1:8002/api/v1/group/deniedlist
```

**Params**:

```json
{
    "peer_id": "QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY222",
    "group_id": "f4273294-2792-4141-80ba-687ce706bc5b",
    "action": "del"
}
```

API return value:

```json
{
    "group_id": "f4273294-2792-4141-80ba-687ce706bc5b",
    "peer_id": "QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY222",
    "owner_pubkey": "CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==",
    "sign": "304402202854e4ed1efa7f4bc468fe73b566d3159e001fddd2d1625008463d2812bdc85a02207f40c91a8a12a139ddd796f11947e4a809e08a31735408e401f0e4866d167852",
    "trx_id": "41343f27-4193-425d-aa39-591aa172b4db",
    "action": "del",
    "memo": ""
}
```

**Params**:

| Param | Description |
| --- | --- |
| "sign" | 组拥有者的签名（可通过 pubkey 验证） |
| "action" | "del" |
| "memo" | "" |

[>>> back top](#top)

<span id="test-group-config"></span>

# Group Config

<span id="api-post-group-config-add"></span>

## Add group config

**API**:  ```*/api/v1/group/config```

- Method: POST
- Usage : Add group config
- Params :
  - [group_id](#param-group_id)
  - action
  - name
  - type
  - value
  - memo

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"action":"add", "group_id":"c8795b55-90bf-4b58-aaa0-86d11fe4e16a", "name":"test_bool", "type":"int", "value":"false", "memo":"add test_bool to group"}' https://127.0.0.1:8002/api/v1/group/config | jq
```

**Params**:

```json
{
    "action": "add",
    "group_id": "c8795b55-90bf-4b58-aaa0-86d11fe4e16a",
    "name": "test_bool",
    "type": "int",
    "value": "false",
    "memo": "add test_bool to group"
}
```

| Param | Description |
| --- | --- |
| "action" | add or del |
| "name" | 配置项的名称 |
| "type" | 配置项的类型，可选值为 "int", "bool", "string" |
| "value" | 配置项的值，必须与 type 相对应 |
| "memo" | memo |

权限：

只有 group owner 可以调用该 API

调用后，通过块同步，组内所有节点获得该配置

API return value:

```json
{
    "group_id": "c8795b55-90bf-4b58-aaa0-86d11fe4e16a",
    "sign": "3045022100e1375e48cfbd51cb78afc413fcca084deae9eb7f8454c54832feb9ae00fada7702203ee6fe2292ea3a87d687ae3369012b7518010e555b913125b8a7bf54f211502a",
    "trx_id": "9e54c173-c1dd-429d-91fa-a6b43c14da77"
}
```

| Param | Description |
| --- | --- |
| "sign" | owner 对该 trx 的签名 |

[>>> back top](#top)

<span id="api-get-group-config-keylist"></span>

## Get group config keylist

**API**:  ```*/api/v1/group/{group_id}/config/keylist```

- Method: GET
- Usage : Get group config keylist
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/group/c8795b55-90bf-4b58-aaa0-86d11fe4e16a/config/keylist
API：/v1/group/<GROUP_ID>/config/keylist
```

API return value:

```json
[
    {
        "Name": "test_string",
        "Type": "STRING"
    }
]
```

**Params**:

| Param | Description |
| --- | --- |
| "name" | 配置项的名称 |
| "type" | 配置项的数据类型 |

[>>> back top](#top)

<span id="api-get-group-config-keyname"></span>

## Get group config keyname

**API**:  ```*/api/v1/group/{group_id}/config/{KEY_NAME}```

- Method: GET
- Usage : Get group config keyname
- Params :
  - [group_id](#param-group_id)
  - key_name

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/group/c8795b55-90bf-4b58-aaa0-86d11fe4e16a/config/test_string | jq
```

API return value:

```json
{
    "Name": "test_string",
    "Type": "STRING",
    "Value": "123",
    "OwnerPubkey": "CAISIQJOfMIyaYuVpzdeXq5p+ku/8pSB6XEmUJfHIJ3A0wCkIg==",
    "OwnerSign": "304502210091dcc8d8e167c128ef59af1b6e2b2efece499043cc149014303b932485cde3240220427f81f2d7482df0d9a4ab2c019528b33776c73daf21ba98921ee6ff4417b1bc",
    "Memo": "add test_string to group",
    "TimeStamp": 1639518490895535600
}
```

参数同[添加组内配置](#api-post-group-config-add)

[>>> back top](#top)

<span id="api-post-group-schema"></span>

## Add group schema

**API**:  ```*/api/v1/group/schema```

- Method: POST
- Usage : Add group schema
- Params :
  - [group_id](#param-group_id)
  - rule
  - type
  - aciton
  - memo

添加组内 app 的 schema json

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"rule":"new_schema","type":"schema_type", "group_id":"13a25432-b791-4d17-a52f-f69266fc3f18", "action":"add", "memo":"memo"}' https://127.0.0.1:8002/api/v1/group/schema
```

```json
{
    "rule": "new_schema",
    "type": "schema_type",
    "group_id": "13a25432-b791-4d17-a52f-f69266fc3f18",
    "action": "add",
    "memo": "memo"
}
```

[>>> back top](#top)

<span id="api-get-group-schema"></span>

## Get group schema

**API**:  ```*/api/v1/group/{group_id}/schema```

- Method: GET
- Usage : Get group schema
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/group/13a25432-b791-4d17-a52f-f69266fc3f18/app/schema | jq
```

API return value:

```json
[
    {
        "Type": "schema_type",
        "Rule": "new_schema",
        "TimeStamp": 1636047963013888300
    }
]
```

[>>> back top](#top)

<span id="test-private-group"></span>

# Private Group

**workflow**:

1. Announce user's encrypt pubkey to a group
2. view announced users
3. approve a users

<span id="api-post-announce-user"></span>

## Announce user

**API**: ```*/api/v1/group/announce```

- Method: POST
- Usage : Announce user
- Params :
  - [group_id](#param-group_id)
  - action, "add" or "remove"
  - type, "user"
  - memo

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add", "type":"user", "memo":"invitation code:a423b3"}' https://127.0.0.1:8003/api/v1/group/announce | jq
```

**Params**:

```json
{
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "action": "add",
    "type": "user",
    "memo": "invitation code:a423b3"
}
```

API return value:

```json
{
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "sign_pubkey": "CAISIQJwgOXjCltm1ijvB26u3DDroKqdw1xjYF/w1fBRVdScYQ==",
    "encrypt_pubkey": "age1fx3ju9a2f3kpdh76375dect95wmvk084p8wxczeqdw8q2m0jtfks2k8pm9",
    "type": "AS_USER",
    "action": "ADD",
    "sign": "304402206a68e3393f4382c9978a19751496e730de94136a15ab77e30bab2f184bcb5646022041a9898bb5ff563a6efeea29b30bac4bebf0d3464eb326fd84322d98919b3715",
    "trx_id": "8a4ae55d-d576-490a-9b9a-80a21c761cef"
}
```

| Param | Description |
| --- | --- |
| "sign_pubkey" | user's sign pubkey |
| "encrypt_pubkey" | user's encrypt pubkey |
| "type" | "AS_USER" |
| "action" | "ADD" |
| "sign" | user's signature |

[>>> back top](#top)

<span id="api-get-announced-users"></span>

## Get announced users

**API**: ```*/api/v1/group/{group_id}/announced/users```

- Method: GET
- Usage : get announced users
- Params :
  - [group_id](#param-group_id)

**Example**:

```bash
curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8002/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/announced/users
```

API return value:

```json
[
    {
        "AnnouncedSignPubkey": "CAISIQIWQX/5Nmy2/YoBbdO9jn4tDgn22prqOWMYusBR6axenw==",
        "AnnouncedEncryptPubkey": "age1a68u5gafkt3yfsz7pr45j5ku3tyyk4xh9ydp3xwpaphksz54kgns99me0g",
        "AnnouncerSign": "30450221009974a5e0f3ea114de8469a806894410d12b5dc5d6d7ee21e49b5482cb062f1740220168185ad84777675ba29773942596f2db0fa5dd810185d2b8113ac0eaf4d7603",
        "Result": "ANNOUNCED"
    }
]
```

**Params**:

| Param | Description |
| --- | --- |
| "AnnouncedPubkey" | user's pubkey |
| "AnnouncerSign" |string | user's signture |
| "Result" | ANNOUNCED or APPROVED |

[>>> back top](#top)

<span id="api-post-group-user"></span>

## Owner approve a user

**API**: ```*/api/v1/group/user```

- Method: POST
- Usage : owner approve a user
- Params :
  - [user_pubkey](#param-user_pubkey)
  - [group_id](#param-group_id)
  - action, "add" or "remove"

**Example**:

```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"user_pubkey":"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==","group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add"}' https://127.0.0.1:8002/api/v1/group/user | jq
```

**Params**:

```json
{
    "user_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "action": "add"
}
```

API return value:

```json
{
    "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
    "user_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
    "owner_pubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
    "sign": "304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea",
    "trx_id": "6bff5556-4dc9-4cb6-a595-2181aaebdc26",
    "memo": "",
    "action": "ADD"
}
```

| Param | Description |
| --- | --- |
| "sign" | signature |

[>>> back top](#top)

<span id="param-list"></span>

# Common Params

<span id="param-group_id"></span>

## group_id

string

可以通过 [API: List all groups](#api-get-groups) 查询 node 所加入的所有 groups （包括自己创建的） 信息

<span id="param-group_name"></span>

## group_name

string, group name

create a group 时的必填字段

可以通过 [API: List all groups](#api-get-groups) 查询 node 所加入的所有 groups （包括自己创建的） 信息

<span id="param-trx_id"></span>

## trx_id/TrxId

可以通过 gettrx API 获取具体内容

<span id="param-block_id"></span>

## block_id/BlockId

[API: Get Block Info](#api-get-block)

one block can have a number of trxs.

<span id="param-node_id"></span>

## node_id

[API: Get Node Info](#api-get-node)

节点的 node_id

* 之前的 user_id 取消了(实际上是 peer_id)
* 前端可以用 pubkey 来当 user_id（唯一标识）

<span id="param-peer_id"></span>

## peer_id

peer_id (可以通过节点信息 API 获得)

<span id="param-owner_pubkey"></span>

<span id="param-user_pubkey"></span>

## owner_pubkey/user_pubkey

owner_pubkey: public key of group owner (ecdsa)

user_pubkey: public key of group user *

When join a new group, a user public key will be created for this group, for group owner, user_pubkey is as same as owner_pubkey

user_pubkey 是用户在组内的唯一身份标识，也用来进行签名

<span id="param-group_status"></span>

## group_status

status of group, a group can have 3 different status:

    - SYNCING
    - SYNC_FAILED
    - IDLE

for detail please check RUM design document.

<span id="param-app_key"></span>

## app_key

string, group app key, requested, length should between 5 to 20

<span id="param-consensus_type"></span>

## consensus_type

string, group consensus type, must be "poa", requested

"poa" or "pos" or "pow", "poa" only for now

链上共识方式，参见 RUM 设计文档

<span id="param-encryption_type"></span>

## encryption_type

string, group encryption type, must be "public", requested

encryption type of group, "public" or "private" 

[>>> back top](#top)