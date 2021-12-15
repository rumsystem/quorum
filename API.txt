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

    - *** 删除组API以废除，所有节点只能“离开”一个组，不管是不是自己创建的 ***

    - /api/vi/group/clear，删除一个组的全部内容，包括如下内容
        - block
        - trx
        - announced
        - scheam
        - denied_list
        - post
        - producer
        
        例子：
           curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"13a25432-b791-4d17-a52f-f69266fc3f18"}' https://127.0.0.1:8002/api/v1/group/clear | jq

        参数：
            group_id

        返回值： 
            {
             "group_id": "13a25432-b791-4d17-a52f-f69266fc3f18",
            "signature": "30450221009634af1636bf7374453cd73088ff992d9020777eb617795e3c93ea5d5008f56d022035342a852e87afa87b5e038147dedf10bb847f60808ec78a470b92dfbff91504"
        }

    *** 目前前端在离开组时需一起调用该API，清除所有组相关的数据，警告用户“如果离开组，所有数据将被删除，再加入需要重新同步”即可 *** 

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
    
            {
                "group_id":"f4273294-2792-4141-80ba-687ce706bc5b","peer_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m",
                "owner_pubkey":"CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==","sign":"30460221008d7480261a3a33f552b268429a08f8b0ede03b4ddc8014d470d84e707a80d600022100b1616d4662f3e7f0c7594381e425e0c26cf25b66a2cef9437d320cccb0871e5b",
                "trx_id":"2f434ac3-c2a8-494a-9c58-d03a8b51dab5",
                "action":"add",
                "memo":""
            }


            group_id：组id
            peer_id:  被屏蔽的用户id
            owner_pubkey: 组拥有者的pubkey
            sign: 组拥有者的签名（可通过pubkey验证）
            trx_id:该操作的trx的id，可以通过gettrx API获取具体内容
            memo: "Add"
    
    - 获取组黑名单 

        例子：

            curl -k -X GET -H 'Content-Type: application/json' https://127.0.0.1:8002/api/v1/group/:group_id/deniedlist

        参数：
            无
        
        说明：获取一个节点禁止访问名单列表

        返回值：
        [
            {
                "GroupId":"f4273294-2792-4141-80ba-687ce706bc5b","PeerId":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJY111","GroupOwnerPubkey":"CAISIQMOjdI2nmRsvg7de3phG579MvqSDkn3lx8TEpiY066DSg==","GroupOwnerSign":"3046022100c2c07b0b03ea5a624dbe07b2cb30ad08a5282a017b873c8defbec9656ae4f8da022100a3659f8410151c811ee331de9cbdf719ec9db33170a95dddfe2c443ace36f3c3",
                "TimeStamp":1632514808574721034,
                "Action":"add",
                "Memo":""
            }
        ]
            
        数组，包含该组已经被Owner屏幕的用户id列表
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
                            "Action" : "Add", *
                            "TimeStamp": 1634756064250457600
                        }
                    ]
                参数：
                    AnnouncedPubkey ： producer pubkey
                    AnnouncerSign： producer的签名
                    Result ： ANNOUNCED or APPROVED，producer刚Announce完毕的状态是ANNOUNCED
                    Action : "ADD" or "REMOVE" 
                    TimeStamp : timestamp

                *ACTION 可以有2种状态，“ADD”表示Producer正常，“REMOVE”表示Producer已经announce自己离开改组

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
            * 请注意，Owner只可以选择在组内Announce过自己的Producer，并且producer的状态应该为“ADD”，没有Announce过的producer是不可以添加的        
        
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
                        "Action": "ADD"
                        "TimeStamp": 1634756064250457600
                    }
                ]
            
            可以看出，经过Owner批准，该Producer的状态（result)变为 APPROVED

        9. Owenr删除组内Producer
           例：curl -k -X POST -H 'Content-Type: application/json' -d '{"producer_pubkey":"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==","group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"remove"}}' https://127.0.0.1:8002/api/v1/group/producer | jq

           *Owner可以随时删除一个Producer, 不管Producer是否Announce离开
           *在实际环境中，Producer完全可以不Announce Remove而直接离开，Owner需要注意到并及时将该Producer从Producer列表中删除

    - Private Group
        1. Announce user's encrypt pubkey to a group
           Example: curl -k -X POST -H 'Content-Type: application/json' -d '{"group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add", "type":"user", "memo":"invitation code:a423b3"}' https://127.0.0.1:8003/api/v1/group/announce | jq

           API：/v1/group/announce
           Params：
               group_id：group id
               action ： add or remove
               type   :  user
               memo   ： memo
           Result：
               {
                 "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
                 "sign_pubkey": "CAISIQJwgOXjCltm1ijvB26u3DDroKqdw1xjYF/w1fBRVdScYQ==",
                 "encrypt_pubkey": "age1fx3ju9a2f3kpdh76375dect95wmvk084p8wxczeqdw8q2m0jtfks2k8pm9",
                 "type": "AS_USER",
                 "action": "ADD",
                 "sign": "304402206a68e3393f4382c9978a19751496e730de94136a15ab77e30bab2f184bcb5646022041a9898bb5ff563a6efeea29b30bac4bebf0d3464eb326fd84322d98919b3715",
                 "trx_id": "8a4ae55d-d576-490a-9b9a-80a21c761cef"
               }
           Params：
               group_id : group id
               sign_pubkey : user's sign pubkey
               encrypt_pubkey : user's encrypt pubkey
               type: AS_USER
               action: ADD
               sign: user's signature
               trx_id : trx_id
        2. view announced users
           Example ：curl -k -X GET -H 'Content-Type: application/json' -d '' https://127.0.0.1:8002/api/v1/group/5ed3f9fe-81e2-450d-9146-7a329aac2b62/announced/users

           API: /v1/group/{group_id}/announced/users
           Params：
               group_id : group id
           Result：
               [
                   {
                     "AnnouncedSignPubkey": "CAISIQIWQX/5Nmy2/YoBbdO9jn4tDgn22prqOWMYusBR6axenw==",
                     "AnnouncedEncryptPubkey": "age1a68u5gafkt3yfsz7pr45j5ku3tyyk4xh9ydp3xwpaphksz54kgns99me0g",
                     "AnnouncerSign": "30450221009974a5e0f3ea114de8469a806894410d12b5dc5d6d7ee21e49b5482cb062f1740220168185ad84777675ba29773942596f2db0fa5dd810185d2b8113ac0eaf4d7603",
                     "Result": "ANNOUNCED"
                   },
               ]
           Params：
               AnnouncedPubkey ： user's pubkey
               AnnouncerSign： user's signture
               Result ： ANNOUNCED or APPROVED

        3. Owner approve a users
            Example：curl -k -X POST -H 'Content-Type: application/json' -d '{"user_pubkey":"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==","group_id":"5ed3f9fe-81e2-450d-9146-7a329aac2b62", "action":"add"}}' https://127.0.0.1:8002/api/v1/group/user | jq

            API: /v1/group/user
            参数:
                "action":"add" // add or remove
                "user_pubkey": user public key
                "group_id": group id 
                "memo" : optional
            返回值：
            {    
                "group_id": "5ed3f9fe-81e2-450d-9146-7a329aac2b62",
                "user_pubkey": "CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==",
                "owner_pubkey": "CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA==",
                "sign": "304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea",
                "trx_id": "6bff5556-4dc9-4cb6-a595-2181aaebdc26",
                "memo": "",
                "action": "ADD"
            }
            参数：
                group_id: group id
                user_pubkey : public key for user just added 
                owner_pubkey: group owner public key                
                sign: signature
                trx_id: trx id
                action : Add or REMOVE
                memo : memo

    - 添加组内配置（GroupConfig）
        添加组内配置项
        例子：curl -k -X POST -H 'Content-Type: application/json' -d '{"action":"add", "group_id":"c8795b55-90bf-4b58-aaa0-86d11fe4e16a", "name":"test_bool", "type":"int", "value":"false", "memo":"add test_bool to group"}' https://127.0.0.1:8002/api/v1/group/config | jq
        API：/v1/group/config
        参数：
            action : add or del
            group_id : group id
            name : 配置项的名称
            type : 配置项的类型，可选值为 "int", "bool", "string"
            value ： 配置项的值，必须与type相对应
            memo : memo
        权限：
            只有group owner可以调用该API
        调用后，通过块同步，组内所有节点获得该配置            

        返回值：
            {
                "group_id": "c8795b55-90bf-4b58-aaa0-86d11fe4e16a",
                "sign": "3045022100e1375e48cfbd51cb78afc413fcca084deae9eb7f8454c54832feb9ae00fada7702203ee6fe2292ea3a87d687ae3369012b7518010e555b913125b8a7bf54f211502a",
                "trx_id": "9e54c173-c1dd-429d-91fa-a6b43c14da77"
            }
        参数：
            group_id : group id
            sign ： owner对该trx的签名
            trx_id : trx_id
    - 查看组内配置key列表
        例子：curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/group/c8795b55-90bf-4b58-aaa0-86d11fe4e16a/config/keylist
        API：/v1/group/<GROUP_ID>/config/keylist
        返回值：
            [{"Name":"test_string","Type":"STRING"}]
        参数：
            name：配置项的名称
            type: 配置项的数据类型
    - 查看组内某个配置的具体值
        例子：curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/group/c8795b55-90bf-4b58-aaa0-86d11fe4e16a/config/test_string | jq
        API：/v1/group/<GROUPID>/config/<KEY_NAME>
        返回值：
            {
                "Name": "test_string",
                "Type": "STRING",
                "Value": "123",
                "OwnerPubkey": "CAISIQJOfMIyaYuVpzdeXq5p+ku/8pSB6XEmUJfHIJ3A0wCkIg==",
                "OwnerSign": "304502210091dcc8d8e167c128ef59af1b6e2b2efece499043cc149014303b932485cde3240220427f81f2d7482df0d9a4ab2c019528b33776c73daf21ba98921ee6ff4417b1bc",
                "Memo": "add test_string to group",
                "TimeStamp": 1639518490895535600
            }
        参数：
            同添加组内配置

    - 添加组内App Schema
        添加组内app的schema json
        例子：curl -k -X POST -H 'Content-Type: application/json' -d '{"rule":"new_schema","type":"schema_type", "group_id":"13a25432-b791-4d17-a52f-f69266fc3f18", "action":"add", "memo":"memo"}' https://127.0.0.1:8002/api/v1/group/schema

    - 查看组内App Schema
        curl -k -X GET -H 'Content-Type: application/json' -d '{}' https://127.0.0.1:8002/api/v1/group/13a25432-b791-4d17-a52f-f69266fc3f18/app/schema | jq

        返回值：
            [
                {
                    "Type": "schema_type",
                    "Rule": "new_schema",
                    "TimeStamp": 1636047963013888300
                }
            ]

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

