package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"net/url"
	"strconv"
	"strings"

	"github.com/rumsystem/quorum/pkg/pb"
)

func GroupSeedToUrl(version int, urls []string, seed *GroupSeed) (string, error) {
	urllist := []string{}
	for _, u := range urls {
		urllist = append(urllist, url.QueryEscape(u))
	}

	//handle genesisBlock
	genesisBlockByt, err := proto.Marshal(seed.GenesisBlock)
	if err != nil {
		return "", err
	}
	genesisBlockB64 := base64.RawURLEncoding.EncodeToString(genesisBlockByt)

	var synctype int32
	if seed.SyncType == "public" {
		synctype = int32(pb.GroupSyncType_PUBLIC)
	} else {
		synctype = int32(pb.GroupSyncType_PRIVATE)
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
	cipherB64 := base64.RawURLEncoding.EncodeToString(cipherkeybytes)

	signByts, err := hex.DecodeString(seed.Signature)
	if err != nil {
		return "", err
	}

	signB64 := base64.RawURLEncoding.EncodeToString(signByts)

	//TBD verify with houju,
	//handle genesisBlock properly
	values := url.Values{}
	values.Add("b", genesisBlockB64)
	values.Add("c", cipherB64)
	values.Add("s", signB64)
	query := values.Encode()
	query = fmt.Sprintf("rum://seed?v=%d&e=%d&n=%d&%s&g=%s&a=%s&u=%s", version, synctype, intconsensustype, query, url.QueryEscape(seed.GroupName), url.QueryEscape(seed.AppKey), strings.Join(urllist, "|"))
	return query, nil
}

func UrlToGroupSeed(seedurl string) (*GroupSeed, []string, error) {
	if !strings.HasPrefix(seedurl, "rum://seed?") {
		return nil, nil, errors.New("invalid Seed URL")
	}
	u, err := url.Parse(seedurl)
	if err != nil {
		return nil, nil, err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, nil, err
	}
	version := q.Get("v")
	if version != "1" {
		return nil, nil, errors.New("unsupport seed url version")
	}

	intsynctype, _ := strconv.Atoi(q.Get("e"))
	synctype := "public"
	if int32(intsynctype) == int32(pb.GroupSyncType_PUBLIC) {
		synctype = "public"
	} else if int32(intsynctype) == int32(pb.GroupSyncType_PRIVATE) {
		synctype = "private"
	}

	consensustype := "poa"
	if q.Get("n") == "1" {
		consensustype = "pos"
	}

	genesisBlockByts, err := base64.RawURLEncoding.DecodeString(q.Get("b"))
	if err != nil {
		return nil, nil, fmt.Errorf("seed decode err: %s", err)
	}

	genesisBlock := &pb.Block{}
	err = proto.Unmarshal(genesisBlockByts, genesisBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("seed decode err: %s", err)
	}

	cipherkeyByts, err := base64.RawURLEncoding.DecodeString(q.Get("c"))
	if err != nil {
		return nil, nil, fmt.Errorf("seed decode err: %s", err)
	}

	signByts, err := base64.RawURLEncoding.DecodeString(q.Get("s"))
	if err != nil {
		return nil, nil, fmt.Errorf("sign decode err: %s", err)
	}

	groupName, err := url.QueryUnescape(q.Get("g"))
	if err != nil {
		return nil, nil, fmt.Errorf("seed decode err: %s", err)
	}

	appKey, err := url.QueryUnescape(q.Get("a"))

	cipherkeyStr := hex.EncodeToString(cipherkeyByts)
	signStr := hex.EncodeToString(signByts)

	seed := &GroupSeed{
		GenesisBlock:  genesisBlock,
		GroupId:       genesisBlock.GroupId,
		GroupName:     groupName,
		OwnerPubkey:   genesisBlock.ProducerPubkey,
		ConsensusType: consensustype,
		SyncType:      synctype,
		CipherKey:     cipherkeyStr,
		AppKey:        appKey,
		Signature:     signStr,
	}

	urlstr := q.Get("u")
	urls := strings.Split(urlstr, "|")
	for i, u := range urls {
		if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
			urls[i] = fmt.Sprintf("%s%s", "https://", u)
		}
	}
	return seed, urls, nil
}
