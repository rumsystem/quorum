package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/testnode"
)

var (
	pidlist                                       []int
	bootstrapapi, peerapi, peerapi2               string
	peerapilist, groupIds                         []string
	bpnode1, bpnode2                              *testnode.NodeInfo
	timerange, fullnodes, groups, posts, synctime int

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

func requestAPI(baseUrl string, endpoint string, method string, payload interface{}, result interface{}) (int, []byte, error) {
	payloadByte := []byte("")
	if payload != nil {
		var err error
		payloadByte, err = json.Marshal(payload)
		if err != nil {
			logger.Errorf("json.Marshal %+v failed: %s", payload, err)
			return 0, nil, err
		}
	}

	statusCode, resp, err := testnode.RequestAPI(baseUrl, endpoint, method, string(payloadByte))
	if err != nil || statusCode >= 400 {
		logger.Errorf("%s %s failed: %s, payload: %s, response: %s", method, endpoint, err, string(payloadByte), resp)
		return statusCode, resp, err
	}

	if result != nil {
		if err := json.Unmarshal(resp, result); err != nil {
			logger.Errorf("json.Unmarshal %+v failed: %s", resp, err)
			return statusCode, resp, err
		}
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		e := fmt.Errorf("validate.Struct failed: %s, result: %+v", err, result)
		logger.Error(e)
		return statusCode, resp, e
	}

	return statusCode, resp, nil
}
