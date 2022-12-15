import requests
import time
import json

#create group 
heads = {'Content-Type': 'application/json'}
url_create_group = 'http://127.0.0.1:8002/api/v1/group'
url_get_groups = 'http://127.0.0.1:8002/api/v1/groups'
url_post_to_group = 'http://127.0.0.1:8002/api/v1/group/content/false'

payload_create_group = {
  "group_name": "my_test_group",
  "consensus_type": "poa",
  "encryption_type": "public",
  "app_key": "test_app"
}

response = requests.post(url_create_group, headers=heads, json=payload_create_group)
jsonResp = response.json()
group_id = jsonResp["group_id"]
respString = "Create Group with groupId <%s>" % group_id
print(respString)

#try post 10000 trxs and verify
for i in range (0, 100000):
    payload_post_to_group = {
        "type":"Add",
        "object":{"type":"Note",
                "content":"simple note by aa",
                "name":"A simple Node id1"},
        "target":{"id": group_id,
                "type":"Group"}              
    }
    
    response = requests.post(url_post_to_group, headers=heads, json=payload_post_to_group)
    trx_id = response.json()['trx_id']
    a = "Post with trxId <%s>" % trx_id
    print(a)  
    time.sleep(0.01)
    
    resp = requests.get(url_get_groups)
    jsonResp = resp.json()["groups"]
    group_info_json = jsonResp[0]
    epoch = group_info_json["epoch"]

    #this sleep is needed , even though the test only involve 1 owner node, make consensus / generate block / apply trx still time consuming.
    time.sleep(0.05)               
    print("current group epoch <", epoch, ">")    
    url_get_block = "http://127.0.0.1:8002/api/v1/block/%s/%d" % (group_id, epoch)
    resp = requests.get(url_get_block)
    blockepoch = resp.json()["Epoch"]
    if  blockepoch!= epoch:
        print("XXXXXXXXXXXXXXXXXX")
        print("Get block failed")
        print(resp.text)
        quit()            
    print("Get epoch", epoch, "done")
    
    #this sleep is needed , even though the test only involve 1 owner node, make consensus / generate block / apply trx still time consuming. 
    time.sleep(0.05)    
    url_get_trx = "http://127.0.0.1:8002/api/v1/trx/%s/%s" % (group_id, trx_id)
    resp = requests.get(url_get_trx)
    c_trx_id = resp.json()["TrxId"]
    c_save_type = resp.json()["StorageType"]
    if c_trx_id != trx_id or c_save_type != "CHAIN":
        print("XXXXXXXXXXXXXXXXXX")
        print("Get trx failed")
        print(resp.text)
        quit()     
    print("Get trx", trx_id, "done")
    

