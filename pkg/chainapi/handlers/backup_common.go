package handlers

type QuorumWasmExportObject struct {
	Keystore []string            `json:"keystore"`
	Seeds    []CreateGroupResult `json:"seeds"`
}
