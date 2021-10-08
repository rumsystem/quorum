![Main Test](https://github.com/rumsystem/quorum/actions/workflows/maintest.yml/badge.svg)

设置测试环境

Run testing

```go test cmd/main* -v```

API Docs

```go run cmd/docs.go```

Open url ```http://localhost:1323/swagger/index.html``` in the browser. 

本地开发环境配置

    1. 安装go
    2. 下载  https://github.com/rumsystem/quorum.git

    共需要3个本地节点，进入本地目录，例如 ~/work/quorum
    
    - 启动BootStrap节点 (mkdir -p config): 
        
        go run cmd/main.go -bootstrap -listen /ip4/0.0.0.0/tcp/10666 -logtostderr=true
        I0420 14:58:47.719592     332 keys.go:47] Load keys from config
        I0420 14:58:47.781916     332 main.go:64] Host created, ID:<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR>, Address:<[/ip4/172.28.230.210/tcp/10666 /ip4/127.0.0.1/tcp/10666]>    
        记下ID <#ID>，例如 QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR
        
    - 启动本地节点:
        RUM_KSPASSWD=<PASSWORD> go run cmd/main.go -peername peer2 -listen /ip4/127.0.0.1/tcp/<NETWORK_PORT> -apilisten :<API_PORT> -peer /ip4/127.0.0.1/tcp/10666/p2p/<BOOT_STRAP_NODE_ID> -configdir <PATH_TO_CONFIG> -datadir <PATH_TO_DATA> -keystoredir <PATH_TO_KEY_STORE> -jsontracer <JSON_TRACER_FILE_NAME> -debug true 
        * 密码第一次启动节点时生成，通过环境变量传入
        * 用bootstrap节点的ID替代 <#ID>
        * 节点A的本地HTTP端口地址为8002，以下所有curl命令中发给8002端口的API Call都是调用节点A

