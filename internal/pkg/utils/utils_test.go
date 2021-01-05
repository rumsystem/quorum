package utils

import (
	"testing"

	maddr "github.com/multiformats/go-multiaddr"
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

func TestGetRandomStr(t *testing.T) {
	for i := 0; i < 20; i++ {
		s := GetRandomStr(i)
		if len(s) != i {
			t.Errorf("Test failed, len(%s) != %d", s, i)
		}
	}

	a := GetRandomStr(10)
	b := GetRandomStr(10)
	if a == b {
		t.Errorf("random two string are equal: %s, %s", a, b)
	}
}
