//go:build js && wasm
// +build js,wasm

// go:build js && wasm
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
		// TODO: load some config here?
		bootAddr := args[0].String()
		go quorum.StartQuorum(qChan, bootAddr)
		return js.ValueOf(true).Bool()
	}))

	js.Global().Set("JoinGroup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		seed := args[0].String()
		go func() {
			res, err := quorumAPI.JoinGroup([]byte(seed))
			if err != nil {
				println(err.Error())
			}
			retBytes, err := json.Marshal(res)
			if err != nil {
				println(err.Error())
			}
			println(string(retBytes))
		}()

		return js.ValueOf(true)
	}))

	js.Global().Set("GetGroups", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		res, err := quorumAPI.GetGroups()
		if err != nil {
			println(err.Error())
		}
		retBytes, err := json.Marshal(res)
		if err != nil {
			println(err.Error())
		}
		println(string(retBytes))
		return js.ValueOf(string(retBytes))
	}))

	js.Global().Set("GetBlockById", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		groupId := args[0].String()
		blockId := args[1].String()
		go func() {
			block, err := quorumAPI.GetBlockById(groupId, blockId)
			if err != nil {
				println(err.Error())
			}
			retBytes, err := json.Marshal(block)
			if err != nil {
				println(err.Error())
			}
			println(string(retBytes))
		}()

		return js.ValueOf(true)
	}))

	js.Global().Set("StopQuorum", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan != nil {
			close(qChan)
			qChan = nil
		}
		return js.ValueOf(true).Bool()
	}))

	js.Global().Set("WSTest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if qChan == nil {
			qChan = make(chan struct{}, 0)
		}
		WSTest()
		return js.ValueOf(true).Bool()
	}))

	js.Global().Set("IndexDBTest", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go quorum.IndexDBTest()
		return js.ValueOf(true).Bool()
	}))
}

func WSTest() {
	go func() {
		openSignal := make(chan struct{})
		ws := js.Global().Get("WebSocket").New("ws://127.0.0.1:4000")
		ws.Set("binaryType", "arraybuffer")

		ws.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			println("opened!!")
			close(openSignal)
			return nil
		}))

		messageHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			arrayBuffer := args[0].Get("data")
			data := arrayBufferToBytes(arrayBuffer)
			println(data)
			return nil
		})
		ws.Call("addEventListener", "message", messageHandler)

		// this will block, and websocket will never open
		// do not do this
		<-openSignal
		println("openSignal fired")

	}()
}

func arrayBufferToBytes(buffer js.Value) []byte {
	view := js.Global().Get("Uint8Array").New(buffer)
	dataLen := view.Length()
	data := make([]byte, dataLen)
	if js.CopyBytesToGo(data, view) != dataLen {
		panic("expected to copy all bytes")
	}
	return data
}

func main() {
	c := make(chan struct{}, 0)

	println("WASM Go Initialized")
	// register functions
	registerCallbacks()
	<-c
}
