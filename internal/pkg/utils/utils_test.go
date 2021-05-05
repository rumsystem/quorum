package utils

import (
	maddr "github.com/multiformats/go-multiaddr"
	"testing"
)

func TestStringsToAddrs(t *testing.T) {
	m, _ := maddr.NewMultiaddr("/ip4/127.0.0.1/tcp/7002")
	excepted := []maddr.Multiaddr{m}
	m1, err := StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/7002"})
	if err != nil {
		t.Errorf("Test failed:%s", err)
	}
	if len(m1) != len(excepted) {
		t.Error("Test failed")
	}
	for i, mitem := range m1 {
		if !mitem.Equal(excepted[i]) {
			t.Errorf("Test failed: %s not %s", mitem, excepted[i])
		}
	}
}
