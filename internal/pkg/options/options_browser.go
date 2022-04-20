//go:build js && wasm
// +build js,wasm

package options

func (opt *NodeOptions) SetSignKeyMap(keyname, addr string) error {
	// no need to implemented in browser
	// to adapt the join/create group API
	return nil
}
