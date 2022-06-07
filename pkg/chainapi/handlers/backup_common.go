package handlers

type QuorumWasmExportObject struct {
	Keystore []string    `json:"keystore"`
	Seeds    []GroupSeed `json:"seeds"`
}
