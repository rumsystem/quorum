//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"syscall/js"

	quorum "github.com/rumsystem/quorum/internal/pkg/wasm"
	quorumAPI "github.com/rumsystem/quorum/internal/pkg/wasm/api"
)

// quit channel
var qChan = make(chan struct{}, 0)

func registerCallbacks() {
	js.Global().Set("StartQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan == nil {
			qChan = make(chan struct{}, 0)
		}
		bootAddr := args[0].String()
		handler := func() (map[string]interface{}, error) {
			ret := make(map[string]interface{})
			ok, err := quorum.StartQuorum(qChan, bootAddr)
			ret["ok"] = ok
			if err != nil {
				return ret, err
			}
			return ret, nil
		}
		return quorum.Promisefy(handler)
	}))
	js.Global().Set("StopQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan != nil {
			close(qChan)
			qChan = nil
		}
		return js.ValueOf(true).Bool()
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
		return quorum.Promisefy(handler)
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
		return quorum.Promisefy(handler)
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
		return quorum.Promisefy(handler)
	}))

	js.Global().Set("IndexDBTest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go quorum.IndexDBTest()
		return js.ValueOf(true).Bool()
	}))
}

func main() {
	c := make(chan struct{}, 0)

	println("WASM Go Initialized")
	// register functions
	registerCallbacks()
	<-c
}
