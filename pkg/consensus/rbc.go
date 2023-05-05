package consensus

import (
	"crypto/sha256"
	"sort"

	"github.com/NebulousLabs/merkletree"
	"github.com/klauspost/reedsolomon"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type Echos []*quorumpb.Echo

func (p Echos) Len() int           { return len(p) }
func (p Echos) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Echos) Less(i, j int) bool { return p[i].Index < p[j].Index }

func MakeRBCInitProposeMessage(groupId, nodename, proposerPubkey string, shards [][]byte, producerList []string, originalDataSize int) ([]*quorumpb.RBCMsg, error) {
	msgs := make([]*quorumpb.RBCMsg, len(shards))

	for i := 0; i < len(msgs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(msgs); j++ {
			tree.Push(shards[j])
		}
		root, proof, proofIndex, n := tree.Prove()

		/*
			//create ECHO for myself
			if producerList[i] == proposerPubkey {
				payload := &quorumpb.Echo{
					RootHash:               root,
					Proof:                  proof,
					Index:                  int64(proofIndex),
					Leaves:                 int64(n),
					OriginalDataSize:       int64(originalDataSize),
					OriginalProposerPubkey: proposerPubkey, //myself
					EchoProviderPubkey:     proposerPubkey, //myself
					EchoProviderSign:       nil,
				}
				//get hash
				bbytes, err := proto.Marshal(payload)
				if err != nil {
					return nil, err
				}
				payloadhash := localcrypto.Hash(bbytes)

				//sign it
				var signature []byte
				ks := localcrypto.GetKeystore()
				signature, err = ks.EthSignByKeyName(groupId, payloadhash, nodename)
				if err != nil {
					return nil, err
				}

				payload.EchoProviderSign = signature

				//put msg to container
				payloadb, err := proto.Marshal(payload)
				if err != nil {
					return nil, err
				}

				msgs[i] = &quorumpb.RBCMsg{
					Type:    quorumpb.RBCMsgType_ECHO,
					Payload: payloadb,
				}
			} else {
		*/
		payload := &quorumpb.InitPropose{
			RootHash:         root,
			Proof:            proof,
			Index:            int64(proofIndex),
			Leaves:           int64(n),
			OriginalDataSize: int64(originalDataSize),
			RecvNodePubkey:   producerList[i],
			ProposerPubkey:   proposerPubkey,
			ProposerSign:     nil,
		}

		//get hash
		bbytes, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}
		payloadhash := localcrypto.Hash(bbytes)

		//sign it
		var signature []byte
		ks := localcrypto.GetKeystore()
		signature, err = ks.EthSignByKeyName(groupId, payloadhash, nodename)
		if err != nil {
			return nil, err
		}

		payload.ProposerSign = signature

		//put msg to container
		payloadb, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}

		msgs[i] = &quorumpb.RBCMsg{
			Type:    quorumpb.RBCMsgType_INIT_PROPOSE,
			Payload: payloadb,
		}
		/*}*/
	}

	return msgs, nil
}

func MakeRBCEchoMessage(groupId, nodename, echoProviderPubkey string, initP *quorumpb.InitPropose, originalDataSize int) (*quorumpb.RBCMsg, error) {
	//just dump my part of InitPropose to ProofMsg and sign it
	payload := &quorumpb.Echo{
		RootHash:               initP.RootHash,
		Proof:                  initP.Proof,
		Index:                  initP.Index,
		Leaves:                 initP.Leaves,
		OriginalDataSize:       initP.OriginalDataSize,
		OriginalProposerPubkey: initP.ProposerPubkey,
		EchoProviderPubkey:     echoProviderPubkey,
		EchoProviderSign:       nil,
	}

	//get hash
	bbytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadhash := localcrypto.Hash(bbytes)

	//sign it
	var signature []byte
	ks := localcrypto.GetKeystore()
	signature, err = ks.EthSignByKeyName(groupId, payloadhash, nodename)
	if err != nil {
		return nil, err
	}

	payload.EchoProviderSign = signature

	payloadb, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &quorumpb.RBCMsg{
		Type:    quorumpb.RBCMsgType_ECHO,
		Payload: payloadb,
	}, nil
}

func MakeRBCReadyMessage(groupId, nodename, providerPubkey, originalProposerPubkey string, roothash []byte) (*quorumpb.RBCMsg, error) {
	ready := &quorumpb.Ready{
		RootHash:               roothash,
		OriginalProposerPubkey: originalProposerPubkey,
		ReadyProviderPubkey:    providerPubkey,
		ReadyProviderSign:      nil,
	}

	//sign root_hash with my pubkey
	bbytes, err := proto.Marshal(ready)
	if err != nil {
		return nil, err
	}

	readyHash := localcrypto.Hash(bbytes)

	var signature []byte
	ks := localcrypto.GetKeystore()
	signature, err = ks.EthSignByKeyName(groupId, readyHash, nodename)
	if err != nil {
		return nil, err
	}

	ready.ReadyProviderSign = signature

	payloadb, err := proto.Marshal(ready)
	if err != nil {
		return nil, err
	}

	return &quorumpb.RBCMsg{
		Type:    quorumpb.RBCMsgType_READY,
		Payload: payloadb,
	}, nil

}

func TryDecodeValue(echos Echos, enc reedsolomon.Encoder, numPShards int, numDShards int) ([]byte, error) {
	//sort proof by indexId
	sort.Sort(echos)

	//any not received index will be marked as nil, which meet the requirement of ecc (mark unavialble shards as nil)
	shards := make([][]byte, numPShards+numDShards)
	for _, p := range echos {
		shards[p.Index] = p.Proof[0]
	}

	//try reconstruct it
	if err := enc.Reconstruct(shards); err != nil {
		return nil, err
	}

	var value []byte
	for _, data := range shards[:numDShards] {
		value = append(value, data...)
	}

	/* IMPORTANT
	   An important thing to note is that you have to keep track of the exact input size.
	   If the size of the input isn't divisible by the number of data shards,
	   extra zeros will be inserted in the last shard.
	*/

	//cut the external 0
	//just get teh originalDataSize from proof[0]
	originalDataSize := echos[0].OriginalDataSize
	receivedDataSize := len(value)
	//diff
	diff := receivedDataSize - int(originalDataSize)
	if diff != 0 {
		value = value[:len(value)-diff]
	}
	return value, nil
}

func ValidateInitPropose(initp *quorumpb.InitPropose) bool {
	return merkletree.VerifyProof(
		sha256.New(),
		initp.RootHash,
		initp.Proof,
		uint64(initp.Index),
		uint64(initp.Leaves))
}

func ValidateEcho(req *quorumpb.Echo) bool {
	return merkletree.VerifyProof(
		sha256.New(),
		req.RootHash,
		req.Proof,
		uint64(req.Index),
		uint64(req.Leaves))
}

func MakeShards(enc reedsolomon.Encoder, data []byte) ([][]byte, error) {
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}

	if err := enc.Encode(shards); err != nil {
		return nil, err
	}

	return shards, nil
}
