![Main Test](https://github.com/huo-ju/quorum/actions/workflows/maintest.yml/badge.svg)


设置测试环境

Run testing

```go test cmd/main* -v```

API Docs

```go run cmd/docs.go```

Open url ```http://localhost:1323/swagger/index.html``` in the browser. 

本地开发环境配置

    1. 安装go
    2. 下载  https://github.com/huo-ju/quorum.git

    共需要3个本地节点，进入本地目录，例如 ~/work/quorum
    
    - 启动BootStrap节点 (mkdir -p config): 
        
        go run cmd/main.go -bootstrap -listen /ip4/0.0.0.0/tcp/10666 -logtostderr=true
        I0420 14:58:47.719592     332 keys.go:47] Load keys from config
        I0420 14:58:47.781916     332 main.go:64] Host created, ID:<QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR>, Address:<[/ip4/172.28.230.210/tcp/10666 /ip4/127.0.0.1/tcp/10666]>    
        记下ID <#ID>，例如 QmR1VFquywCnakSThwWQY6euj9sRBn3586LDUm5vsfCDJR
        
    - 启动本地节点A:
        go run cmd/main.go -peername peer2 -listen /ip4/127.0.0.1/tcp/7002 -apilisten :8002 -peer /ip4/127.0.0.1/tcp/10666/p2p/<#ID> -logtostderr=true -jsontracer jsontracer2.json    
        * 用bootstrap节点的ID替代 <#ID>
        * 节点A的本地HTTP端口地址为8002，以下所有curl命令中发给8002端口的API Call都是调用节点A

    - 启动本地节点B:
        go run cmd/main.go -peername peer3 -listen /ip4/127.0.0.1/tcp/7003 -apilisten :8003 -peer /ip4/127.0.0.1/tcp/10666/p2p/<#ID> -logtostderr=true -jsontracer jsontracer3.json
        * 用bootstrap节点的ID替代 <#ID>
        * 节点A的本地HTTP端口地址为8003，以下所有curl命令中发给8003端口的API Call都是调用节点B

