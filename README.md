设置测试环境

Run testing

```go test cmd/main* -v```


本地开发环境配置

    1. 安装go
    2. 下载  https://github.com/huo-ju/quorum.git

    共需要3个本地节点，进入本地目录，例如 ~/work/quorum
    
    - 启动BootStrap节点 : 
        
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

            *这个返回的json串就是新创建组的“种子”，应该通过ATM提供的接口上传至PRS链，目前保存到文件中即可                

    - 查看节点A所拥有的组

        执行：
            curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/v1/group

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
            curl -X GET -H 'Content-Type: application/json' -d '{"group_id":"846011a8-1c58-4a35-b70f-83195c3bc2e8"}' http://127.0.0.1:8003/api/v1/group/content
        
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

        {"node_publickey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCk7Mwxj3gnM8TDCg92M3b2baGoLkk04I+piMoPPTwYjG57CTA9d4dxgFCHOWCa2M93P5+2Ap7w6pGhAPgpldEkFS8zrU4IpKk2Syqh163IDjVnhB2hmrQ2+ZdkXVNOdLTv3zdHR5+iUQAeM7Tp5mE1Sas3iWIJmJn3RPXV0+LV9Ugy/kMYnXMPDuggsTidGK8sjSQNK0ks+BCDrF37vnbiEBfyDJ42jMhvycxzUecTNrNVtUuElkuXsBa+hwPnnljQRCwi1aeTU0v5w6dOwZw1Lpx2FEKhxU5H5ti72XKGfmcU5KznkYsBkS15Xl8mx6Hu9VGq9KHmk5xSJZSi/Eu49RldcE1DGh1LKurxc5cKgy20kqoRU19b8Ba3z1I+0yeEAKBl8QWYaTbRovV2w+xABteKI+FQHfmGs4tkt+VUs1rhOTniqQ1qFwMDZX0ZcuK3GnnaLd+GvppXM1C1wqlpL+zv8LsSFSTQkCd1pz9SO2nz2UM047HIbn463DN89xrT3oIT2AiP50ozH0VoMZ1UAuHJfkQyJA0noh9t8CYjYYqe/N3sRgkY2L3P47TAzhqai0uJBEv22OJIx2Dt50eUhSUjP0UzkMssGxWLlqRZ0s+XrrLs6mTnE6zRj2+GqjFekj4thhSernXQuA3f1hqFnUYs3HIYScxCQvPVzTPZTQIDAQAB","node_status":"NODE_ONLINE",
        "node_version":"ver 0.01"}

        参数：
            node_publickey: 组创建者的pubkey
            node_status : 节点状态，值可以是 “NODE_ONLINE"和”NODE_OFFLINE“
            node_version : 节点的协议版本号

        - /api/v1/block (TBD)            
        - /api/v1/trx (TBD)

