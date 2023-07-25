package data

import (
	"bytes"
	"encoding/base64"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func GetReqBlocksMsg(keyalias string, groupId string, requesterPubkey string, fromBlock uint64, blkReq int32) (*quorumpb.ReqBlock, error) {
	reqBlock := &quorumpb.ReqBlock{}
	reqBlock.GroupId = groupId
	reqBlock.FromBlock = fromBlock
	reqBlock.BlksRequested = blkReq
	reqBlock.ReqPubkey = requesterPubkey

	//hash the request
	byts, err := proto.Marshal(reqBlock)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(byts)
	reqBlock.Hash = hash

	//sign
	ks := localcrypto.GetKeystore()
	signature, err := ks.EthSignByKeyAlias(keyalias, hash)
	if err != nil {
		return nil, err
	}
	reqBlock.Sign = signature
	return reqBlock, nil
}

func GetReqBlocksRespMsg(keyalias string, req *quorumpb.ReqBlock, provider string, blocks []*quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.ReqBlockResp, error) {
	reqBlockResp := &quorumpb.ReqBlockResp{}
	reqBlockResp.GroupId = req.GroupId
	reqBlockResp.RequesterPubkey = req.ReqPubkey
	reqBlockResp.ProviderPubkey = provider
	reqBlockResp.Result = result
	reqBlockResp.FromBlock = req.FromBlock
	reqBlockResp.BlksRequested = req.BlksRequested
	reqBlockResp.BlksProvided = int32(len(blocks))
	blockBundles := &quorumpb.BlocksBundle{}
	blockBundles.Blocks = blocks
	reqBlockResp.Blocks = blockBundles

	//hash the response
	byts, err := proto.Marshal(reqBlockResp)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(byts)
	reqBlockResp.Hash = hash

	//sign
	ks := localcrypto.GetKeystore()
	signature, err := ks.EthSignByKeyAlias(keyalias, hash)
	if err != nil {
		return nil, err
	}
	reqBlockResp.Sign = signature

	return reqBlockResp, nil
}

func VerifyReqBlock(req *quorumpb.ReqBlock) (bool, error) {
	//clone the req
	clonereq := &quorumpb.ReqBlock{}
	clonereq.GroupId = req.GroupId
	clonereq.FromBlock = req.FromBlock
	clonereq.BlksRequested = req.BlksRequested
	clonereq.ReqPubkey = req.ReqPubkey

	//hash the request
	byts, err := proto.Marshal(clonereq)
	if err != nil {
		return false, err
	}

	hash := localcrypto.Hash(byts)

	//compare hash with the req
	if !bytes.Equal(hash, req.Hash) {
		return false, fmt.Errorf("hash not match")
	}

	ks := localcrypto.GetKeystore()
	bytespubkey, err := base64.RawURLEncoding.DecodeString(req.ReqPubkey)

	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			r := ks.EthVerifySign(hash, req.Sign, ethpubkey)
			return r, nil
		}
		return false, err
	}
	return false, err
}

func VerifyReqBlockResp(resp *quorumpb.ReqBlockResp) (bool, error) {
	//clone the resp
	cloneresp := &quorumpb.ReqBlockResp{}
	cloneresp.GroupId = resp.GroupId
	cloneresp.RequesterPubkey = resp.RequesterPubkey
	cloneresp.ProviderPubkey = resp.ProviderPubkey
	cloneresp.Result = resp.Result
	cloneresp.FromBlock = resp.FromBlock
	cloneresp.BlksRequested = resp.BlksRequested
	cloneresp.BlksProvided = resp.BlksProvided
	cloneresp.Blocks = resp.Blocks

	//hash the response
	byts, err := proto.Marshal(cloneresp)
	if err != nil {
		return false, err
	}

	hash := localcrypto.Hash(byts)

	//compare hash with the resp
	if !bytes.Equal(hash, resp.Hash) {
		return false, fmt.Errorf("hash not match")
	}

	ks := localcrypto.GetKeystore()
	bytespubkey, err := base64.RawURLEncoding.DecodeString(resp.ProviderPubkey)

	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			r := ks.EthVerifySign(hash, resp.Sign, ethpubkey)
			return r, nil
		}
		return false, err
	}

	return false, err
}
