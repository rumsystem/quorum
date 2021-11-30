package crypto

import (
	"fmt"
	"testing"

	"github.com/rumsystem/quorum/internal/pkg/utils"
)

func TestKeyType(t *testing.T) {
	var strList []string
	for i := 0; i < 10; i++ {
		strList = append(strList, utils.GetRandomStr(2*i))
	}

	ktPrefixs := map[KeyType]string{
		Encrypt: "encrypt_",
		Sign:    "sign_",
	}

	for kt, prefix := range ktPrefixs {
		for _, s := range strList {
			if kt.Prefix() != prefix {
				t.Errorf("key type prefix error, expect %s, got %s", prefix, kt.Prefix())
			}

			if kt.NameString(s) != fmt.Sprintf("%s%s", prefix, s) {
				t.Errorf("key type name string error, expect %s, got %s", fmt.Sprintf("%s%s", prefix, s), kt.NameString(s))
			}
		}
	}
}
