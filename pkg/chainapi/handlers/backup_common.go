package handlers

type QuorumWasmExportObject struct {
	Keystore []string `json:"keystore"`
	Seeds    []string `json:"seeds"`
}
