package handlers

import (
	"fmt"
	"strings"
)

const (
	PREFIX_ALLOW_RESERVE = "AllowReserve"
	PREFIX_ALLOW_CONNECT = "AllowConnect"
	PREFIX_BLACKLIST     = "Blacklist"
)

func GetAllowConnectKey(peer string) string {
	return fmt.Sprintf("%s_%s", PREFIX_ALLOW_CONNECT, peer)
}

func GetAllowReserveKey(peer string) string {
	return fmt.Sprintf("%s_%s", PREFIX_ALLOW_RESERVE, peer)
}

func GetBlackListPrefixKey(serverPeer string) string {
	return fmt.Sprintf("%s_%s", PREFIX_BLACKLIST, serverPeer)
}

func GetBlackListKey(serverPeer string, banPeer string) string {
	// like `Blacklist_$from_$to`
	return fmt.Sprintf("%s_%s", GetBlackListPrefixKey(serverPeer), banPeer)
}

func GetBlacklistPeerFromKeyByPrefix(key string, prefix string) string {
	return strings.ReplaceAll(key, fmt.Sprintf("%s_", prefix), "")
}
