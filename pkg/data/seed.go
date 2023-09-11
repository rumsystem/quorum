package data

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func VerifyGroupSeed(seed *quorumpb.GroupSeed) (bool, error) {
	seedClone := proto.Clone(seed).(*quorumpb.GroupSeed)
	seedClone.Hash = nil
	seedClone.Signature = nil

	//verify hash and signature
	seedCloneByts, err := proto.Marshal(seedClone)
	if err != nil {
		return false, err
	}

	hash := localcrypto.Hash(seedCloneByts)
	if !bytes.Equal(hash, seed.Hash) {
		msg := fmt.Sprintf("hash not match, expect %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(seed.Hash))
		return false, errors.New(msg)
	}

	verified, err := VerifySign(seed.OwnerPubkey, hash, seed.Signature)
	if err != nil {
		return false, err
	}

	if !verified {
		return false, errors.New("signature not verified")
	}

	//verify genesis block
	r, err := ValidGenesisBlockPoa(seed.GenesisBlock)
	if err != nil {
		return false, err
	}

	if !r {
		msg := "join group failed, verify genesis block failed"
		return false, errors.New(msg)
	}

	return true, nil
}
