package main

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/huo-ju/quercus/pkg/pubsub"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	api "github.com/rumsystem/quorum/internal/pkg/api"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	pubsubconn "github.com/rumsystem/quorum/internal/pkg/pubsubconn"
	"google.golang.org/protobuf/encoding/protojson"

	//"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"syscall"
)

var (
	signalch chan os.Signal
	mainlog  = logging.Logger("main")
)

func newGroupItem(params *api.GroupSeed, hexkey string, userencryptPubkey string) *quorumpb.GroupItem {
	privkey, _ := ethcrypto.HexToECDSA(hexkey)
	pubkeybytes := ethcrypto.FromECDSAPub(&privkey.PublicKey)
	p2ppubkey, _ := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, _ := p2pcrypto.MarshalPublicKey(p2ppubkey)

	var item *quorumpb.GroupItem
	item = &quorumpb.GroupItem{}
	item.GroupId = params.GroupId
	item.GroupName = params.GroupName
	item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	item.UserSignPubkey = item.OwnerPubKey
	item.UserEncryptPubkey = userencryptPubkey
	item.ConsenseType = quorumpb.GroupConsenseType_POA

	if params.EncryptionType == "public" {
		item.EncryptType = quorumpb.GroupEncryptType_PUBLIC
	} else {
		item.EncryptType = quorumpb.GroupEncryptType_PRIVATE
	}

	item.CipherKey = params.CipherKey
	item.HighestHeight = 0
	item.HighestBlockId = append(item.HighestBlockId, params.GenesisBlock.BlockId)
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = params.GenesisBlock
	return item
}

func joinGroupItem(params *api.GroupSeed, hexkey string) *quorumpb.GroupItem {
	privkey, _ := ethcrypto.HexToECDSA(hexkey)
	pubkeybytes := ethcrypto.FromECDSAPub(&privkey.PublicKey)
	p2ppubkey, _ := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, _ := p2pcrypto.MarshalPublicKey(p2ppubkey)

	item := &quorumpb.GroupItem{}
	item.OwnerPubKey = params.OwnerPubKey
	item.GroupId = params.GroupId
	item.GroupName = params.GroupName
	ownerPubkeyBytes, _ := p2pcrypto.ConfigDecodeKey(params.OwnerPubKey)
	item.OwnerPubKey = p2pcrypto.ConfigEncodeKey(ownerPubkeyBytes)
	item.CipherKey = params.CipherKey
	item.HighestHeight = 0
	item.HighestBlockId = append(item.HighestBlockId, params.GenesisBlock.BlockId)
	item.LastUpdate = time.Now().UnixNano()
	item.GenesisBlock = params.GenesisBlock

	item.ConsenseType = quorumpb.GroupConsenseType_POA
	item.UserSignPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	return item
}

func newGroup(ps *pubsub.Pubsub, params *api.GroupSeed, ks *localcrypto.MockKeyStore, nodename string) *chain.Group {
	keyname := fmt.Sprintf("%s_%s", nodename, params.GroupId)
	hexkey, err := ks.GetHexKey(localcrypto.Sign.NameString(keyname))

	userEncryptKey, err := ks.GetEncodedPubkey(keyname, localcrypto.Encrypt)
	if err != nil {
		return nil
	}
	mainlog.Infof("Create group hexkey: %s", hexkey)

	item := newGroupItem(params, hexkey, userEncryptKey)
	grp := chain.Group{Item: item}
	grp.ChainCtx = &chain.Chain{}

	ctx := context.Background()
	producerChannelId := chain.PRODUCER_CHANNEL_PREFIX + grp.Item.GroupId
	producerPubsubconn := pubsubconn.InitQuercusConn(ctx, ps, chain.PRODUCER_CHANNEL_PREFIX+nodename) //quercus_channel.go Init
	producerPubsubconn.JoinChannel(producerChannelId, grp.ChainCtx)

	userChannelId := chain.USER_CHANNEL_PREFIX + grp.Item.GroupId
	userPubsubconn := pubsubconn.InitQuercusConn(ctx, ps, chain.USER_CHANNEL_PREFIX+nodename) //quercus_channel.go Init
	userPubsubconn.JoinChannel(userChannelId, grp.ChainCtx)
	grp.ChainCtx.CustomInit(nodename, &grp, producerPubsubconn, userPubsubconn)

	err = chain.GetDbMgr().AddGensisBlock(item.GenesisBlock, nodename)
	if err != nil {
		return nil
	}
	return &grp
}

