# RUM development tutorial 

![Main Test](https://github.com/rumsystem/quorum/actions/workflows/maintest.yml/badge.svg)

## Run testing

```go test cmd/main* -v```

## Generate API Docs

```go run cmd/docs.go```

Open url ```http://localhost:1323/swagger/index.html``` in the browser. 

## Setup local test network

1. Download and install go (ver 1.15.2)
2. Clone quorum project from 
```https://github.com/rumsystem/quorum.git```
3. 3 local nodes will be created
    - bootstrap_node (bootstrap)
    - owner_node   (group owner node)
    - user_node    (group user node)
4. cd to quorum souce code path and create config/ dir
```bash
mkdir -p config 
```
5. start bootstrap node 
```bash
go run cmd/main.go -bootstrap -listen /ip4/0.0.0.0/tcp/10666 -logtostderr=true
```
output
```bash
I0420 14:58:47.719592     332 keys.go:47] Load keys from config
I0420 14:58:47.781916     332 main.go:64] Host created, ID:<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR>, Address:<[/ip4/172.28.230.210/tcp/10666 /ip4/127.0.0.1/tcp/10666]>
```
Record <HOST_ID>, for example:
    QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR
5. Start owner node
```bash
go run cmd/main.go -peername owner -listen /ip4/127.0.0.1/tcp/7002 -apilisten :8002 -peer /ip4/127.0.0.1/tcp/10666/p2p/<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR> -configdir config -datadir data -keystoredir ownerkeystore  -jsontracer ownertracer.json -debug true 
```        
- For the first time, user will be asked to input a password for the node, if not given, a password will be created for the node
- After a password is created, next time user will be asked to input the password to open node.
- env RUM_KSPASSWD can be used to input node password, like:
```bash
RUM_KSPASSWD=<node_passwor> go run cmd/main.go...
```
6. Start user node
```bash
go run cmd/main.go -peername user -listen /ip4/127.0.0.1/tcp/7003 -apilisten :8003 -peer /ip4/127.0.0.1/tcp/10666/p2p/<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR> -configdir config -datadir data -keystoredir ownerkeystore  -jsontracer usertracer.json -debug true 
``` 

## Owner node create "test_group"
```bash
curl -k -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' https://127.0.0.1:8002/api/v1/group
```
- API:/api/v1/group 
- Method:POST
- Usage : create new group
- Params : 
    * group_name：string, group name, requested
    * consensus_type:string, group consensus type, must be "poa", requested
    * encryption_type: string, group encryption type, must be "public", requested
    * app_key: string, group app key, requested, length should between 5 to 20
- API return value:
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
- Params:        
    * genesis_block       //genesis block, the first block of the group
    * group_id            
    * group_name          
    * owner_pubkey        //owner pubkey(ecdsa)               
    * owner_encryptpubkey //owner encryption key(age)
    * consensus_type      
    * encryption_type     
    * cipher_key          //aes key*
    * app_key             
    * signature           //owner signature

    \*neglect group encryption type (public or private), all trx except "POST" will be encrypted by cipher_key

returned json string from API call is the "seed" of the newly created group.

## Owner node list all groups
```bash
curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/groups
```
- API: /api/v1/group 
- Method: GET
- Params: none
- API return value:
```json
{
    "groups": 
    [
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
            "highest_block_id": "a865ae03-d8ce-40fc-abf6-ea6f6132c35a"
            "group_status": "IDLE"
        }
    ]
}
```
- Params:
    * group_id           
    * group_name        
    * owner_pubkey      
    * user_pubkey       \*
    * consensus_type     
    * encryption_type     
    * cipher_key          
    * app_key                  
    * last_updated        
    * highest_height      \*\*
    * highest_block_id    \*\*\*                 
    * group_status        \*\*\*\*

    \* When join a new group, a user public key will be created for this group, for group owner, user_pubkey is as same as owner_pubkey
    
    \*\* Heighty of the "highest" block in this group 
    
    \*\*\* block_id of the "highest" block in this group
    
    \*\*\*\* status of group, a group can has 3 different status
    - SYNCING
    - SYNC_FAILED  
    - IDLE

    for detail please check RUM design document          


    - 节点B加入组"my_test_group"

        执行：

            curl -k -X POST -H 'Content-Type: application/json' -d '{"genesis_block":{"BlockId":"36ac6e22-80a1-4d54-abbb-8bd2c55ef8cf","GroupId":"eae3f0db-a034-4c5f-a25f-b1177390ec4d","ProducerPubKey":"CAISIQMJIG4do9g8PBixH432YXVQmD7Ilqp7DzbGxgLJHbRoFA==","Hash":"fDGwAPJbHHG0GpKLQZnRolK9FUO5nSIod/iprwQQn8g=","Signature":"MEYCIQDo5uge+saujb0WR6ZreISDYWpRzY6PQ3f5ly7vtHHgkQIhAKcuwDT2fIpBDx/7lQU6mIBQKJuQeI0Zbw3W7kHfBO28","Timestamp":1631804384241781200},"group_id":"eae3f0db-a034-4c5f-a25f-b1177390ec4d","group_name":"my_test_group","owner_pubkey":"CAISIQMJIG4do9g8PBixH432YXVQmD7Ilqp7DzbGxgLJHbRoFA==","owner_encryptpubkey":"age1lx3zh5sc5cureh484t5tm2036lhrzdnh96rfaft6echs9cqsefss4yn886","consensus_type":"poa","encryption_type":"public","cipher_key":"3994c4224da17ad50504c78458f37249149477c7bc643f3fe78e44033c17874a","signature":"30450220591361918948140c8ad1736cde3831f326470f2d3c5105a0b63867c7b216857c0221008921422c6e1974834d5610d4c6ad1a9dd0394ac464dfc12659cde41d75172d14"}' https://127.0.0.1:8003/api/v1/group/join
        
            API：/v1/group/join ，加入一个组
            参数：组“种子”的json串（之前步骤的结果）

        返回值：
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
            
            group_id            组id
            group_name          组名称
            owner_pubkey        组owner的公钥
            user_pubkey         本节点在组内的签名公钥 *
            user_encryptpubkey  本节点在组内的加密公钥 **
            consensus_type      组共识类型（POA POS)
            encryption_type     组加密类型（PRIVATE PUBLIC）
            cipher_key          组内协议对称加密密钥           
            signature           签名

            * user_pubkey是用户在组内的唯一身份标识，也用来进行签名
            ** 如果组类型为PRIVATE，则该加密公钥需要用其他协议进行组内广播（TBD）

        节点B加入组后，开始自动同步(SYNCING)，同步完成后状态变为（IDLE)

    - 节点A post to group

        执行：

            curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/content

            参数： 
                group_id : 组id
                content：发布的内容

        返回值：

            {"trx_id":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d"}

            参数：
                trx_id: post的trx_id


     Like/Dislike a object in the group 

    curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Like","object":{"id":"578e65d0-9b61-4937-8e7c-f00e2b262753"}, "target":{"id":"c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/content


    - 节点B查询组内节点A的POST
        
        执行:
            curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8003/api/v1/group/c0c8dc7d-4b61-4366-9ac3-fd1c6df0bf55/content
        
        参数：
            group_id : 组id

        返回值:
            [{"TrxId":"da2aaf30-39a8-4fe4-a0a0-44ceb71ac013","Publisher":"CAISIQOlA37+ghb05D5ZAKExjsto/H7eeCmkagcZ+BY/pjSOKw==","Content":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"TypeUrl":"quorum.pb.Object","TimeStamp":1629748212762123400}]

            参数：
	            TrxId     string    //trx_id
	            Publisher string    //发布者
	            Content   string    //内容
                TypeURL   string    //Type
	            TimeStamp int64                  


    - /v1/group/leave ，离开一个组

        例子：

            curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' https://127.0.0.1:8002/api/v1/group/leave

        返回值:

            {"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","signature":"Owner Signature"}

            参数
                group_id :  组id
                signature : 签名

    - /api/v1/group ，删除一个组

        例子：

            curl -k -X DELETE -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' https://127.0.0.1:8003/api/v1/group

        返回值:
            {"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB","signature":"owner_signature"}

        参数
            group_id : 组id

    - /api/v1/node , 获取节点信息

        例子：

            curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8003/api/v1/node

        返回值：

            {"node_publickey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCyOIkTFxF1v8xborMo/6k45AMpfijbmT3OithJ/XTn8MDhnMw6j/jzw8YFSIfDj4KfjpwlyuVVZbSxHjeFFKMAWJgkeTNxRYLxXQWbZKd6d9PeRKLpdv/oEyDoPpdigMON84M1VWx9W0/lJ8Nps+cuI+7ugMLue40lAAUXXTPSaKy7vrgvQplKyfE4chPRY+bOAdmZDm76G00bGW6p4D2SLgApGXaG4grhGGvmJutAByIcaJRlpQu2mvgvjUAArP+YLw8scNvWzShGU/gz8tUFtus6c/cez/TmIUjeuD2hbbM+Gn1CxJxx/v0P59+hQT+f2NCM8yKC2KoXQkm5Llz2cUbJWbfOOQEkDCWRibYNEIUHYjWEL5xOcKLb4ie3vmJ5mz3kmI0iEDcx7OvTw7dtJGCo9GG5yPLITI0T3ygsjLUIpooY6PhOTIWvMqBVmiovUzb6cUb5Tms226KkP2ZOqNqqkwkN6zGI27ePdRde5N9N9zkwZd9ESaeOeea1BGDINyfpV1x2jk90BXRE7sB7f4eQrhCwtEHsoiZLUV4QevKO03XMMAGOmT6fQGACe6sVSeGfouNjKsgp0KrTRTtHIJCdGHNUNiv38ZGgRUWiwzPR83aJ24OJT2CNhLUvZk8tu5PagV19+4VKQ5OIOotHJusLvc1oibKCwv7sf6b2pQIDAQAB","node_status":"NODE_ONLINE","node_version":"ver 0.01","user_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m"}

            参数：
                node_publickey: 组创建者的pubkey
                node_status   : 节点状态，值可以是 “NODE_ONLINE"和”NODE_OFFLINE“
                node_version  : 节点的协议版本号
                node_id       : 节点的node_id*

            *之前的user_id取消了(实际上是peer_id)
            *现在只返回真正的node_id，前端可以用pubkey来当user_id（唯一标识）

    - 获取一个块的完整内容
    
        例子：
            ccurl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8003/api/v1/block/<GROUP_ID>/<BLOCK_ID>

        返回值：
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
                    

    - 获取一个trx的完整内容
        
        例子:
            curl -k -X GET -H 'Content-Type: application/json' -d https://127.0.0.1:8003/api/v1/trx/<GROUP_ID>/<TRX_ID>

            * "裸"trx的内容，data部分是加密的(加密类型由组类型决定)
            * 客户端应通过获取Content的API来获取解密之后的内容

    - 添加组黑名单
    
        例子：
            curl -k -X POST -H 'Content-Type: application/json' -d '{"peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","group_id":"f4273294-2792-4141-80ba-687ce706bc5b", "action":"add"}}' https://127.0.0.1:8002/api/v1/group/deniedlist
        
            参数： 
                action: "add" 
                peer_id:   想要屏蔽的peer_id
                group_id:组id，哪个组想要屏蔽该用户
                memo ： meno

            说明：只有创建该组的节点才能执行此操作，也就是需要group_owner的权限，添加后会通过block广播至组中其他节点
            注意：黑名单操作分为2种情况
                1. 被组屏蔽的节点发出的trx会被producer或拒绝拒绝，因此无法向节点中发布内容，但是因为新块是通过广播发送的，此时该节点仍可以获得组中得新块（也即只要节点不退出，仍然可以看到新内容)
                2. 被组屏蔽的节点如果退出并再次打开，此时发送的ASK_NEXT请求将被Owner或Producer拒绝，因此无法获取节点中最新的块

        返回值：
            {"group_id":"f4273294-2792-4141-80ba-687ce706bc5b","peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","owner_pubkey":"CAISIQMOjdI2nm
Rsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==","sign":"30460221008d7480261a3a33f552b268429a08f8b0ede03b4ddc8014d470d84e707a80d600022100b1616d4662f3e7f0c75
94381e425e0c26cf25b66a2cef9437d320cccb0871e5b","trx_id":"2f434ac3-c2a8-494a-9c58-d03a8b51dab5","action":"add","memo":""}


            group_id：组id
            peer_id:  被屏蔽的用户id
            owner_pubkey: 组拥有者的pubkey
            sign: 组拥有者的签名（可通过pubkey验证）
            trx_id:该操作的trx的id，可以通过gettrx API获取具体内容
            memo: "Add"
    
    - 获取组黑名单 

        例子：

            curl -k -X GET -H 'Content-Type: application/json' http://127.0.0.1:8002/api/v1/group/:group_id/deniedlist

        参数：
            无
        
        说明：获取一个节点的blacklist

        返回值：
[{"GroupId":"f4273294-2792-4141-80ba-687ce706bc5b","PeerId":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY111","GroupOwnerPubkey":"CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==","GroupOwnerSign":"3046022100c2c07b0b03ea5a624dbe07b2cb30ad08a5282a017b873c8defbec9656ae4f8da022100a3659f8410151c811ee331de9cbdf719ec9db33170a95dddfe2c443ace36f3c3","TimeStamp":1632514808574721034,"Action":"add","Memo":""},{"GroupId":"f4273294-2792-4141-80ba-687ce706bc5b","PeerId":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY222","GroupOwnerPubkey":"CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==","GroupOwnerSign":"304502201f21befe212c21b77abf49f5eacb24148a2e99f5c8e969a718a6bd7d5a051e2b022100fce4099125474fc765c14c143d7271583ea9e65677960106ebc7288ea93191c9","TimeStamp":1632515625550737844,"Action":"add","Memo":""}]
            
            数组，包含该节点所有已经屏蔽的组-用户对
                GroupId:组id
                PeerId:用户id
                OwnerPubkey：组拥有者pubkey
                OwnerSign:执行该操作的签名
                Action: add or del
                Timestamp：操作执行的时间戳

    - 删除组黑名单
    
        例子：
            curl -k -X POST -H 'Content-Type: application/json' -d '{"peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY222","group_id":"f4273294-2792-4141-80ba-687ce706bc5b", "action":"del"}}' http://127.0.0.1:8002/api/v1/group/deniedlist

        参数： 
        action: "del"
        peer_id:  想要解除的peer_id (可以通过节点信息API获得)
        group_id ：必须为"Group"

        结果：
            
            {"group_id":"f4273294-2792-4141-80ba-687ce706bc5b","peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY222","owner_pubkey":"CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==","sign":"304402202854e4ed1efa7f4bc468fe73b566d3159e001fddd2d1625008463d2812bdc85a02207f40c91a8a12a139ddd796f11947e4a809e08a31735408e401f0e4866d167852","trx_id":"41343f27-4193-425d-aa39-591aa172b4db","action":"del","memo":""}

            group_id：组id
            peer_id:  移除黑名单的用户id
            owner_pubkey: 组拥有者的pubkey
            sign: 组拥有者的签名（可通过pubkey验证）
            trx_id:该操作的trx的id，可以通过gettrx API获取具体内容
            action: "del"
            memo: ""

    - Producer

        Producer作为组内“生产者”存在，可以代替Owner出块，组内有其他Producer之后，Owenr可以不用保持随时在线，
        在Owner下线的时间内，Producer将代替Owner执行收集Trx并出块的任务
        
        关于Producer的内容，如具体的共识算法，Producer的收益等内容，请参考RUM设计文档

        Owner作为组内第一个Producer存在，有其他Producer存在时，如果Owner在线，也将作为一个Producer存在

        有Producer存在的流程如下

            1. Owner 创建组
            2. Owner 作为Producer存在，负责出块
            3. 其他Producer获得组的seed，加入组，完成同步
            4. Producer用Announce API将自己作为Producer的意愿告知Owner
                例： curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add", "type":"producer", "memo":"producer p1, realiable and cheap, online 24hr"}' https://127.0.0.1:8005/api/v1/group/announce | jq

                API：/v1/group/announce
                参数：
                    group_id：组id
                    action ： add or remove
                    type   :  producer 
                    memo   ： memo
                返回值：
                    {
                        "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
                        "sign_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
                        "encrypt_pubkey": "",
                        "type": "AS_PRODUCER",
                        "action": "ADD",
                        "sign": "3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5",
                        "trx_id": "2e86c7fb-908e-4528-8f87-d3548e0137ab"
                    }
                参数：
                    group_id : 组id
                    sign_pubkey : producer在本组的签名pubkey
                    encrypt_pubkey : 没有使用
                    type: AS_PRODUCER
                    action: ADD
                    sign: producer的签名
                    trx_id : trx_id
            5. 其他节点（包括Owner节点）查看所有Announce过的Producer
                例：curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8002/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/announced/producers

                API: /v1/group/{group_id}/announced/producers
                参数：
                    group_id : 组id
                返回值：
                    [
                        {
                            "AnnouncedPubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
                            "AnnouncerSign": "3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5",
                            "Result": "ANNOUCNED",
                            "TimeStamp": 1634756064250457600
                        }
                    ]
                参数：
                    AnnouncedPubkey ： producer pubkey
                    AnnouncerSign： producer的签名
                    Result ： ANNOUNCED or APPROVED，producer刚Announce完毕的状态是ANNOUNCED
                    TimeStamp : timestamp

        6. Owner批准某个Producer
            例：curl -k -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==","group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add"}}' https://127.0.0.1:8002/api/v1/group/producer | jq

            API: /v1/group/producer          
            参数:
                "action":"add" // add or remove
                "producer_pubkey": producer public key
                "group_id": group id 
                "memo" : optional
            返回值：
            {    
                "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
                "producer_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
                "owner_pubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
                "sign": "304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea",
                "trx_id": "6bff5556-4dc9-4cb6-a595-2181aaebdc26",
                "memo": "",
                "action": "ADD"
            }
            参数：
                group_id: group id
                producer_pubkey : publikc key for producer just added 
                owner_pubkey: group owner public key                
                sign: 签名
                trx_id: trx id
                action : Add or REMOVE
                memo : memo
            * 请注意，Owner只可以选择在组内Announce过自己的Producer，
              没有Announce过的producer是不可以添加的
        
        7. 查看组内目前的实际批准的producer
            例：curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8005/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/producers | jq
            
            API: /v1/group/{group_id}/producers
            参数：
                group_id : 组id                            
            返回值
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
            参数：
                "ProducerPubkey": Producer Pubkey,
                "OwnerPubkey":    Owner Pubkey
                "OwnerSign"：     Owner 签名
                "TimeStamp":      Timestamp,
                "BlockProduced":  该Producer目前实际生产的区块数

            * 注意，如果ProducerPubkey和OwnerPubkey相同，则说明这是Owner，上例可以看出，Owner目前共生产了3个区块，Producer <CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ> 还没有生产区块

        8. 查看Announce Producer状态
            例：curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8002/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/announced/producers

            返回值：
                [
                    {
                        "AnnouncedPubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
                        "AnnouncerSign": "3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5",
                        "Result": "APPROVED",
                        "TimeStamp": 1634756064250457600
                    }
                ]
            
            可以看出，经过Owner批准，该Producer的状态（result)变为 APPROVED

    - 添加组内App Schema
        添加组内app的schema json (only API works, parse and apply schema TBD)

    - Trx生命周期，加密和出块过程

        - Trx种类

            所有链上操作均是Trx，客户端相关的Trx有5种
            - POST     user发送组内信息(POST Object)
            - ANNOUNCE user宣布自己在组内的公钥
            - AUTH     Owner调整组内权限
            - SCHEMA   Owner管理组内数据schema
            - PRODUCER Owner管理组内producer
        
        - Trx加密类型

            为了确保后加入的用户能正确使用组内功能，根据trx类型，进行如下形式的加密

            - POST
                - 强加密组： 每个发送节点都要根据自己手中的组内成员名单（公钥名单），对POST的内容进行非对称加密，然后发送，收到trx的节点使用自己的公钥对trx进行解密
                - 弱加密组： 每个发送节点都用seed中的对称加密字串对收发的trx数据进行对称加密

            - 其他trx
                - 所有其他链相关的协议均使用弱加密组策略（用seed中的对称加密字串进行收发）

        - 出块流程/共识策略

            一个Trx被push到链上后，根据group的不同共识类型，将被采取不同形式对待：

            - 链上共识方式，参见RUM设计文档

        
    - Trx状态判断
        
        同其他链相似，Trx的发送没有重试机制，客户端应自己保存并判断一个Trx的状态，具体过程如下

        1. 发送一个trx时，获取trx_id

        curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/content

        {"trx_id":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d"}

        2. 将这个trx标记为“发送中”
        3. 查询组内的内容
        例：curl -k -X GET -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' http://127.0.0.1:8002/api/v1/group/content
    
        返回值：
        [
            {
                "TrxId":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d","Publisher":"Qmbt56A7gVueThDVxfvLstxSR7BhE6M8doqxZXKWGBEbxT",
                "Content":{
                        "type":"Note",
                        "content":"simple note by aa",
                        "name":"A simple Node id1"
                      },
                "TimeStamp":1619656412253363059
            }
        ]

        4. 设置一个超时，目前建议是30秒，不断查询，直到相同trx_id的内容出现在返回结果中，即可认为trx发送成功（被包含在块中），如上例所示
        5. 如果超时被触发，没有查到结果，即认为发送trx失败，客户端可以自行处理重发

        * AUTH相关的trx处理方式相同（黑名单）

    - 节点网络信息

        curl -k http://localhost:8002/api/v1/network

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

        这里需要注意， nat_type和addrs都会改变，开始的时候没有公网地址，类型是Unknown 之后会变成Private，再过一段时间反向链接成功的话，就变成Public，同时Addrs里面出现公网地址。

    - 手动发起同步

        客户端可以手动触发某个组和组内其他节点同步块

        例：curl -X POST -H 'Content-Type: application/json' -d '' http://<IP_ADDR>/api/v1/group/<GROUP_ID>/startsync
        API: v1/group/<GROUP_ID>/startsync
        参数：group_id : 组id
        返回值：
        200 ： {"GroupId":<GROUP_ID>,"Error":""}， GROUP_ID的组正常开始同步，同时组的状态会变为SYNCING
        500 ： {"GroupId":<GROUP_ID>,"Error":"GROUP_ALREADY_IN_SYNCING"}, GROUP_ID的组当前正在同步中


    - App API:

        Request content with senders filter

        curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[ "CAISIQP8dKlMcBXzqKrnQSDLiSGWH+bRsUCmzX42D9F41CPzag=="]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20" 

        Requst all content

        curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20" 

