import requests
import time
import json


#create group 
heads = {'Content-Type': 'application/json'}
url_create_group = 'http://127.0.0.1:8002/api/v1/group'
url_get_groups =  'http://127.0.0.1:8002/api/v1/groups'
url_post_to_group = 'http://127.0.0.1:8002/api/v1/group/content/false'

payload_create_group = {
  "group_name": "my_test_group",
  "consensus_type": "poa",
  "encryption_type": "public",
  "app_key": "test_app"
}

#response = requests.post(url_create_group, headers=heads, json=payload_create_group)
#jsonResp = response.json()
#group_id = jsonResp["group_id"]
#respString = "Create Group with groupId <%s>" % group_id
#print(respString)

TRX_COUNT = 100
group_id = 'ef0c809c-2eab-41a2-87fe-ec4de7b5a855'
trx_id_list = []
#try post 10000 trxs and verify
for i in range (0, TRX_COUNT):
    payload_post_to_group = {
        "type":"Add",
        "object":{"type":"Note",
                "content":"10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
                "name":"A simple Node id1"},
        "target":{"id": group_id,
                "type":"Group"}              
    }
    
    response = requests.post(url_post_to_group, headers=heads, json=payload_post_to_group)
    trx_id = response.json()['trx_id']
    a = "Post with trxId <%s>" % trx_id
    trx_id_list.append(trx_id)
    print(a) 



# #wait 2s
# time.sleep(10)  
# resp = requests.get(url_get_groups, headers=heads)
# jsonResp = resp.json()["groups"]
# group_info_json = jsonResp[0]
# highest_epoch = group_info_json["epoch"]
# print("Highest epoch:", highest_epoch)

# for epoch in range (1, highest_epoch):
#     print("Verify epoch <", epoch,  "> on o1")    
#     url_get_block = "http://127.0.0.1:8002/api/v1/block/%s/%d" % (group_id, epoch)
#     resp = requests.get(url_get_block, headers=heads)
#     blockepoch = resp.json()["Epoch"]
#     if  blockepoch!= epoch:
#         print("XXXXXXXXXXXXXXXXXX")
#         print("Get block failed")
#         print(resp.text)
#         quit()            
    
# for epoch in range (1, highest_epoch):
#     print("Verify epoch <", epoch,  "> on p1")        
#     url_get_block = "http://127.0.0.1:8003/api/v1/block/%s/%d" % (group_id, epoch)
#     resp = requests.get(url_get_block, headers=heads)
#     blockepoch = resp.json()["Epoch"]
#     if  blockepoch!= epoch:
#         print("XXXXXXXXXXXXXXXXXX")
#         print("Get block failed")
#         print(resp.text)
#         quit()
    
# for epoch in range (1, highest_epoch):
#     print("Verify epoch <", epoch,  "> on p2")    
#     url_get_block = "http://127.0.0.1:8004/api/v1/block/%s/%d" % (group_id, epoch)
#     resp = requests.get(url_get_block, headers=heads)
#     blockepoch = resp.json()["Epoch"]
#     if  blockepoch!= epoch:
#         print("XXXXXXXXXXXXXXXXXX")
#         print("Get block failed")
#         print(resp.text)
#         quit()  

# # for epoch in range (1, highest_epoch):
# #     print("Verify epoch <", epoch,  "> on u1")    
# #     url_get_block = "http://127.0.0.1:8006/api/v1/block/%s/%d" % (group_id, epoch)
# #     resp = requests.get(url_get_block, headers=heads)
# #     blockepoch = resp.json()["Epoch"]
# #     if  blockepoch!= epoch:
# #         print("XXXXXXXXXXXXXXXXXX")
# #         print("Get block failed")
# #         print(resp.text)
# #         quit()  
        
# for trx_id in trx_id_list:
#     url_get_trx = "http://127.0.0.1:8002/api/v1/trx/%s/%s" % (group_id, trx_id)
#     resp = requests.get(url_get_trx, headers=heads)
#     c_trx_id = resp.json()["TrxId"]
#     c_save_type = resp.json()["StorageType"]
#     if c_trx_id != trx_id or c_save_type != "CHAIN":
#         print("XXXXXXXXXXXXXXXXXX")
#         print("Get trx failed")
#         print(resp.text)
#         quit()     
#     print("Get trx", trx_id, " on o1 success")

#     # url_get_trx = "http://127.0.0.1:8006/api/v1/trx/%s/%s" % (group_id, trx_id)
#     # resp = requests.get(url_get_trx, headers=heads)
#     # c_trx_id = resp.json()["TrxId"]
#     # c_save_type = resp.json()["StorageType"]
#     # if c_trx_id != trx_id or c_save_type != "CHAIN":
#     #     print("XXXXXXXXXXXXXXXXXX")
#     #     print("Get trx failed")
#     #     print(resp.text)
#     #     quit()     
#     # print("Get trx", trx_id, " on u1 success")