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

	"github.com/huo-ju/quorum/cmd/cli/config"
)

var ApiServer string

func SetApiServer(apiServer string) {
	if len(apiServer) > 0 {
		if strings.HasPrefix(apiServer, "http") {
			ApiServer = apiServer
		} else {
			ApiServer = "http://" + apiServer
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

func Groups() (groupsInfo *GroupInfoListStruct, err error) {
	url := ApiServer + "/api/v1/groups"
	ret := GroupInfoListStruct{}
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

func Content(groupId string) (contents *[]ContentStruct, err error) {
	num := 1000
	if config.RumConfig.Quorum.MaxContentSize > 0 {
		num = config.RumConfig.Quorum.MaxContentSize
	}
	url := fmt.Sprintf(
		"%s/app/api/v1/group/%s/content?start=0&num=%d&reverse=true", ApiServer, groupId, num)
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

func CreateGroup(name string) (*GroupSeedStruct, error) {
	data := CreateGroupReqStruct{name}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := ApiServer + "/api/v1/group"
	ret := GroupSeedStruct{}
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

func TrxInfo(trxId string) (trx *TrxStruct, err error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	url := ApiServer + "/api/v1/trx/" + trxId
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

func newHTTPClient() (*http.Client, error) {
	certPath := config.RumConfig.Quorum.ServerSSLCertificate
	keyPath := config.RumConfig.Quorum.ServerSSLCertificateKey
	client := &http.Client{}
	if certPath != "" && keyPath != "" {
		caCert, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true}
		client = &http.Client{Transport: transport, Timeout: 30 * time.Second}
	}
	return client, nil
}
func httpGet(url string) ([]byte, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}

func httpPost(url string, data []byte) ([]byte, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}
func httpDelete(url string, data []byte) ([]byte, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}