本地API调用流程

    ** 注意，因为缺少链鉴权，请严格按照顺序进行 **
    ** API开发过程可参考本流程，即能用UI实现如下过程即可 **

    - 节点A创建组 "my_test_group"

        执行：
            curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group"}' http://127.0.0.1:8002/api/v1/group 

        API：/api/v1/group  ，创建新组
        参数： "group_name" string , 组名称，必填
       
        API返回值：

            {"genesis_block":{"Cid":"e5101baf-4fbb-49a9-8dc3-5c137fb918c7","GroupId":"846011a8-1c58-4a35-b70f-83195c3bc2e8","PrevBlockId":"","BlockNum":1,"Timestamp":1619075680398997998,"Hash":"d7b4c6bbe72967f092a9ed8460eb8deb39e22e6f9bdbe03f308bb5fe1ed507fc","PreviousHash":"","Producer":"QmeqL59zzQ8QkjGcPfEpVMd3MbsYDVoHTQuS34xEgwH6Bt","Signature":"Signature from producer","Trxs":null},"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","group_name":"my_test_group","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB","signature":"owner_signature"}
            

            参数：
                genesis_block  新创建组的genesis block
                group_id    
                group_name
                owner_pubkey   新建组的Owner公钥
                signature      新建组owner对结果的签名 

            *这个返回的json串就是新创建组的“种子”，保存到文件中

    - 查看节点A所拥有的组

        执行：
            curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/groups

            Method: GET
            Endpoint : /api/v1/group ，返回节点所加入（含自己创建）的所有组
            参数 : 无

        API返回值：

            {"groups":[{"OwnerPubKey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB","GroupId":"846011a8-1c58-4a35-b70f-83195c3bc2e8","GroupName":"","LastUpdate":1619075680399069448,"LatestBlockNum":1,"LatestBlockId":"e5101baf-4fbb-49a9-8dc3-5c137fb918c7","GroupStatus":"GROUP_READY"},{"OwnerPubKey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB","GroupId":"86e45bc9-238c-4b9b-8e27-5a3286319d71","GroupName":"","LastUpdate":1619075872254556107,"LatestBlockNum":1,"LatestBlockId":"6c857974-b480-4517-8c68-641db6fdfda5","GroupStatus":"GROUP_READY"}]}


            参数：
                group_items, group_info数组，group_info数组大部分于上步相同，GroupStatus用于显示组状态，
                该参数有2个值
                    - GROUP_READY
                    - GROUP_SYNCING
                这个参数可以用来显示组状态，详见设计文档，以上结果可见Node A处于Ready状态
        
        因为节点A刚刚创建my_test_group，没有人知道组的“种子”，该组肯定处于ready状态，无需等待同步
        如果这时退出节点A再重新启动，则节点A处于 GROUP_SYNCING 状态，等待与其他加入组的人同步，以获取组中（别人发的）新消息

    - 节点B加入组"my_test_group"

        执行：

        curl -X POST -H 'Content-Type: application/json' -d '{"genesis_block":{"Cid":"e5101baf-4fbb-49a9-8dc3-5c13
        7fb918c7","GroupId":"846011a8-1c58-4a35-b70f-83195c3bc2e8","PrevBlockId":"","BlockNum":1,"Timestamp":1619075680398997998,"Hash":"d7b4c6bbe72967f092a9ed8460eb8deb39e22e6f9bdbe03f308bb5fe1ed507fc","PreviousHash":"","Producer":"QmeqL59zzQ8QkjGcPfEpVMd3MbsYDVoHTQuS34xEgwH6Bt","Signature":"Signature from producer","Trxs":null},"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","group_name":"my_test_group","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB","signature":"owner_signature"}' http://127.0.0.1:8003/api/v1/group/join
        
        API：/v1/group/join ，加入一个组
        参数：组的“种子”json串（之前步骤的结果）

        返回值：
            {"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","signature":"Owner Signature"}

            group_id, 所加入组的id
            signature，加入组操作的签名

        节点B加入组my_test_group后，如果获取节点B的组信息，可以看到节点B中my_group先处于GROUP_TEST_SYNCING状态，在于节点A中的my_test_group同步后，即变为 GROUP_READY状态

    - 确定2个节点中的my_test_group都处于GROUP_READY状态

    - 节点A post to group
        执行：
            curl -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","type":"Group"}}' http://127.0.0.1:8002/api/v1/group/content

            参数： 
                group_id : 组id
                content：发布的内容

        返回值：
            {"trx_id":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d"}

            参数：
                trx_id: post的trx_id

    - 节点 B 查询新的刚刚获得内容
        
        执行:
            curl -X GET -H 'Content-Type: application/json' -d '' http://127.0.0.1:8003/api/v1/group846011a8-1c58-4a35-b70f-83195c3bc2e8/content
        
        参数：
            group_id : 组id

        返回值:
            [{"TrxId":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d","Publisher":"Qmbt56A7gVueThDVxfvLstxSR7BhE6M8doqxZXKWGBEbxT","Content":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"TimeStamp":1619656412253363059}]

            参数："时间戳" ："内容"
	            TrxId     string    //trx_id
	            Publisher string    //发布者
	            Content   string    //内容
	            TimeStamp int64      
            
            * 应按照时间戳对内容进行排序显示

    - 节点A也可以查询刚刚发布的内容

    其他API
        - /v1/group/leave ，离开一个组

        例子：

        curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' http://127.0.0.1:8002/api/v1/group/leave

        返回值:
        {"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","signature":"Owner Signature"}

        参数
            group_id : 组id

        - /api/v1/group ，删除一个组

        例子：
        curl -X DELETE -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' http://127.0.0.1:8003/api/v1/group

        返回值:
        {"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDGfeXQnKNIZpZeKDkj+FgAnVwBMa/GLjNrglsZduWWopYiPMBv9NEv59kG7JjYAaVZsWD5t2tgHPSYzHO6uc6DB9rphFprvx3dH8N/GDNh52J+FuS65Bw56sSIv/Y0e4D15WJsQV2/AQQvW90qpLTnRPY2VVUWiuETJQFDBvm1tmTRz+QOHQEqye0ogQSTtIvCdFcf3u7Bb2IAVTqFOIQlB98zwd1UNST9mkzgIYv3jIQ7lgA85/EC7v77J4Igin6/76aUgFbz++4f05Lns1nzvnGcorXSB7Dz//L+94Pyi00Y5ekN/YE3dk01PEr5Ucvi7bF1YfX2S+B4ruliZeTab05kysO5eKJF5Fd17YaEsIJb1d5kTXWL93/TJ5DkajLYmv9JGPjz78OUSMkz2FgS25hy4wIQpg0pP2We+mUoYK5B22FYdOuIropKq0VAzQeG/dFMAt7rFGNP8GLmQF0qV/KEE4xO3+kJdcWMDykMLdzOGwJzG9NHksIZPj4yxJP+jFdffZ9hHR0AuQlyCTg4Us13PTAYn6pTtwkvy0aS7J2Q8+IwNLuMJrfwjZYxTkdqJcvlck6+2IbLHYyBVi5TxT2zERB4Eg0iuJYq2VFWEkEWsUMtDda5G3jEI9yL/afjhVn6xmyo1D7aoeYqXqIx9Y/8jpRC4nN1wMfpsO+qdQIDAQAB","signature":"owner_signature"}

        参数
            group_id : 组id

        - /api/v1/node , 获取节点信息

        例子：
        curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8003/api/v1/node

        返回值：

        {"node_publickey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCyOIkTFxF1v8xborMo/6k45AMpfijbmT3OithJ/XTn8MDhnMw6j/jzw8YFSIfDj4KfjpwlyuVVZbSxHjeFFKMAWJgkeTNxRYLxXQWbZKd6d9PeRKLpdv/oEyDoPpdigMON84M1VWx9W0/lJ8Nps+cuI+7ugMLue40lAAUXXTPSaKy7vrgvQplKyfE4chPRY+bOAdmZDm76G00bGW6p4D2SLgApGXaG4grhGGvmJutAByIcaJRlpQu2mvgvjUAArP+YLw8scNvWzShGU/gz8tUFtus6c/cez/TmIUjeuD2hbbM+Gn1CxJxx/v0P59+hQT+f2NCM8yKC2KoXQkm5Llz2cUbJWbfOOQEkDCWRibYNEIUHYjWEL5xOcKLb4ie3vmJ5mz3kmI0iEDcx7OvTw7dtJGCo9GG5yPLITI0T3ygsjLUIpooY6PhOTIWvMqBVmiovUzb6cUb5Tms226KkP2ZOqNqqkwkN6zGI27ePdRde5N9N9zkwZd9ESaeOeea1BGDINyfpV1x2jk90BXRE7sB7f4eQrhCwtEHsoiZLUV4QevKO03XMMAGOmT6fQGACe6sVSeGfouNjKsgp0KrTRTtHIJCdGHNUNiv38ZGgRUWiwzPR83aJ24OJT2CNhLUvZk8tu5PagV19+4VKQ5OIOotHJusLvc1oibKCwv7sf6b2pQIDAQAB","node_status":"NODE_ONLINE","node_version":"ver 0.01","user_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m"}

        参数：
            node_publickey: 组创建者的pubkey
            node_status   : 节点状态，值可以是 “NODE_ONLINE"和”NODE_OFFLINE“
            node_version  : 节点的协议版本号
            node_id       : 节点的node_id

            *之前的user_id取消了(实际上是peer_id)
            *现在只返回真正的node_id，前端可以用pubkey来当user_id（唯一标识）

        - 获取一个块的完整内容
        例子：
            curl -X GET -H 'Content-Type: application/json' -d '{"block_id":"<BLOCK_ID>"}' http://127.0.0.1:8003/api/v1/block

        - 获取一个trx的完整内容
        例子:
            curl -X GET -H 'Content-Type: application/json' -d '{"trx_id":"<TRX_ID>"}' http://127.0.0.1:8003/api/v1/trx


        - 添加组黑名单
        
            例子：
            curl -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Auth", "id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m"},"target":{"id":"3171b27b-2241-41ca-b3a5-144d34ed5bee", "type":"Group"}}' http://127.0.0.1:8002/api/v1/group/blacklist
            
            参数： 
                type: "Add" 
                object : 
                    type：必须是"Auth"
                    id:想要屏蔽的user_id (可以通过节点信息API获得)
                target                                             
                    id:组id，哪个组想要屏蔽该用户
                    type：必须为"Group"

            说明：只有创建该组的节点才能执行此操作，也就是需要group_owner的权限，添加后会通过block广播至组中其他节点
                  注意：黑名单操作分为2种情况
                        1. 被组屏蔽的节点发出的block会被其他节点拒绝，因此无法向节点中发布内容，但是此时该节点仍可以获得组中其他节点的发出的新块（也即只要节点不退出，仍然可以看到新内容，但是不能发送）
                        2. 被组屏蔽的节点如果退出node，下次上线时将无法获取节点中最新的块，所以会一直处于syncing状态

            返回值：
                {"group_id":"3171b27b-2241-41ca-b3a5-144d34ed5bee","user_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCsnOQxSxyeBQdbR3DVys2BNBo78QHkzuf7xCUwxu8Aizu1Xz7eC/7V0ISm//jUtx+wfGvA1n9F4Pi/tuVtpP7ysuETbflYFwn1HFmQkB2KAfpXBh9nPdz4ZpYxKRac6t38VPFLrRzHQZWlzyP0bYiLLGKc2oPlDqIlPDsxQWDA7pHAvHYd2SfUtiLRHvDKQvRmOk2IUcKJF0kWaVvok68Nn1+ihbxyF2kGzd02SdGe0W8qbdYFT9K/Sx4ed/qE+43dzhCbNh0fEBiNDeAHdsssZ+6HiSGSlPS1SSrlSazQUF9ZglrnRN6Jtx/ezqP25ZpMsHMFbYl8fgETkxQUp2gDpvrZ1sW2jJIdcuhUP0BCfbvcis+YkOosd0Map9Z+KN6MHAEcN+zwCtVvbWRJCs3u3VzyOOxZN7A/o4LHEAvM9eAWObWcxvlMZABncaTC4+9gYIUI5N9nJY6ETmDsUdL6B/9zCiXnXaOZDEhzg5AxAkEShqoUW5OOupk9Lm42g4PKLrBR/qhzGzJEyXWXp09xRV7SFpmUJP6KnKLKDnthMYsrKMVYuX5SwIBd4RSWVU9gm52eHUS/wNSbEp0WiiWe9lBHMje2dSoSUqfV9HXIf8AIDD37vq5aJsj1PgH8VuARgtmCHdPSngODUcU8f3J7t3WXys75njOptB9AcW2fWwIDAQAB","sign":"20e9863fddadac7846a5e6caa50dbb39483f8f33479ce0ecf3b7a02441b31a317647a8fd28ff171363d94ae3f31ebda6e2e5c9e915be340988ec3b4e77fce36143baa4797c48cb0b5a358aa995f59098eda7d8494c2c91146d6aca7b9c4e0ce2df88d0e371c7e2e7a43ef83a6a5e7fb2616aea6a45a940f2bd5d4fbdd95bb4b6518e1d4cc234a6ed76ae31265175317ce82255a61501f96f8292840642e67ac5d860484df3c1ff23ba08daa2ad4a49855e51ceab194e27b7c723b026ec0a19e3da3e53d62634ee59cbf1fa2445148afa94be8a114a7559268aac33c3d6ce102c69a978496da2c25e215593c2b856a90c75bdf2a83f39540ea0979716b2d45e19a14e8c95d655d3d82e8fd9f2814d16352efd188eeb3ca681a2b4b501d98d1be1a716b8bc37697cf2699f4d962a1fa38588a2f4b2163de1540a9e46572b185a16170fb4efb2a08a04374f70c06548f8883a4bc2e2e0d2eda3f82ed3e3492c2f422ff0f92f432015bd6a6e5ecc603dc8bdba97c21c6a8600a940722f09a4bd6e14a632a037e3ad5925c178b602755626c2a172fbaa038f5efe8e82cf6644fa310d4da95bdd4a639bbba034e4bff31860835d6ab7371b42abe6f9864393816ef855d375701c84ccd86894496723ead59f1a71866a3e38bf262f3db5936881bb0550257c22be0d04b49b32c6ab70a403bb182d02a299509983269df37be54d540f59","trx_id":"4a890c84-4e8d-4706-bbdf-13c7cfd3ea98","memo":"Add"}

                group_id：组id
                user_id:  被屏蔽的用户id
                owner_pubkey: 组拥有者的pubkey
                sign: 组拥有者的签名（可通过pubkey验证）
                trx_id:该操作的trx的id，可以通过gettrx API获取具体内容
                memo: "Add"
        
        - 获取组黑名单 
            例子：
                curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/group/blacklist

            参数：
                无
            
            说明：获取一个节点的blacklist

            返回值：
                {"blocked":[{"GroupId":"3171b27b-2241-41ca-b3a5-144d34ed5bee","UserId":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","OwnerPubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCsnOQxSxyeBQdbR3DVys2BNBo78QHkzuf7xCUwxu8Aizu1Xz7eC/7V0ISm//jUtx+wfGvA1n9F4Pi/tuVtpP7ysuETbflYFwn1HFmQkB2KAfpXBh9nPdz4ZpYxKRac6t38VPFLrRzHQZWlzyP0bYiLLGKc2oPlDqIlPDsxQWDA7pHAvHYd2SfUtiLRHvDKQvRmOk2IUcKJF0kWaVvok68Nn1+ihbxyF2kGzd02SdGe0W8qbdYFT9K/Sx4ed/qE+43dzhCbNh0fEBiNDeAHdsssZ+6HiSGSlPS1SSrlSazQUF9ZglrnRN6Jtx/ezqP25ZpMsHMFbYl8fgETkxQUp2gDpvrZ1sW2jJIdcuhUP0BCfbvcis+YkOosd0Map9Z+KN6MHAEcN+zwCtVvbWRJCs3u3VzyOOxZN7A/o4LHEAvM9eAWObWcxvlMZABncaTC4+9gYIUI5N9nJY6ETmDsUdL6B/9zCiXnXaOZDEhzg5AxAkEShqoUW5OOupk9Lm42g4PKLrBR/qhzGzJEyXWXp09xRV7SFpmUJP6KnKLKDnthMYsrKMVYuX5SwIBd4RSWVU9gm52eHUS/wNSbEp0WiiWe9lBHMje2dSoSUqfV9HXIf8AIDD37vq5aJsj1PgH8VuARgtmCHdPSngODUcU8f3J7t3WXys75njOptB9AcW2fWwIDAQAB","OwnerSign":"20e9863fddadac7846a5e6caa50dbb39483f8f33479ce0ecf3b7a02441b31a317647a8fd28ff171363d94ae3f31ebda6e2e5c9e915be340988ec3b4e77fce36143baa4797c48cb0b5a358aa995f59098eda7d8494c2c91146d6aca7b9c4e0ce2df88d0e371c7e2e7a43ef83a6a5e7fb2616aea6a45a940f2bd5d4fbdd95bb4b6518e1d4cc234a6ed76ae31265175317ce82255a61501f96f8292840642e67ac5d860484df3c1ff23ba08daa2ad4a49855e51ceab194e27b7c723b026ec0a19e3da3e53d62634ee59cbf1fa2445148afa94be8a114a7559268aac33c3d6ce102c69a978496da2c25e215593c2b856a90c75bdf2a83f39540ea0979716b2d45e19a14e8c95d655d3d82e8fd9f2814d16352efd188eeb3ca681a2b4b501d98d1be1a716b8bc37697cf2699f4d962a1fa38588a2f4b2163de1540a9e46572b185a16170fb4efb2a08a04374f70c06548f8883a4bc2e2e0d2eda3f82ed3e3492c2f422ff0f92f432015bd6a6e5ecc603dc8bdba97c21c6a8600a940722f09a4bd6e14a632a037e3ad5925c178b602755626c2a172fbaa038f5efe8e82cf6644fa310d4da95bdd4a639bbba034e4bff31860835d6ab7371b42abe6f9864393816ef855d375701c84ccd86894496723ead59f1a71866a3e38bf262f3db5936881bb0550257c22be0d04b49b32c6ab70a403bb182d02a299509983269df37be54d540f59","Memo":"Add","TimeStamp":1621532089763312100}]}

                blocked：数组，包含该节点所有已经屏蔽的组-用户对
                    GroupId:组id
                    UserId:用户id
                    OwnerPubkey：组拥有者pubkey
                    OwnerSign:执行该操作的签名
                    Memo: Add or Remove
                    Timestamp：操作执行的时间戳

        - 删除组黑名单
        
            例子：
            curl -X POST -H 'Content-Type: application/json' -d '{"type":"Remove","object":{"type":"Auth", "id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m"},"target":{"id":"3171b27b-2241-41ca-b3a5-144d34ed5bee", "type":"Group"}}' http://127.0.0.1:8002/api/v1/group/blacklist

            参数： 
            type: "Remove"
            object : 
                type：必须是"Auth"
                id:  想要解除的user_id (可以通过节点信息API获得)
            target                                             
                id:组id
                type：必须为"Group"

            结果：
                {"group_id":"3171b27b-2241-41ca-b3a5-144d34ed5bee","user_id":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCsnOQxSxyeBQdbR3DVys2BNBo78QHkzuf7xCUwxu8Aizu1Xz7eC/7V0ISm//jUtx+wfGvA1n9F4Pi/tuVtpP7ysuETbflYFwn1HFmQkB2KAfpXBh9nPdz4ZpYxKRac6t38VPFLrRzHQZWlzyP0bYiLLGKc2oPlDqIlPDsxQWDA7pHAvHYd2SfUtiLRHvDKQvRmOk2IUcKJF0kWaVvok68Nn1+ihbxyF2kGzd02SdGe0W8qbdYFT9K/Sx4ed/qE+43dzhCbNh0fEBiNDeAHdsssZ+6HiSGSlPS1SSrlSazQUF9ZglrnRN6Jtx/ezqP25ZpMsHMFbYl8fgETkxQUp2gDpvrZ1sW2jJIdcuhUP0BCfbvcis+YkOosd0Map9Z+KN6MHAEcN+zwCtVvbWRJCs3u3VzyOOxZN7A/o4LHEAvM9eAWObWcxvlMZABncaTC4+9gYIUI5N9nJY6ETmDsUdL6B/9zCiXnXaOZDEhzg5AxAkEShqoUW5OOupk9Lm42g4PKLrBR/qhzGzJEyXWXp09xRV7SFpmUJP6KnKLKDnthMYsrKMVYuX5SwIBd4RSWVU9gm52eHUS/wNSbEp0WiiWe9lBHMje2dSoSUqfV9HXIf8AIDD37vq5aJsj1PgH8VuARgtmCHdPSngODUcU8f3J7t3WXys75njOptB9AcW2fWwIDAQAB","sign":"20e9863fddadac7846a5e6caa50dbb39483f8f33479ce0ecf3b7a02441b31a317647a8fd28ff171363d94ae3f31ebda6e2e5c9e915be340988ec3b4e77fce36143baa4797c48cb0b5a358aa995f59098eda7d8494c2c91146d6aca7b9c4e0ce2df88d0e371c7e2e7a43ef83a6a5e7fb2616aea6a45a940f2bd5d4fbdd95bb4b6518e1d4cc234a6ed76ae31265175317ce82255a61501f96f8292840642e67ac5d860484df3c1ff23ba08daa2ad4a49855e51ceab194e27b7c723b026ec0a19e3da3e53d62634ee59cbf1fa2445148afa94be8a114a7559268aac33c3d6ce102c69a978496da2c25e215593c2b856a90c75bdf2a83f39540ea0979716b2d45e19a14e8c95d655d3d82e8fd9f2814d16352efd188eeb3ca681a2b4b501d98d1be1a716b8bc37697cf2699f4d962a1fa38588a2f4b2163de1540a9e46572b185a16170fb4efb2a08a04374f70c06548f8883a4bc2e2e0d2eda3f82ed3e3492c2f422ff0f92f432015bd6a6e5ecc603dc8bdba97c21c6a8600a940722f09a4bd6e14a632a037e3ad5925c178b602755626c2a172fbaa038f5efe8e82cf6644fa310d4da95bdd4a639bbba034e4bff31860835d6ab7371b42abe6f9864393816ef855d375701c84ccd86894496723ead59f1a71866a3e38bf262f3db5936881bb0550257c22be0d04b49b32c6ab70a403bb182d02a299509983269df37be54d540f59","trx_id":"41204fd0-f9cb-497f-a62e-0d43d755a5b9","memo":"Remove"}

                group_id：组id
                user_id:  移除黑名单的用户id
                owner_pubkey: 组拥有者的pubkey
                sign: 组拥有者的签名（可通过pubkey验证）
                trx_id:该操作的trx的id，可以通过gettrx API获取具体内容
                memo: "Remove"

        - Trx生命周期和出块过程
            所有链上操作均是Trx，客户端相关的Trx有2种
                - POST 发送组内信息
                - AUTH 调整组内权限
            一个Trx被push到链上后，根据group的不同共识类型，将被采取不同形式对待

            - POA (权威证明)
                权威证明共识要求组创建者必须时刻保持在线，一个组中的节点，信任且仅信任组创建者，所有区块必须有组创建者生产并签名，
                所有节点只同步组由创建者生产的区块（可以由其他节点提供），如果组创建者不在线，则组无法正常工作，所有提交的trx都将超时（需客户端自行判断）
                节点启动时，组创建者不会与其他节点同步（因为他应该是一个组的最终内容存储者和生产者），而非所有者节点则会互相同步区块，直到获得ON_TOP消息
        
            - POS (抵押共识)
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

            curl -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"target":{"id":"846011a8-1c58-4a35-b70f-83195c3bc2e8","type":"Group"}}' http://127.0.0.1:8002/api/v1/group/content

            {"trx_id":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d"}

            2. 将这个trx标记为“发送中”
            3. 查询组内的内容

            curl -X GET -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' http://127.0.0.1:8002/api/v1/group/content
        
            [{"TrxId":"f73c94a0-2bb9-4d19-9efc-c9f1f7e87b1d","Publisher":"Qmbt56A7gVueThDVxfvLstxSR7BhE6M8doqxZXKWGBEbxT","Content":{"type":"Note","content":"simple note by aa","name":"A simple Node id1"},"TimeStamp":1619656412253363059}]

            4. 设置一个超时，目前建议是20秒，不断查询，直到相同trx_id的内容出现在返回结果中，即可认为trx发送成功（被包含在块中）
            5. 如果超时被触发，没有查到结果，即认为发送trx失败，客户端可以自行处理重发

        - AUTH Trx状态判断
            
            大体流程同POST Trx状态判断，步骤3略有不同，需要查询当前的blacklist条目
            curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/group/blacklist
        
            {"blocked":[{"GroupId":"3171b27b-2241-41ca-b3a5-144d34ed5bee","UserId":"QmQZcijmay86LFCDFiuD8ToNhZwCYZ9XaNpeDWVWWJYt7m","OwnerPubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCsnOQxSxyeBQdbR3DVys2BNBo78QHkzuf7xCUwxu8Aizu1Xz7eC/7V0ISm//jUtx+wfGvA1n9F4Pi/tuVtpP7ysuETbflYFwn1HFmQkB2KAfpXBh9nPdz4ZpYxKRac6t38VPFLrRzHQZWlzyP0bYiLLGKc2oPlDqIlPDsxQWDA7pHAvHYd2SfUtiLRHvDKQvRmOk2IUcKJF0kWaVvok68Nn1+ihbxyF2kGzd02SdGe0W8qbdYFT9K/Sx4ed/qE+43dzhCbNh0fEBiNDeAHdsssZ+6HiSGSlPS1SSrlSazQUF9ZglrnRN6Jtx/ezqP25ZpMsHMFbYl8fgETkxQUp2gDpvrZ1sW2jJIdcuhUP0BCfbvcis+YkOosd0Map9Z+KN6MHAEcN+zwCtVvbWRJCs3u3VzyOOxZN7A/o4LHEAvM9eAWObWcxvlMZABncaTC4+9gYIUI5N9nJY6ETmDsUdL6B/9zCiXnXaOZDEhzg5AxAkEShqoUW5OOupk9Lm42g4PKLrBR/qhzGzJEyXWXp09xRV7SFpmUJP6KnKLKDnthMYsrKMVYuX5SwIBd4RSWVU9gm52eHUS/wNSbEp0WiiWe9lBHMje2dSoSUqfV9HXIf8AIDD37vq5aJsj1PgH8VuARgtmCHdPSngODUcU8f3J7t3WXys75njOptB9AcW2fWwIDAQAB","OwnerSign":"20e9863fddadac7846a5e6caa50dbb39483f8f33479ce0ecf3b7a02441b31a317647a8fd28ff171363d94ae3f31ebda6e2e5c9e915be340988ec3b4e77fce36143baa4797c48cb0b5a358aa995f59098eda7d8494c2c91146d6aca7b9c4e0ce2df88d0e371c7e2e7a43ef83a6a5e7fb2616aea6a45a940f2bd5d4fbdd95bb4b6518e1d4cc234a6ed76ae31265175317ce82255a61501f96f8292840642e67ac5d860484df3c1ff23ba08daa2ad4a49855e51ceab194e27b7c723b026ec0a19e3da3e53d62634ee59cbf1fa2445148afa94be8a114a7559268aac33c3d6ce102c69a978496da2c25e215593c2b856a90c75bdf2a83f39540ea0979716b2d45e19a14e8c95d655d3d82e8fd9f2814d16352efd188eeb3ca681a2b4b501d98d1be1a716b8bc37697cf2699f4d962a1fa38588a2f4b2163de1540a9e46572b185a16170fb4efb2a08a04374f70c06548f8883a4bc2e2e0d2eda3f82ed3e3492c2f422ff0f92f432015bd6a6e5ecc603dc8bdba97c21c6a8600a940722f09a4bd6e14a632a037e3ad5925c178b602755626c2a172fbaa038f5efe8e82cf6644fa310d4da95bdd4a639bbba034e4bff31860835d6ab7371b42abe6f9864393816ef855d375701c84ccd86894496723ead59f1a71866a3e38bf262f3db5936881bb0550257c22be0d04b49b32c6ab70a403bb182d02a299509983269df37be54d540f59","Memo":"Add","TimeStamp":1621532089763312100}]}

            可以看到，相关条目同样有trx_id字段，与POST Trx做同样处理即可

            
	- 节点网络信息
	
	curl http://localhost:8002/api/v1/network
	
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

    
    - Update Profile    
    Update Users Profile . 
    Post a Person Object to the target Group.
      
    Update mixin Payment id
    ```curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Update","person":{"wallet":[ {"id":"2222-2222-2222","type":"mixin","name":"mixin messenger"} ]},"target":{"id":"b1b60370-9e96-41fb-84cc-4546d529ccc6","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/profile```

    Update Profile Name
    ```curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Update","person":{"name":"HuoJu"},"target":{"id":"b1b60370-9e96-41fb-84cc-4546d529ccc6","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/profile```

    Update Profile Avatar
    ```curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Update","person":{"name":"Rob","image":{"mediaType":"image/png", "content":"" }},"target":{"id":"b1b60370-9e96-41fb-84cc-4546d529ccc6","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/profile```

    content is the image with  base64 encoding.

    - Reply Object
    Reply to a Object with trxid

    ```curl -k -X POST -H 'Content-Type: application/json' -d '{"type":"Add","object":{"type":"Note","content":"reply to note a","name":" A TEST simple", "inreplyto":{ "trxid":"4308db96-56a7-4c0c-bde9-de182b0e6b6a"} },"target":{"id":"b1b60370-9e96-41fb-84cc-4546d529ccc6","type":"Group"}}' https://127.0.0.1:8002/api/v1/group/content```



App API:

    Request content with senders filter

    curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[ "CAISIQP8dKlMcBXzqKrnQSDLiSGWH+bRsUCmzX42D9F41CPzag=="]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20" 

   Requst all content

   curl -v -X POST -H 'Content-Type: application/json' -d '{"senders":[]}' "http://localhost:8002/app/api/v1/group/5a3224cc-40b0-4491-bfc7-9b76b85b5dd8/content?start=0&num=20" 


