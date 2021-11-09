package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/rumsystem/quorum/cmd/cli/config"
	qApi "github.com/rumsystem/quorum/internal/pkg/api"
)

var ApiServer string

func SetApiServer(apiServer string) {
	if len(apiServer) > 0 {
		if strings.HasPrefix(apiServer, "https") {
			ApiServer = apiServer
		} else {
			ApiServer = "https://" + apiServer
		}
		config.RumConfig.Quorum.Server = ApiServer
	}
}

func IsValidApiServer() bool {
	return len(ApiServer) > 0
}

func Node() (*NodeInfoStruct, error) {
	url := ApiServer + "/api/v1/node"
	ret := NodeInfoStruct{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func Network() (networkInfo *NetworkInfoStruct, err error) {
	url := ApiServer + "/api/v1/network"
	ret := NetworkInfoStruct{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func Ping() (res *map[string]PingInfoItemStruct, err error) {
	url := ApiServer + "/api/v1/network/peers/ping"
	ret := make(map[string]PingInfoItemStruct)
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func Groups() (groupsInfo *qApi.GroupInfoList, err error) {
	url := ApiServer + "/api/v1/groups"
	ret := qApi.GroupInfoList{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func Content(groupId string, opt PagerOpt) (contents *[]ContentStruct, err error) {
	num := 20
	if config.RumConfig.Quorum.MaxContentSize > 0 {
		num = config.RumConfig.Quorum.MaxContentSize
	}
	url := fmt.Sprintf(
		"%s/app/api/v1/group/%s/content?num=%d&reverse=%v", ApiServer, groupId, num, opt.Reverse)
	if opt.StartTrxId != "" {
		url = fmt.Sprintf("%s&starttrx=%s", url, opt.StartTrxId)
	}
	ret := []ContentStruct{}
	body, err := httpPost(url, []byte("{}"))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func ForceSyncGroup(groupId string) (syncRes *GroupForceSyncRetStruct, err error) {
	url := fmt.Sprintf(
		"%s/api/v1/group/%s/startsync", ApiServer, groupId)
	ret := GroupForceSyncRetStruct{}
	body, err := httpPost(url, []byte(""))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil || ret.GroupId == "" {
		return nil, errors.New(string(body))
	}
	if ret.Error != "" {
		return nil, errors.New(ret.Error)
	}
	return &ret, nil
}

func IsQuorumContentMessage(content ContentStruct) bool {
	// only support Note
	if content.TypeUrl == "quorum.pb.Object" {
		innerType, hasKey := content.Content["type"]
		if hasKey && innerType == "Note" {
			return true
		}
	}
	return false
}

func IsQuorumContentUserInfo(content ContentStruct) bool {
	// only support Note
	if content.TypeUrl == "quorum.pb.Person" {
		_, hasKey := content.Content["name"]
		if hasKey {
			return true
		}
	}
	return false
}

func Nick(groupId string, nick string) (*NickRespStruct, error) {
	data := NickReqStruct{
		Person: QuorumPersonStruct{
			Name: nick,
		},
		Target: QuorumTargetStruct{
			Id:   groupId,
			Type: "Group",
		},
		Type: "Update",
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	url := ApiServer + "/api/v1/group/profile"
	ret := NickRespStruct{}
	body, err := httpPost(url, json_data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func TokenApply() (*TokenRespStruct, error) {
	url := ApiServer + "/app/api/v1/token/apply"
	ret := TokenRespStruct{}
	body, err := httpPost(url, []byte(""))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func CreateContent(groupId string, content string) (*ContentRespStruct, error) {
	data := ContentReqStruct{
		Object: ContentReqObjectStruct{
			Content: content,
			Name:    "",
			Type:    "Note",
		},
		Target: ContentReqTargetStruct{
			Id:   groupId,
			Type: "Group",
		},
		Type: "Add",
	}
	url := ApiServer + "/api/v1/group/content"
	ret := ContentRespStruct{}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body, err := httpPost(url, json_data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func CreateGroup(data CreateGroupReqStruct) ([]byte, error) {
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := ApiServer + "/api/v1/group"
	return httpPost(url, json_data)
}

func LeaveGroup(gid string) (*GroupLeaveRetStruct, error) {
	data := LeaveGroupReqStruct{gid}
	url := ApiServer + "/api/v1/group/leave"
	ret := GroupLeaveRetStruct{}

	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body, err := httpPost(url, json_data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}

	return &ret, nil
}

func DelGroup(gid string) (*GroupDelRetStruct, error) {
	data := DeleteGroupReqStruct{gid}
	url := ApiServer + "/api/v1/group"
	ret := GroupDelRetStruct{}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	body, err := httpDelete(url, json_data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func TrxInfo(groupId string, trxId string) (trx *TrxStruct, err error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	url := fmt.Sprintf("%s/api/v1/trx/%s/%s", ApiServer, groupId, trxId)
	ret := TrxStruct{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func JoinGroup(seed string) (*JoinRespStruct, error) {
	url := ApiServer + "/api/v1/group/join"
	ret := JoinRespStruct{}
	body, err := httpPost(url, []byte(seed))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func GetBlockById(groupId string, id string) (*BlockStruct, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	url := fmt.Sprintf("%s/api/v1/block/%s/%s", ApiServer, groupId, id)
	ret := BlockStruct{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func newHTTPClient() (*http.Client, error) {
	certPath := config.RumConfig.Quorum.ServerSSLCertificate

	if certPath != "" {
		caCert, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		if err != nil {
			return nil, err
		}
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		if config.RumConfig.Quorum.ServerSSLInsecure {
			tlsConfig.InsecureSkipVerify = true
		}

		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true}
		// 5 seconds timeout, all timeout will be ignored, since we refresh all data every half second
		return &http.Client{Transport: transport, Timeout: 5 * time.Second}, nil
	}
	return &http.Client{}, nil
}

func checkJWTError(body string) error {
	if strings.Contains(body, "missing or malformed jwt") {
		return errors.New("missing or malformed jwt")
	}

	if strings.Contains(body, "please find jwt token in peer options") {
		return errors.New("Someone applied before, please find jwt token in peer options, then update your config and reload.")
	}
	return nil
}

func httpGet(url string) ([]byte, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	jwt := config.RumConfig.Quorum.JWT
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}
	return body, err
}

func httpPost(url string, data []byte) ([]byte, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	jwt := config.RumConfig.Quorum.JWT
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}
	return body, err
}
func httpDelete(url string, data []byte) ([]byte, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	jwt := config.RumConfig.Quorum.JWT
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}
	return body, err
}
