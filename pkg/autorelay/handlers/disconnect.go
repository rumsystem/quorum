package handlers

type DisconnectParam struct {
	FromPeer string `json:"from_peer"`
	ToPeer   string `json:"to_peer"`
}
type DisconnectResult struct {
	Ok bool `json:"ok"`
}
