//go:build js && wasm
// +build js,wasm

package wasm

import (
	"encoding/json"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/pkg/wasm/api"
	quorumAPI "github.com/rumsystem/quorum/pkg/wasm/api"
	"github.com/rumsystem/quorum/pkg/wasm/utils"
)

// quit channel
var qChan chan struct{} = nil

func RegisterJSFunctions() {
	js.Global().Set("SetDebug", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		enableDebug := args[0].Bool()
		if enableDebug {
			logging.SetAllLoggers(0)
		}
		return true
	}))
	js.Global().Set("SetLoggingLevel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		lvl := args[0].Int()
		logging.SetAllLoggers(lvl)
		return true
	}))
	js.Global().Set("GetKeystoreBackupReadableStream", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		password := args[0].String()
		return utils.GetKeystoreBackupReadableStream(password)
	}))

	js.Global().Set("KeystoreBackupRaw", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		password := args[0].String()
		onWrite := args[1]
		onFinish := args[2]

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			err := utils.KeystoreBackupRaw(
				password,
				func(str string) {
					onWrite.Invoke(js.ValueOf(str))
				},
				func() {
					onFinish.Invoke()
				},
			)
			if err != nil {
				return nil, err
			}
			ret["ok"] = true
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("KeystoreRestoreRaw", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		password := args[0].String()
		content := args[1].String()

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			err := utils.KeystoreRestoreRaw(
				password,
				content,
			)
			if err != nil {
				return nil, err
			}
			ret["ok"] = true
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("StartQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan == nil {
			qChan = make(chan struct{}, 0)
		}
		if len(args) < 2 {
			return nil
		}
		password := args[0].String()
		bootAddrsStr := args[1].String()
		bootAddrs := strings.Split(bootAddrsStr, ",")

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			ok, err := StartQuorum(qChan, password, bootAddrs)
			ret["ok"] = ok
			if err != nil {
				return ret, err
			}
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("IsQuorumRunning", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ret := qChan != nil
		return js.ValueOf(ret).Bool()
	}))

	js.Global().Set("StopQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan != nil {
			close(qChan)
			qChan = nil
		}
		return js.ValueOf(true).Bool()
	}))

	js.Global().Set("StartSync", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.StartSync(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("Announce", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.Announce([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetGroupProducers", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetGroupProducers(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetGroupSeed", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetGroupSeed(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetAnnouncedGroupProducers", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetAnnouncedGroupProducers(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetAnnouncedGroupUsers", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetAnnouncedGroupUsers(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GroupProducer", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GroupProducer([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("AddPeers", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		peersStr := args[0].String()
		peers := strings.Split(peersStr, ",")

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := api.AddPeers(peers)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("CreateGroup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.CreateGroup([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("MgrAppConfig", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.MgrAppConfig([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("MgrChainConfig", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.MgrChainConfig([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetChainTrxAllowList", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetChainTrxAllowList(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetChainTrxDenyList", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetChainTrxDenyList(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetChainTrxAuthMode", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		trxType := args[1].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetChainTrxAuthMode(groupId, trxType)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetAppConfigKeyList", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetAppConfigKeyList(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetAppConfigItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		itemKey := args[1].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetAppConfigItem(itemKey, groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("Ping", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		peer := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.Ping(peer)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("UpdateProfile", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.UpdateProfile([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetTrx", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			// TODO: return a Promise.reject
			return nil
		}
		groupId := args[0].String()
		trxId := args[1].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, _, err := quorumAPI.GetTrx(groupId, trxId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetPubQueue", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		groupId := args[0].String()
		status := ""
		trxId := ""
		if len(args) >= 2 { // optional
			status = args[1].String()
		}
		if len(args) >= 3 {
			trxId = args[2].String()
		}
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetPubQueue(groupId, status, trxId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("PostToGroup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsonStr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.PostToGroup([]byte(jsonStr))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetNodeInfo", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetNodeInfo()
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetNetwork", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetNetwork()
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetContent", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 5 {
			return nil
		}
		groupId := args[0].String()
		num := args[1].Int()
		startTrx := args[2].String()
		nonce, _ := strconv.ParseInt(args[3].String(), 10, 64)
		reverse := args[4].Bool()
		includestarttrx := args[5].Bool()

		senders := []string{}
		for i := 6; i < len(args); i += 1 {
			sender := args[i].String()
			senders = append(senders, sender)
		}

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetContent(groupId, num, startTrx, nonce, reverse, includestarttrx, senders)
			if err != nil {
				return ret, err
			}
			retBytes, _ := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("JoinGroup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		seed := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.JoinGroup([]byte(seed))
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("LeaveGroup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.LeaveGroup(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("ClearGroupData", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.ClearGroupData(groupId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetGroups", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetGroups()
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetBlockById", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		blockId := args[1].String()

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetBlockById(groupId, blockId)
			if err != nil {
				return ret, err
			}
			retBytes, err := json.Marshal(res)
			json.Unmarshal(retBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("GetDecodedBlockById", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		blockId := args[1].String()

		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			res, err := quorumAPI.GetDecodedBlockById(groupId, blockId)
			if err != nil {
				return ret, err
			}
			resBytes, err := json.Marshal(res)
			json.Unmarshal(resBytes, &ret)
			return ret, nil
		}
		return Promisefy(handler)
	}))

	js.Global().Set("IndexDBTest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go IndexDBTest()
		return js.ValueOf(true).Bool()
	}))
}
