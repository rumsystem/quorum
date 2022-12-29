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

type Proofs []*quorumpb.Proof

func (p Proofs) Len() int           { return len(p) }
func (p Proofs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Proofs) Less(i, j int) bool { return p[i].Index < p[j].Index }

func MakeRBCProofMessages(groupId, nodename, proposerPubkey string, shards [][]byte) ([]*quorumpb.BroadcastMsg, error) {
	msgs := make([]*quorumpb.BroadcastMsg, len(shards))

	for i := 0; i < len(msgs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[i])
		}
		root, proof, proofIndex, n := tree.Prove()

		payload := &quorumpb.Proof{
			RootHash:       root,
			Proof:          proof,
			Index:          int64(proofIndex),
			Leaves:         int64(n),
			ProposerPubkey: proposerPubkey,
			ProposerSign:   nil,
		}

		bbytes, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}

		payloadhash := localcrypto.Hash(bbytes)

		var signature []byte
		ks := localcrypto.GetKeystore()
		signature, err = ks.EthSignByKeyName(groupId, payloadhash, nodename)
		if err != nil {
			return nil, err
		}
		payload.ProposerSign = signature

		payloadb, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}

		msgs[i] = &quorumpb.BroadcastMsg{
			Type:    quorumpb.BroadcastMsgType_PROOF,
			Payload: payloadb,
		}
	}

	return msgs, nil
}

func MakeRBCReadyMessage(groupId, nodename, signPubkey string, roothash []byte, ProposerPubkey string) (*quorumpb.BroadcastMsg, error) {
	ready := &quorumpb.Ready{
		RootHash:            roothash,       //proof.RootHash,
		ProofProviderPubkey: ProposerPubkey, // proof.ProposerPubkey, //pubkey for who send the original proof msg
		ProposerPubkey:      signPubkey,
		ProposerSign:        nil,
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

	ready.ProposerSign = signature

	payloadb, err := proto.Marshal(ready)
	if err != nil {
		return nil, err
	}

	readyMsg := &quorumpb.BroadcastMsg{
		Type:    quorumpb.BroadcastMsgType_READY,
		Payload: payloadb,
	}

	return readyMsg, nil
}

func TryDecodeValue(proofs Proofs, enc reedsolomon.Encoder, numPShards int, numDShards int) ([]byte, error) {
	//sort proof by indexId
	sort.Sort(proofs)

	shards := make([][]byte, numPShards+numDShards)
	for _, p := range proofs {
		shards[p.Index] = p.Proof[0]
	}

	if err := enc.Reconstruct(shards); err != nil {
		return nil, err
	}

	var value []byte
	for _, data := range shards[:numDShards] {
		value = append(value, data...)
	}

	return value, nil
}

func ValidateProof(req *quorumpb.Proof) bool {
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