func joinGroup(ps *pubsub.Pubsub, params *api.GroupSeed, ks *localcrypto.MockKeyStore, nodename string) *chain.Group {
	keyname := fmt.Sprintf("%s_%s", nodename, params.GroupId)
	hexkey, err := ks.GetHexKey(localcrypto.Sign.NameString(keyname))
	if err != nil {
		fmt.Println(err)
	}

	item := joinGroupItem(params, hexkey)

	grp := chain.Group{Item: item}
	grp.ChainCtx = &chain.Chain{}
	ctx := context.Background()

	producerChannelId := chain.PRODUCER_CHANNEL_PREFIX + grp.Item.GroupId
	producerPubsubconn := pubsubconn.InitQuercusConn(ctx, ps, chain.PRODUCER_CHANNEL_PREFIX+nodename)
	producerPubsubconn.JoinChannel(producerChannelId, grp.ChainCtx)

	userChannelId := chain.USER_CHANNEL_PREFIX + grp.Item.GroupId
	userPubsubconn := pubsubconn.InitQuercusConn(ctx, ps, chain.USER_CHANNEL_PREFIX+nodename)
	userPubsubconn.JoinChannel(userChannelId, grp.ChainCtx)
	grp.ChainCtx.CustomInit(nodename, &grp, producerPubsubconn, userPubsubconn)

	err = chain.GetDbMgr().AddGensisBlock(item.GenesisBlock, nodename)
	if err != nil {
		return nil
	}
	return &grp
}

