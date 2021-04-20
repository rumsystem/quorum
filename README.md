设置测试环境

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
            curl -X POST -H 'Content-Type: application/json' -d '{"group_name":"my_test_group"}' http://127.0.0.1:8002/api/creategrp

            API：creategrp，创建新组
            参数： "group_name" string , 组名称，必填
       
        API返回值：

            {"genesis_block":
                    "{
                        \"Cid\":\"acdcf2c1-7add-4c70-af3b-8b1c29d5eafc\",\"GroupId\":\"3d18f604-4410-4db8-ac46-d28d6de0386f\",
                        \"PrevBlockId\":\"\",
                        \"BlockNum\":1,
                        \"Timestamp\":1618951803997806500,\"Hash\":\"7e261f2a678a796ff1b1a594164265a77ac4d8d3e2f473221837d8aeb671585c\",\"PreviousHash\":\"\",\"Producer\":\"QmfKAoHmfF1R8RBQqcKKjXf5dHx1m1oNvuT685mCVsUY5c\",\"Signature\":\"Signature from producer\",\"Trxs\":null
                    }",
            "group_id":"3d18f604-4410-4db8-ac46-d28d6de0386f",
            "group_name":"my_test_group","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCk7Mwxj3gnM8TDCg92M3b2baGoLkk04I+piMoPPTwYjG57CTA9d4dxgFCHOWCa2M93P5+2Ap7w6pGhAPgpldEkFS8zrU4IpKk2Syqh163IDjVnhB2hmrQ2+ZdkXVNOdLTv3zdHR5+iUQAeM7Tp5mE1Sas3iWIJmJn3RPXV0+LV9Ugy/kMYnXMPDuggsTidGK8sjSQNK0ks+BCDrF37vnbiEBfyDJ42jMhvycxzUecTNrNVtUuElkuXsBa+hwPnnljQRCwi1aeTU0v5w6dOwZw1Lpx2FEKhxU5H5ti72XKGfmcU5KznkYsBkS15Xl8mx6Hu9VGq9KHmk5xSJZSi/Eu49RldcE1DGh1LKurxc5cKgy20kqoRU19b8Ba3z1I+0yeEAKBl8QWYaTbRovV2w+xABteKI+FQHfmGs4tkt+VUs1rhOTniqQ1qFwMDZX0ZcuK3GnnaLd+GvppXM1C1wqlpL+zv8LsSFSTQkCd1pz9SO2nz2UM047HIbn463DN89xrT3oIT2AiP50ozH0VoMZ1UAuHJfkQyJA0noh9t8CYjYYqe/N3sRgkY2L3P47TAzhqai0uJBEv22OJIx2Dt50eUhSUjP0UzkMssGxWLlqRZ0s+XrrLs6mTnE6zRj2+GqjFekj4thhSernXQuA3f1hqFnUYs3HIYScxCQvPVzTPZTQIDAQAB",
            "signature":"owner_signature"}

            参数：
                genesis_block  新创建组的genesis block
                group_id    
                group_name
                owner_pubkey   新建组的Owner公钥
                signature      新建组owner对结果的签名 

            *这个返回的json串就是新创建组的“种子”，应该通过ATM提供的接口上传至PRS链，目前保存到文件中即可                

    - 查看节点A所拥有的组

        执行：
            curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/getgrps

            API : getgrps，返回节点所加入（含自己创建）的所有组
            参数 : 无

        API返回值：
            {"group_items":
                "[
                    {\"OwnerPubKey\":\"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCk7Mwxj3gnM8TDCg92M3b2baGoLkk04I+piMoPPTwYjG57CTA9d4dxgFCHOWCa2M93P5+2Ap7w6pGhAPgpldEkFS8zrU4IpKk2Syqh163IDjVnhB2hmrQ2+ZdkXVNOdLTv3zdHR5+iUQAeM7Tp5mE1Sas3iWIJmJn3RPXV0+LV9Ugy/kMYnXMPDuggsTidGK8sjSQNK0ks+BCDrF37vnbiEBfyDJ42jMhvycxzUecTNrNVtUuElkuXsBa+hwPnnljQRCwi1aeTU0v5w6dOwZw1Lpx2FEKhxU5H5ti72XKGfmcU5KznkYsBkS15Xl8mx6Hu9VGq9KHmk5xSJZSi/Eu49RldcE1DGh1LKurxc5cKgy20kqoRU19b8Ba3z1I+0yeEAKBl8QWYaTbRovV2w+xABteKI+FQHfmGs4tkt+VUs1rhOTniqQ1qFwMDZX0ZcuK3GnnaLd+GvppXM1C1wqlpL+zv8LsSFSTQkCd1pz9SO2nz2UM047HIbn463DN89xrT3oIT2AiP50ozH0VoMZ1UAuHJfkQyJA0noh9t8CYjYYqe/N3sRgkY2L3P47TAzhqai0uJBEv22OJIx2Dt50eUhSUjP0UzkMssGxWLlqRZ0s+XrrLs6mTnE6zRj2+GqjFekj4thhSernXQuA3f1hqFnUYs3HIYScxCQvPVzTPZTQIDAQAB\",\"GroupId\":\"3d18f604-4410-4db8-ac46-d28d6de0386f\",
                    \"GroupName\":\"my_test_group\",
                    \"LastUpdate\":1618951803997899600,
                    \"LatestBlockNum\":1,\"LatestBlockId\":\"acdcf2c1-7add-4c70-af3b-8b1c29d5eafc\",\"GroupStatus\":\"GROUP_READY\"}
                ]"
            }

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
        curl -X POST -H 'Content-Type: application/json' -d 
            '{"genesis_block":" 
                {
                    \"Cid\":\"acdcf2c1-7add-4c70-af3b-8b1c29d5eafc\",\"GroupId\":\"3d18f604-4410-4db8-ac46-d28d6de0386f\",\"PrevBlockId\":\"\",\"BlockNum\":1,\"Timestamp\":1618951803997806500,\"Hash\":\"7e261f2a678a796ff1b1a594164265a77ac4d8d3e2f473221837d8aeb671585c\",\"PreviousHash\":\"\",
                    \"Producer\":\"QmfKAoHmfF1R8RBQqcKKjXf5dHx1m1oNvuT685mCVsUY5c\",\"Signature\":\"Signature from producer\",
                    \"Trxs\":null
                }",

                "group_id":"3d18f604-4410-4db8-ac46-d28d6de0386f",
                "group_name":"my_test_group","owner_pubkey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCk7Mwxj3gnM8TDCg92M3b2baGoLkk04I+piMoPPTwYjG57CTA9d4dxgFCHOWCa2M93P5+2Ap7w6pGhAPgpldEkFS8zrU4IpKk2Syqh163IDjVnhB2hmrQ2+ZdkXVNOdLTv3zdHR5+iUQAeM7Tp5mE1Sas3iWIJmJn3RPXV0+LV9Ugy/kMYnXMPDuggsTidGK8sjSQNK0ks+BCDrF37vnbiEBfyDJ42jMhvycxzUecTNrNVtUuElkuXsBa+hwPnnljQRCwi1aeTU0v5w6dOwZw1Lpx2FEKhxU5H5ti72XKGfmcU5KznkYsBkS15Xl8mx6Hu9VGq9KHmk5xSJZSi/Eu49RldcE1DGh1LKurxc5cKgy20kqoRU19b8Ba3z1I+0yeEAKBl8QWYaTbRovV2w+xABteKI+FQHfmGs4tkt+VUs1rhOTniqQ1qFwMDZX0ZcuK3GnnaLd+GvppXM1C1wqlpL+zv8LsSFSTQkCd1pz9SO2nz2UM047HIbn463DN89xrT3oIT2AiP50ozH0VoMZ1UAuHJfkQyJA0noh9t8CYjYYqe/N3sRgkY2L3P47TAzhqai0uJBEv22OJIx2Dt50eUhSUjP0UzkMssGxWLlqRZ0s+XrrLs6mTnE6zRj2+GqjFekj4thhSernXQuA3f1hqFnUYs3HIYScxCQvPVzTPZTQIDAQAB",
             "signature":"owner_signature"
            }' http://127.0.0.1:8003/api/joingrp
        
        API：joingrp，加入一个组
        参数：组的“种子”json串（之前步骤的结果）

        返回值：{
                "group_id":"3d18f604-4410-4db8-ac46-d28d6de0386f",
                "signature":"Owner Signature"}

            group_id, 所加入组的id
            signature，加入组操作的签名

        节点B加入组my_test_group后，如果获取节点B的组信息，可以看到节点B中my_group先处于GROUP_TEST_SYNCING状态，在于节点A中的my_test_group同步后，即变为 GROUP_READY状态

    - 确定2个节点中的my_test_group都处于GROUP_READY状态

    - 节点A post to group
        执行：
            curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"3d18f604-4410-4db8-ac46-d28d6de0386f", "content":"some test contents"}' http://127.0.0.1:8002/api/posttogrp

            参数： 
                group_id : 组id
                content：发布的内容

        返回值：
            {"trx_id":"fc369ee1-a344-48fe-8fae-f2ee3551d327"}

            参数：
                trx_id: post的trx_id

    - 节点 B 查询新的刚刚获得内容
        
        执行:
            curl -X GET -H 'Content-Type: application/json' -d '{"group_id":"19199871-eeb4-4068-b2bb-44187509846a"}' http://127.0.0.1:8003/api/getgrpctn
        
        参数：
            group_id : 组id

        返回值:
            {"1618959433130449600":"some test contents"}

            参数：{时间戳} ：“内容”  

            *待完善

            返回应为包含数个如下结构的json数组

            type GroupContentItem struct {
	            TrxId     string    //trx_id
	            Publisher string    //发布者
	            Content   string    //内容
	            TimeStamp int64     
            }

    - 节点A也可以查询刚刚发布的内容

    其他API
        - leavegrp，离开一个组

        例子：
        curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"fcc2594e-9ad8-435a-bb85-d30bd51c84ba"}' http://127.0.0.1:8002/api/leavegrp

        参数
            group_id : 组id

        - rmgrp，删除一个组

        例子：
        curl -X POST -H 'Content-Type: application/json' -d '{"group_id":"f786bc94-f740-449b-9d3a-1a65e3afaf7b"}' http://127.0.0.1:8002/api/rmgrp

        参数
            group_id : 组id

        - getnodeinfo, 获取节点信息
        例子：
        curl -X GET -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8002/api/getnodeinfo

        返回值：

        {"node_publickey":"CAASpgQwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCk7Mwxj3gnM8TDCg92M3b2baGoLkk04I+piMoPPTwYjG57CTA9d4dxgFCHOWCa2M93P5+2Ap7w6pGhAPgpldEkFS8zrU4IpKk2Syqh163IDjVnhB2hmrQ2+ZdkXVNOdLTv3zdHR5+iUQAeM7Tp5mE1Sas3iWIJmJn3RPXV0+LV9Ugy/kMYnXMPDuggsTidGK8sjSQNK0ks+BCDrF37vnbiEBfyDJ42jMhvycxzUecTNrNVtUuElkuXsBa+hwPnnljQRCwi1aeTU0v5w6dOwZw1Lpx2FEKhxU5H5ti72XKGfmcU5KznkYsBkS15Xl8mx6Hu9VGq9KHmk5xSJZSi/Eu49RldcE1DGh1LKurxc5cKgy20kqoRU19b8Ba3z1I+0yeEAKBl8QWYaTbRovV2w+xABteKI+FQHfmGs4tkt+VUs1rhOTniqQ1qFwMDZX0ZcuK3GnnaLd+GvppXM1C1wqlpL+zv8LsSFSTQkCd1pz9SO2nz2UM047HIbn463DN89xrT3oIT2AiP50ozH0VoMZ1UAuHJfkQyJA0noh9t8CYjYYqe/N3sRgkY2L3P47TAzhqai0uJBEv22OJIx2Dt50eUhSUjP0UzkMssGxWLlqRZ0s+XrrLs6mTnE6zRj2+GqjFekj4thhSernXQuA3f1hqFnUYs3HIYScxCQvPVzTPZTQIDAQAB","node_status":"NODE_ONLINE",
        "node_version":"ver 0.01"}

        参数：
            node_publickey: 组创建者的pubkey
            node_status : 节点状态，值可以是 “NODE_ONLINE"和”NODE_OFFLINE“
            node_version : 节点的协议版本号

        - getblk (TBD)            
        - gettrx (TBD)

