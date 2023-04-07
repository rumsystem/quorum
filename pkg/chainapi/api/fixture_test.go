package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"

	"filippo.io/age"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-querystring/query"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/testnode"
)

var (
	pidlist                                       []int
	bootstrapapi, peerapi, peerapi2               string
	peerapilist, groupIds                         []string
	bpnode1, bpnode2                              *testnode.NodeInfo
	timerange, fullnodes, groups, posts, synctime int
	ethPrivkey                                    *ecdsa.PrivateKey
	ethPubkey                                     string
	ageIdentity                                   *age.X25519Identity

	logger = logging.Logger("api")
)

func TestMain(m *testing.M) {
	fullnodes = 2
	pidch := make(chan int)

	go func() {
		for {
			select {
			case pid := <-pidch:
				logger.Debug("receive pid", pid)
				pidlist = append(pidlist, pid)
				if len(pidlist) == 3 {
					return
				}
			}
		}
	}()

	var tempdatadir string
	ctx := context.Background()
	cliargs := testnode.Nodecliargs{Rextest: false}
	var nodeInfos []*testnode.NodeInfo
	var err error
	nodeInfos, tempdatadir, err = testnode.RunNodesWithBootstrap(ctx, cliargs, pidch, fullnodes, 2)
	if err != nil {
		panic(err)
	}

	logger.Debugf("nodeInfos: %+v", nodeInfos)
	bootstrapapi = nodeInfos[0].APIBaseUrl
	peerapi = nodeInfos[1].APIBaseUrl
	peerapi2 = nodeInfos[2].APIBaseUrl
	peerapilist = []string{peerapi, peerapi2}
	bpnode1 = nodeInfos[len(nodeInfos)-2]
	bpnode2 = nodeInfos[len(nodeInfos)-1]

	// eth key
	ethPrivkey, err = ethcrypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	ethPubkey = ethcrypto.PubkeyToAddress(ethPrivkey.PublicKey).Hex()

	// age private key
	ageIdentity, err = age.GenerateX25519Identity()
	if err != nil {
		panic(err)
	}

	exitVal := m.Run()
	logger.Debug("after tests clean:", tempdatadir)
	testnode.Cleanup(tempdatadir, nodeInfos)
	os.Exit(exitVal)
}

func GetMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func StringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	sort.Strings(a)
	sort.Strings(b)

	b = b[:len(a)]
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func StringSetIn(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func RandString(n int) string {
	rand.Seed(time.Now().UnixNano())

	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func requestAPI(baseUrl string, endpoint string, method string, payload interface{}, headers http.Header, result interface{}, isSkipValidate bool) (int, []byte, error) {
	_url, err := url.JoinPath(baseUrl, endpoint)
	if err != nil {
		return 0, nil, fmt.Errorf("url.JoinPath(%s, %s) failed: %s", baseUrl, endpoint, err)
	}

	if method == "GET" && payload != nil {
		q, err := query.Values(payload)
		if err != nil {
			return 0, nil, fmt.Errorf("convert struct %+v to query string failed: %s", payload, err)
		}
		_url = fmt.Sprintf("%s?%s", _url, q.Encode())
	}

	statusCode, resp, err := utils.RequestAPI(_url, method, payload, headers, result)
	if err != nil || statusCode >= 400 {
		e := fmt.Errorf("%s %s failed: %s, payload: %+v, response: %s", method, _url, err, payload, resp)
		logger.Error(e)
		return statusCode, resp, e
	}

	if result != nil {
		if err := json.Unmarshal(resp, result); err != nil {
			logger.Errorf("json.Unmarshal %+v failed: %s", resp, err)
			return statusCode, resp, err
		}
	}

	if !isSkipValidate {
		validate := validator.New()
		if err := validate.Struct(result); err != nil {
			e := fmt.Errorf("validate.Struct failed: %s, response: %s", err, resp)
			logger.Error(e)
			return statusCode, resp, e
		}
	}

	return statusCode, resp, nil
}

func requestNSdk(urls []string, endpoint string, method string, payload interface{}, headers http.Header, result interface{}, isSkipValidate bool) (int, []byte, error) {
	apiUrl, err := url.Parse(urls[0])
	if err != nil {
		return 0, nil, fmt.Errorf("url.Parse(%s) failed: %s", urls[0], err)
	}

	baseUrl := fmt.Sprintf("%s://%s%s", apiUrl.Scheme, apiUrl.Host, apiUrl.Path)
	token := apiUrl.Query().Get("jwt")
	if token == "" {
		return 0, nil, fmt.Errorf("invalid jwt token: %s", token)
	}

	if headers == nil {
		headers = http.Header{}
	}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	return requestAPI(baseUrl, endpoint, method, payload, headers, result, isSkipValidate)
}