API

    - 节点A创建组 "my_test_group"

        执行：

        curl -k -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group", "consensus_type":"poa", "encryption_type":"public", "app_key":"test_app"}' https://127.0.0.1:8002/api/v1/group

            API：  /api/v1/group  ，创建新组
            参数：
                "group_name"      string, 组名称，必填
                "consensus_type"  string, 组共识类型，必填，目前仅支持 "poa" (proof of authority)
                "encryption_type" string, 组加密类型，必填， "public" or "private"
                "app_key"         strnig, 组 app key, 必填，长度为5到20的字符串，用来标识本组的对应的app
       
        API返回值：
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
            
            参数：
                genesis_block       组的genesis block
                group_id            组id   
                group_name          组名称
                owner_pubkey        组owner签名用公钥(ecdsa)               
                owner_encryptpubkey 组owner加密key(age)
                consensus_type      组共识类型
                encryption_type     组加密类型 
                cipher_key          组内协议对称加密密钥(aes)
                app_key             组app key
                signature           组owner对结果的签名 
            
            *不管组内加密类型如何设置，组内除了POST之外的其他协议都通过该key进行对称加密/解密            

        该调用返回的json串就是新创建组的“种子”，保存到文件中

    - 查看节点A所拥有的组

        执行：

            curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/groups

            Endpoint : /api/v1/group ，返回节点所加入（含自己创建）的所有组
            参数 : 无

        API返回值：
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
                "highest_block_id": [
                    "a865ae03-d8ce-40fc-abf6-ea6f6132c35a"
                ],
                "group_status": "IDLE"
                }
            ]
            }

            group_id            组id
            group_name          组名称
            owner_pubkey        组owner的公钥
            user_pubkey         本节点在组内的公钥
            consensus_type      组共识类型（POA POS)
            encryption_type     组加密类型（PRIVATE PUBLIC）
            cipher_key          组内协议对称加密密钥
            app_key             组app key                  
            last_updated        最后收到块的时间戳
            highest_height      组内最"高"的块高度 **
            highest_block_id    组内最高的块的block_id ***                 
            group_status        组状态 *

                * 该参数有3个可能的返回值，这个参数可以用来显示组状态，详见设计文档
                    - SYNCING
                    - SYNC_FAILED
                    - IDLE
                ** genesis block高度是0
                ***  注意：有多条等长链存在时，可能存在多个块
        
        刚刚创建的组处于IDLE状态

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

    - 管理组内producer

        添加一个producer到组内，只有group owner有此权限

        - 例子                
           
            curl -k -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":"CAISIQKLni9lu428QDeJTjlsjxWXAtaM5+paRIUiiRGgd/H+uw==","group_id":"7bcc598f-d7a8-4299-8a1d-bdf918d48458", "action":"add"}}' https://127.0.0.1:8002/api/v1/group/producer

            - 参数

                "action":"add" // add or del
                "producer_pubkey": producer public key
                "group_id": group id 
                "memo": any memo (optional)


        - 返回值
            {"group_id":"7bcc598f-d7a8-4299-8a1d-bdf918d48458","producer_pubkey":"CAISIQKLni9lu428QDeJTjlsjxWXAtaM5+paRIUiiRGgd/H+uw==","owner_pubkey":"CAISIQP1AJ5cNaoYMBqQV/LoYuIlkWqworXRZ3Q0xf9/3APbBQ==","sign":"304602210095368059515c5845050b7bf1b462988dc10e30c8d338b2ff7ff49d8104058e46022100e654475ed18ee8e65799c5b31abc50275d46b810194945aebcca7247894e03cf","trx_id":"074fe705-6585-43ba-a1d5-fa4b9ada3d89","memo":"","action":"add"}

            group_id: group id
            owner_pubkey: group owner public key
            producer_pubkey : publikc key for producer just added 
            sign: 签名
            trx_id: trx id
            action : Add or del
            memo : meno

    - 查看组内producer

        - 例子            
            curl -k -H 'Content-Type: application/json' https://127.0.0.1:8002/api/v1/group/7bcc598f-d7a8-4299-8a1d-bdf918d48458/producers

        - 参数
            group_id: group id

        - 返回值
            [{"ProducerPubkey":"CAISIQP1AJ5cNaoYMBqQV/LoYuIlkWqworXRZ3Q0xf9/3APbBQ=="}]

        Array of ProducerPubkey


    - Announce 组内 public key

        用户声明自己在组内的public key，对于强加密组，组成员在发送POST trx时，会对该组所有announce过的用户的公钥进行非对称加密，该用必须announce自己的公钥，才能解密其他用户发送的POST

        - 例子

            curl -X POST -H 'Content-Type: application/json' -d '{"action":"add","type":"userpubkey","group_id":"7bcc598f-d7a8-4299-8a1d-bdf918d48458"}' http://127.0.0.1:8003/api/v1/group/announce


            - 参数
                "action":"add" // add or del
                "group_id": group id
                "type": must be "userpubkey"

        - 返回值
            {"group_id":"7bcc598f-d7a8-4299-8a1d-bdf918d48458","producer_pubkey":"CAISIQKLni9lu428QDeJTjlsjxWXAtaM5+paRIUiiRGgd/H+uw==","owner_pubkey":"CAISIQP1AJ5cNaoYMBqQV/LoYuIlkWqworXRZ3Q0xf9/3APbBQ==","sign":"304602210095368059515c5845050b7bf1b462988dc10e30c8d338b2ff7ff49d8104058e46022100e654475ed18ee8e65799c5b31abc50275d46b810194945aebcca7247894e03cf","trx_id":"074fe705-6585-43ba-a1d5-fa4b9ada3d89","memo":"","action":"add"}

            "group_id":group id
            "pubkey_announced": User pubkey
            "sign": 用户签名                
            "trx_id":trx id,
            "action": "add" or "del"
            "memo": memo

    - 查看组内 Announced userPubkey
        获得当前组内Announce过的Pubkey列表

        - 例子
        url -k https://127.0.0.1:8002/api/v1/group/7bcc598f-d7a8-4299-8a1d-bdf918d48458/announced/users

        - 参数
            group_id: group id

        - 返回值
            [{"UserPubkey":"age1jmnvu657vv62n0acujfjsq73ellqq8wygfxq2xazgycnap38lc8q6y7nfk"},{"UserPubkey":"age1nc0xrgg4uhf8w4kh7ee0znrv7h3248pnxqxf2wm9qm3nm87qxafqgupvhs"}]

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

            - POA (权威证明)

                权威证明共识要求组创建者必须时刻保持在线（或指定一个或一组producer），一个组中的节点，信任且仅信任组创建者（或producer），所有区块必须有组创建者（或producer）生产并签名，所有节点只同步组创建者（或producer）生产的区块，所有节点只接受来自组创建者（或producer）提供的block，如果组创建者（或所有producer均）不在线，则组无法正常工作，所有提交的trx都将超时（需客户端自行判断），所有组节点将处于SYNCING_FAILED和SYNCING之间的状态

                - 组生产者
                    在没有添加任何一个producer之前，组创建者是组内的生产者和内容提供者，一旦组创建者指定至少一个producer, 则组生产者变为普通组用户（仍保有组内权限），所有生产都由producer进行，producer之间会通过协议来保证最终一致性，因此所有producer都是可被信任的。当用户上线询问最新block时（ASK_NEXT)，只有producer会响应这个请求，并根据实际情况，给出有新块（BLOCK_IN_TRX）或者同步完成（ON_TOP)的回应。
                    如果组内所有producer均不能正常工作，则组无法正常工作
                
                -POA组出块流程如下
                    - Group Owner是组内第一个producer，也是权限最高的producer
                    - 如果Group Owner是唯一的producer，则所有块都要由他负责生产，也*必须*由他提供，因此如果owner不在线，或节点无法与Owner节点相连，则组或节点无法正常工作
                    - Group Owner可以添加producer节点到组内，也可以从组中删除某个producer
                    - 有其他producer存在的情况下，出块逻辑如下
                        1. 某组内节点产生/签名/广播一个trx（POST）
                        2. 该trx被所有在线的producer接受
                        3. 所有收到该trx的producer开始出块流程
                        4. producer启动出块定时器，等待10秒，继续收集该时间段内可能的其他trx，
                        5. producer启动等待定时器，等待40秒，开始准备接受其他Producer生产的Block
                        5. producer的出块时间到后，所有producer一起出块，将10秒内收集到的trx一起打包
                        7. producer在PRODUCER_CHANNEL广播自己刚刚生产的block
                        8. producer收集其他producer生产的区块，放到自己的<producer> <block>列表中，并按照Hash排序
                        9. 所有在线producer按照一个BFT算法对这个列表形成共识，并接受获胜的块 （TBD，是否将竞争结果写入块中？？？）
                        10. 本轮获胜的producer（排第一的produer）在USER_CHANNEL对所有组用户广播这个新block
                        11. 所有节点接受该块
                        14. 如果某节点发现这个块不能与自己的top block链接，则说明缺少某一（或某些）块，开始发送ask_next，试图从producer获得完整的block列表                        
                    - 仅有Group Owner存在的情况下（没有其他producer存在）出块逻辑：
                        1. 某组内节点产生/签名/广播一个trx（POST）
                        2. 该trx被Owner接受
                        3. Owner开始出块流程
                        4. Owner启动出块定时器，等待10秒，继续收集该时间段内可能的其他trx，
                        5. Owner的出块时间到后，将10秒内收集到的trx一起打包
                        6. Owner在USER_CHANNEL对所有组用户广播这个新block
                        7. 所有节点接受该块
                        8. 如果某节点发现这个块不能与自己的top block链接，则说明缺少某一（或某些）块，开始发送ask_next，试图从Owner获得完整的block列表 
    
            - POS (抵押共识）****修改中，并非最终版本****
                会被广播并有所有在线节点收集，并出发一轮出块流程，出块流程如下（伪码）

                If NOT IN PRODUCE ROUTINE
                START A ROUND OF CHALLENGE
                ELSE
                IF RECEIVE CHALLENGE ITEM FROM OTHER NODE {
                    SET STATUS TO *IN_PRODUCE*
                    SEND RESPONSE *ONLY ONCE*
                }                    

                WAIT 10S FOR INCOMING CHALLENGE RESPONSE
                WHEN TIME UP, SORT AND LOCK CHALLENGE RESPONSE TABLE

                REPEAT TILL PRODUCE DONE OR TIMEOUT OR RUN_OUT_OF CHALLENGE TABLE ITEMS{
                    IF I AM LUCKY
                        PRODUCE BLOCK
                    ELSE {
                        WAIT 5S INCOMING BLOCK
                        IF BLOCK COMES {
                            IF BLOCK IS VALID
                                PRODUCE_DONE
                            ELSE
                                REJECT AND CONTINUE
                        }
                        ELSE
                            UPDATE CHALLENGE TABLE INDEX
                    }                        
                }

                DO CLEANUP

                1. 节点1发起挑战（如果不在出块流程中），发送一个challenge trx
                2. 其他节点收到挑战请求，存储，产生并发送自己的挑战响应
                3. 每轮挑战10秒钟，在这10秒钟里，每个节点都会形成一个同样的挑战结果“榜单”，按照挑战结果的大小排序
                4. 挑战结束后，开始出块过程
                5. 每轮出块时间是5秒，在这五秒钟内，节点们检查挑战榜的顺序，如果本轮轮到自己，则出块，如果不是自己出块，则等待相应节点出块，如果时间到并没有等到相应的块，则进入下一轮出块过程
                6. 更新挑战榜index，开始新一轮出块过程
                7. 如果用尽挑战榜名单也没有出块，则说明本组内全体节点都下线，则出块失败，也即发送trx失败，客户端应重试
        
    - POST Trx状态判断
        
        同其他链相似，Trx的发送没有重试机制，客户端应自己保存并判断一个Trx的状态，具体过程如下

        1. 发送一个trx，获取trx_id

        curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","type":"Group"}}' http://127.0.0.1:8002/api/v1/group/content

        {"trx_id":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d"}

        2. 将这个trx标记为“发送中”
        3. 查询组内的内容

        curl -k -X GET -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' http://127.0.0.1:8002/api/v1/group/content
    
        [{"TrxId":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d","Publisher":"Qmbt56A7gVueThDVxfvLstxSR7BhE6M8doqxZXKWGBEbxT","Content":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"TimeStamp":1619656412253363059}]

        4. 设置一个超时，目前建议是30秒，不断查询，直到相同trx_id的内容出现在返回结果中，即可认为trx发送成功（被包含在块中）
        5. 如果超时被触发，没有查到结果，即认为发送trx失败，客户端可以自行处理重发

    - AUTH Trx状态判断
        
        大体流程同POST Trx状态判断，步骤3略有不同，需要查询当前的blacklist条目
        curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/group/blacklist
    
        {"blocked":[{"GroupId":"3171b27b-2241-41ca-b3a5-144d34ed5bee","UserId":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","OwnerPubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCsnOQxSxyeBQdbR3DVys2BNBo78QHkzuf7xCUwxu8Aizu1Xz7eC/7V0ISm//jUtx+wfGvA1n9F4Pi/tuVtpP7ysuETbflYFwn1HFmQkB2KAfpXBh9nPdz4ZpYxKRac6t38VPFLrRzHQZWlzyP0bYiLLGKc2oPlDqIlPDsxQWDA7pHAvHYd2SfUtiLRHvDKQvRmOk2IUcKJF0kWaVvok68Nn1+ihbxyF2kGzd02SdGe0W8qbdYFT9K/Sx4ed/qE+43dzhCbNh0fEBiNDeAHdsssZ+6HiSGSlPS1SSrlSazQUF9ZglrnRN6Jtx/ezqP25ZpMsHMFbYl8fgETkxQUp2gDpvrZ1sW2jJIdcuhUP0BCfbvcis+YkOosd0Map9Z+KN6MHAEcN+zwCtVvbWRJCs3u3VzyOOxZN7A/o4LHEAvM9eAWObWcxvlMZABncaTC4+9gYIUI5N9nJY6ETmDsUdL6B/9zCiXnXaOZDEhzg5AxAkEShqoUW5OOupk9Lm42g4PKLrBR/qhzGzJEyXWXp09xRV7SFpmUJP6KnKLKDnthMYsrKMVYuX5SwIBd4RSWVU9gm52eHUS/wNSbEp0WiiWe9lBHMje2dSoSUqfV9HXIf8AIDD37vq5aJsj1PgH8VuARgtmCHdPSngODUcU8f3J7t3WXys75njOptB9AcW2fWwIDAQAB","OwnerSign":"20e9863fddadac7846a5e6caa50dbb39483f8f33479ce0ecf3b7a02441b31a317647a8fd28ff171363d94ae3f31ebda6e2e5c9e915be340988ec3b4e77fce36143baa4797c48cb0b5a358aa995f59098eda7d8494c2c91146d6aca7b9c4e0ce2df88d0e371c7e2e7a43ef83a6a5e7fb2616aea6a45a940f2bd5d4fbdd95bb4b6518e1d4cc234a6ed76ae31265175317ce82255a61501f96f8292840642e67ac5d860484df3c1ff23ba08daa2ad4a49855e51ceab194e27b7c723b026ec0a19e3da3e53d62634ee59cbf1fa2445148afa94be8a114a7559268aac33c3d6ce102c69a978496da2c25e215593c2b856a90c75bdf2a83f39540ea0979716b2d45e19a14e8c95d655d3d82e8fd9f2814d16352efd188eeb3ca681a2b4b501d98d1be1a716b8bc37697cf2699f4d962a1fa38588a2f4b2163de1540a9e46572b185a16170fb4efb2a08a04374f70c06548f8883a4bc2e2e0d2eda3f82ed3e3492c2f422ff0f92f432015bd6a6e5ecc603dc8bdba97c21c6a8600a940722f09a4bd6e14a632a037e3ad5925c178b602755626c2a172fbaa038f5efe8e82cf6644fa310d4da95bdd4a639bbba034e4bff31860835d6ab7371b42abe6f9864393816ef855d375701c84ccd86894496723ead59f1a71866a3e38bf262f3db5936881bb0550257c22be0d04b49b32c6ab70a403bb182d02a299509983269df37be54d540f59","Memo":"Add","TimeStamp":1621532089763312100}]}

        可以看到，相关条目同样有trx_id字段，与POST Trx做同样处理即可
        
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

    - 手动发起同步block

        客户端可以手动触发某个组和组内其他节点同步块

        例子：

            curl -X POST -H 'Content-Type: application/json' -d '' http://<IP_ADDR>/api/v1/group/<GROUP_ID>/startsync

        参数:
            group_id, 需要发起同步的组的group_id

        返回值：
            200 ： {"GroupId":<GROUP_ID>,"Error":""}， GROUP_ID的组正常开始同步，同时组的状态会变为SYNCING
            500 ： {"GroupId":<GROUP_ID>,"Error":"GROUP_ALREADY_IN_SYNCING"}, GROUP_ID的组当前正在同步中


    - App API:

        Request content with senders filter

        curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[ "CAISIQP8dKlMcBXzqKrnQSDLiSGWH+bRsUCmzX42D9F41CPzag=="]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20" 

        Requst all content

        curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20" 