func main() {

	logging.SetLogLevel("producer", "debug")
	logging.SetLogLevel("chain", "debug")
	logging.SetLogLevel("syncer", "debug")
	logging.SetLogLevel("dbmgr", "debug")
	logging.SetLogLevel("chainctx", "debug")
	logging.SetLogLevel("group", "debug")
	logging.SetLogLevel("chan", "debug")

	signalch = make(chan os.Signal, 1)

	//cons := chain.NewMolasses(nil)
	//fmt.Println(cons)
	ctx, _ := context.WithCancel(context.Background())

	joingroupitemstr := `{"genesis_block":{"BlockId":"910a164a-7afc-4a1a-b43f-2419bce606c6","GroupId":"7d73a361-a69e-4529-a76a-18fbc0ac34cc","BlockNum":1,"Timestamp":1630443404855835826,"ProducerPubKey":"CAISIQM/DIum4C5wjxnJJooGuLwQ6c7V+h0gxGadN4Hi2JZR+w==","Hash":"3Z8BKX26Piis7GMVSaWkYqBqh3hJ6g0Z7tCEGv7QoAQ=","Signature":"MEYCIQDLW4KwtUn00rlAKMt9Rom6L8ZfQCYAB0BrvkzgZMyroAIhAK+1vO/+CxQtWP1s1UJkmAgD7hMybCzHjlHJac7ncleN"},"group_id":"7d73a361-a69e-4529-a76a-18fbc0ac34cc","group_name":"pb_group_1_public","owner_pubkey":"CAISIQM/DIum4C5wjxnJJooGuLwQ6c7V+h0gxGadN4Hi2JZR+w==","owner_encryptpubkey":"age104y8fsk38nz465y5veq30rlrnv80wpd8wvudhspxjpl89m4cqvpsu8d9vq","consensus_type":"poa","encryption_type":"public","cipher_key":"09123fae1f1b9cec0df984f25b3aceb4c318d033ba2b4ce8ab128c48379a0a7b","signature":"3045022012df97dfd9e39e47f4c31c85c3e4e62cbc760b13960d07eb1144775a99e00a31022100d6618a44261c24d6b76cc331cc5258dd4a17ddf870ae1ae1ddbdfb9aa8381611"}`
	hexkey := "9bf3271d1188ab114c99a500a19b1bcd7e4caa795070db6461698c389dbf3db0"

	grpparams := &api.GroupSeed{}
	_ = json.Unmarshal([]byte(joingroupitemstr), grpparams)

	ps := pubsub.NewPubsub()
	ks, _, _ := localcrypto.InitMockKeyStore("mock", "/tmp/mockks")
	ks.Unlock(nil, "")

	if ok, _ := ks.IfKeyExist(grpparams.GroupId); ok == false {
		ks.Import(fmt.Sprintf("%s_%s", "node1", grpparams.GroupId), hexkey, localcrypto.Sign, "")
		ks.NewKey(fmt.Sprintf("%s_%s", "node1", grpparams.GroupId), localcrypto.Encrypt, "")
	}

	chain.InitCtx(ctx, "testpeer", nil, "", "", "sim")
	chain.GetNodeCtx().Keystore = ks
	//producer node1 create the group
	grp1 := newGroup(ps, grpparams, ks, "node1")
	errgrp := grp1.StartSync()

	//producer node2 join to the group
	ks.NewKey(fmt.Sprintf("%s_%s", "node2", grpparams.GroupId), localcrypto.Encrypt, "")
	ks.NewKey(fmt.Sprintf("%s_%s", "node2", grpparams.GroupId), localcrypto.Sign, "")
	node2grp := joinGroup(ps, grpparams, ks, "node2")
	errgrp = node2grp.StartSync()

	//producer node3 join to the group
	ks.NewKey(fmt.Sprintf("%s_%s", "node3", grpparams.GroupId), localcrypto.Encrypt, "")
	ks.NewKey(fmt.Sprintf("%s_%s", "node3", grpparams.GroupId), localcrypto.Sign, "")
	node3grp := joinGroup(ps, grpparams, ks, "node3")
	errgrp = node3grp.StartSync()
	fmt.Println(errgrp)

	//producer node4 join to the group
	//ks.NewKey(fmt.Sprintf("%s_%s", "node4", grpparams.GroupId), localcrypto.Encrypt, "")
	//ks.NewKey(fmt.Sprintf("%s_%s", "node4", grpparams.GroupId), localcrypto.Sign, "")
	//node4grp := joinGroup(ps, grpparams, ks, "node4")
	//errgrp = node4grp.StartSync()
	//fmt.Println(errgrp)

	//0. show group Consensus

	mainlog.Infof("grp1.ChainCtx.Consensus %s", grp1.ChainCtx.Consensus)

	//1. send user post to producer channel
	poststr := `{"type":"Add","object":{"type":"Note","content":"TEST 1 NOTE simple note","name":" A TEST simple"},"target":{"id":"7d73a361-a69e-4529-a76a-18fbc0ac34cc","type":"Group"}}'`
	pbobj := quorumpb.Activity{}
	_ = protojson.Unmarshal([]byte(poststr), &pbobj)
	encodedcontent, _ := quorumpb.ContentToBytes(pbobj.Object)

	producerTrxMgr := grp1.ChainCtx.GetProducerTrxMgr()
	trx, err := producerTrxMgr.CreateTrx(quorumpb.TrxType_POST, encodedcontent)

	err = producerTrxMgr.CustomSendTrx(trx)
	if err != nil {
		mainlog.Infof("send trx1 err %s", err)
	}

	//2.

	//mainlog.Infof("sleep 30s then send the trx2...")
	//time.Sleep(30 * time.Second)
	//mainlog.Infof("send the trx2...")
	//poststr = `{"type":"Add","object":{"type":"Note","content":"TEST 2 NOTE simple note","name":" A TEST simple"},"target":{"id":"7d73a361-a69e-4529-a76a-18fbc0ac34cc","type":"Group"}}'`
	//pbobj = quorumpb.Activity{}
	//_ = protojson.Unmarshal([]byte(poststr), &pbobj)
	//encodedcontent, _ = quorumpb.ContentToBytes(pbobj.Object)

	//trx, err = producerTrxMgr.CreateTrx(quorumpb.TrxType_POST, encodedcontent)

	//err = producerTrxMgr.CustomSendTrx(trx)
	//if err != nil {
	//	mainlog.Infof("send trx2 err %s", err)
	//}

	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)
	fmt.Printf("On Signal <%s>\n", signalType)
}
