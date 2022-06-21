package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	"github.com/rumsystem/rumchaindata/pkg/pb"
	"math/big"
	"net/url"
	"strconv"
	"strings"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

func initSignKey(groupId string, ks localcrypto.Keystore, nodeoptions *options.NodeOptions) (string, error) {
	hexkey, err := ks.GetEncodedPubkey(groupId, localcrypto.Sign)
	if err != nil && strings.HasPrefix(err.Error(), "key not exist ") {
		newsignaddr, err := ks.NewKeyWithDefaultPassword(groupId, localcrypto.Sign)
		if err == nil && newsignaddr != "" {
			err = nodeoptions.SetSignKeyMap(groupId, newsignaddr)
			if err != nil {
				return "", errors.New(fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error()))
			}
			hexkey, err = ks.GetEncodedPubkey(groupId, localcrypto.Sign)
		} else {
			return "", errors.New("create new group key err:" + err.Error())
		}
	}
	return hexkey, nil
}

func initEncryptKey(groupId string, bks localcrypto.Keystore) (string, error) {
	userEncryptKey, err := bks.GetEncodedPubkey(groupId, localcrypto.Encrypt)
	if err != nil {
		if strings.HasPrefix(err.Error(), "key not exist ") {
			userEncryptKey, err = bks.NewKeyWithDefaultPassword(groupId, localcrypto.Encrypt)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return userEncryptKey, nil
}

func GroupSeedToUrl(version int, urls []string, seed *GroupSeed) (string, error) {
	urllist := []string{}
	for _, u := range urls {
		urllist = append(urllist, url.QueryEscape(u))
	}

	b64buuid, _ := guuid.Parse(seed.GenesisBlock.BlockId)
	b64guuid, _ := guuid.Parse(seed.GenesisBlock.GroupId)
	b64bstr := base64.RawURLEncoding.EncodeToString(b64buuid[:])
	b64gstr := base64.RawURLEncoding.EncodeToString(b64guuid[:])

	b64timestampstr := base64.RawURLEncoding.EncodeToString(big.NewInt(seed.GenesisBlock.TimeStamp).Bytes())

	var intencrypttype int32
	if seed.EncryptionType == "public" {
		intencrypttype = int32(pb.GroupEncryptType_PUBLIC)
	} else {
		intencrypttype = int32(pb.GroupEncryptType_PRIVATE)
	}

	var intconsensustype int32
	if seed.ConsensusType == "poa" {
		intconsensustype = 0
	} else if seed.ConsensusType == "pos" {
		intconsensustype = 1
	}

	cipherkeybytes, err := hex.DecodeString(seed.CipherKey)
	if err != nil {
		return "", err
	}
	b64cipher := base64.RawURLEncoding.EncodeToString(cipherkeybytes)
	b64sign := base64.RawURLEncoding.EncodeToString(seed.GenesisBlock.Signature)

	byteproducerpubkey, err := p2pcrypto.ConfigDecodeKey(seed.GenesisBlock.ProducerPubKey)
	if err != nil {
		return "", err
	}
	b64producerpubkey := base64.RawURLEncoding.EncodeToString(byteproducerpubkey)

	values := url.Values{}
	values.Add("b", b64bstr)
	values.Add("g", b64gstr)
	values.Add("k", b64producerpubkey)
	values.Add("t", b64timestampstr)
	values.Add("s", b64sign)
	values.Add("c", b64cipher)
	query := values.Encode()
	query = fmt.Sprintf("rum://seed?v=%d&e=%d&n=%d&%s&a=%s&y=%s&u=%s", version, intencrypttype, intconsensustype, query, url.QueryEscape(seed.GroupName), url.QueryEscape(seed.AppKey), strings.Join(urllist, "|"))
	return query, nil
}

func UrlToGroupSeed(seedurl string) (*GroupSeed, error) {
	if !strings.HasPrefix(seedurl, "rum://seed?") {
		return nil, errors.New("invalid Seed URL")
	}
	u, err := url.Parse(seedurl)
	if err != nil {
		return nil, err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}
	version := q.Get("v")
	if version != "1" {
		return nil, errors.New("unsupport seed url version")
	}
	b64bstr := q.Get("b")
	b64gstr := q.Get("g")

	b64bbyte, err := base64.RawURLEncoding.DecodeString(b64bstr)
	b64gbyte, err := base64.RawURLEncoding.DecodeString(b64gstr)
	b64buuid, err := guuid.FromBytes(b64bbyte)
	if err != nil {
		return nil, fmt.Errorf("uuid decode err: %s", err)
	}
	b64guuid, err := guuid.FromBytes(b64gbyte)
	if err != nil {
		return nil, fmt.Errorf("uuid decode err: %s", err)
	}

	b64producerpubkey := q.Get("k")
	b64producerpubkeybyte, err := base64.RawURLEncoding.DecodeString(b64producerpubkey)
	producerpubkey := p2pcrypto.ConfigEncodeKey(b64producerpubkeybyte)

	b64timestampstr := q.Get("t")

	b64timestampbyte, err := base64.RawURLEncoding.DecodeString(b64timestampstr)
	timestamp := big.NewInt(0).SetBytes(b64timestampbyte).Int64()

	b64sign := q.Get("s")
	b64signbyte, err := base64.RawURLEncoding.DecodeString(b64sign)

	if err != nil {
		return nil, fmt.Errorf("sign decode err: %s", err)
	}

	b64cipher := q.Get("c")
	cipherkeybytes, err := base64.RawURLEncoding.DecodeString(b64cipher)
	if err != nil {
		return nil, fmt.Errorf("seed decode err: %s", err)
	}

	cipherkeyhexstr := hex.EncodeToString(cipherkeybytes)

	appkey, err := url.QueryUnescape(q.Get("y"))
	if err != nil {
		return nil, fmt.Errorf("seed decode err: %s", err)
	}

	genesisBlock := &pb.Block{
		BlockId:        b64buuid.String(),
		GroupId:        b64guuid.String(),
		PrevBlockId:    "",
		PreviousHash:   nil,
		TimeStamp:      timestamp,
		ProducerPubKey: producerpubkey,
		Trxs:           nil,
		Signature:      b64signbyte,
	}

	hash, err := rumchaindata.BlockHash(genesisBlock)
	if err != nil {
		return nil, err
	}
	genesisBlock.Hash = hash

	consensustype := "poa"
	if q.Get("n") == "1" {
		consensustype = "pos"
	}

	intencrypttype, _ := strconv.Atoi(q.Get("e"))

	encryptiontype := "public"
	if int32(intencrypttype) == int32(pb.GroupEncryptType_PUBLIC) {
		encryptiontype = "public"
	} else if int32(intencrypttype) == int32(pb.GroupEncryptType_PRIVATE) {
		encryptiontype = "private"
	}

	seed := &GroupSeed{
		GenesisBlock:   genesisBlock,
		GroupName:      q.Get("a"),
		ConsensusType:  consensustype,
		EncryptionType: encryptiontype,
		CipherKey:      cipherkeyhexstr,
		AppKey:         appkey,
	}
	return seed, nil
}
