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
	"github.com/rumsystem/quorum/internal/pkg/handlers"
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

func GetPubQueue(groupId string, trxId string, status string) (*handlers.PubQueueInfo, error) {
	url := fmt.Sprintf(
		"%s/api/v1/group/%s/pubqueue?trx=%s&status=%s", ApiServer, groupId, trxId, status)
	ret := handlers.PubQueueInfo{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil || ret.GroupId == "" {
		return nil, errors.New(string(body))
	}
	return &ret, nil
}

func PubQueueAck(trxIds []string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/trx/ack", ApiServer)
	param := qApi.PubQueueAckPayload{trxIds}
	payload, err := json.Marshal(&param)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	body, err := httpPost(url, payload)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return ret, nil
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

func AddGroupConfig(groupId, key, tp, value, memo string) (*handlers.AppConfigResult, error) {
	return ModifyGroupConfig("add", groupId, key, tp, value, memo)
}

func DelGroupConfig(groupId, key, tp, value, memo string) (*handlers.AppConfigResult, error) {
	return ModifyGroupConfig("del", groupId, key, tp, value, memo)
}

func ModifyGroupConfig(action, groupId, key, tp, value, memo string) (*handlers.AppConfigResult, error) {
	data := handlers.AppConfigParam{
		Action:  action,
		GroupId: groupId,
		Name:    key,
		Type:    tp,
		Value:   value,
		Memo:    memo,
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := ApiServer + "/api/v1/group/appconfig"
	ret := handlers.AppConfigResult{}
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

func GetGroupConfigList(groupId string) ([]*handlers.AppConfigKeyListItem, error) {
	url := fmt.Sprintf("%s/api/v1/group/%s/config/keylist", ApiServer, groupId)
	ret := []*handlers.AppConfigKeyListItem{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return ret, nil
}

func GetGroupConfig(groupId, key string) (*handlers.AppConfigKeyItem, error) {
	url := fmt.Sprintf("%s/api/v1/group/%s/config/%s", ApiServer, groupId, key)
	ret := handlers.AppConfigKeyItem{}
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

// /v1/group/chainconfig
func UpdateChainConfig(groupId, tp, config, memo string) (*handlers.ChainConfigResult, error) {
	data := handlers.ChainConfigParams{
		GroupId: groupId,
		Type:    tp,
		Config:  config,
		Memo:    memo,
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := ApiServer + "/api/v1/group/chainconfig"
	ret := handlers.ChainConfigResult{}
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

// /v1/group/:group_id/trx/auth/:trx_type
func GetChainAuthMode(groupId, trxType string) (*handlers.TrxAuthItem, error) {
	url := fmt.Sprintf("%s/api/v1/group/%s/trx/auth/%s", ApiServer, groupId, trxType)
	ret := handlers.TrxAuthItem{}
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

// /v1/group/:group_id/trx/allowlist
func GetChainAllowList(groupId string) ([]*handlers.ChainSendTrxRuleListItem, error) {
	url := fmt.Sprintf("%s/api/v1/group/%s/trx/allowlist", ApiServer, groupId)
	ret := []*handlers.ChainSendTrxRuleListItem{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return ret, nil
}

// /v1/group/:group_id/trx/denylist
func GetChainDenyList(groupId string) ([]*handlers.ChainSendTrxRuleListItem, error) {
	url := fmt.Sprintf("%s/api/v1/group/%s/trx/denylist", ApiServer, groupId)
	ret := []*handlers.ChainSendTrxRuleListItem{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return ret, nil
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

func GetGroupSeed(gid string) (*handlers.GroupSeed, error) {
	url := fmt.Sprintf("%s/api/v1/group/%s/seed", ApiServer, gid)
	ret := handlers.GroupSeed{}
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

type BackupResult struct {
	// encrypt json.Marshal([]GroupSeed)
	Seeds    string `json:"seeds"`
	Keystore string `json:"keystore"`
	Config   string `json:"config" validate:"required"`
}

func DoBackup() (*BackupResult, error) {
	url := fmt.Sprintf("%s/api/v1/backup", ApiServer)
	ret := BackupResult{}
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

func AnnouncedUsers(groupId string) ([]*handlers.AnnouncedUserListItem, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	url := fmt.Sprintf("%s/api/v1/group/%s/announced/users", ApiServer, groupId)
	ret := []*handlers.AnnouncedUserListItem{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return ret, nil
}

func AnnouncedProducers(groupId string) ([]*handlers.AnnouncedProducerListItem, error) {
	if !IsValidApiServer() {
		return nil, errors.New("api server is invalid: " + ApiServer)
	}
	url := fmt.Sprintf("%s/api/v1/group/%s/announced/producers", ApiServer, groupId)
	ret := []*handlers.AnnouncedProducerListItem{}
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, errors.New(string(body))
	}
	return ret, nil
}

func ApproveAnnouncedUser(groupId string, user *handlers.AnnouncedUserListItem, removal bool) (*ApproveGrpUserResult, error) {
	ret := &ApproveGrpUserResult{}
	url := ApiServer + "/api/v1/group/user"

	action := "add"
	if removal {
		action = "remove"
	}

	data := ApproveGrpUserParam{
		Action:     action,
		UserPubkey: user.AnnouncedSignPubkey,
		GroupId:    groupId,
		Memo:       "by cli",
	}
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

	return ret, nil
}

func ApproveAnnouncedProducer(groupId string, user *handlers.AnnouncedProducerListItem, removal bool) (*handlers.GrpProducerResult, error) {
	ret := &handlers.GrpProducerResult{}
	url := ApiServer + "/api/v1/group/producer"

	action := "add"
	if removal {
		action = "remove"
	}

	data := handlers.GrpProducerParam{
		Action:         action,
		ProducerPubkey: user.AnnouncedPubkey,
		GroupId:        groupId,
		Memo:           "by cli",
	}
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

	return ret, nil
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

	if strings.Contains(string(body), "error") {
		return nil, errors.New(string(body))
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
