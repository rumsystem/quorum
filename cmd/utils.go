package cmd

import (
	"fmt"
	"os"
	"strings"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

// reutrn EBUSY if LOCK is exist
func CheckLockError(err error) {
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "Another process is using this Badger database.") {
			logger.Errorf(errStr)
			os.Exit(16)
		}
	}
}

type InitKeystoreParam struct {
	KeystoreName   string
	KeystoreDir    string
	KeystorePwd    string
	DefaultKeyName string
	ConfigDir      string
	PeerName       string
}

func InitDefaultKeystore(config InitKeystoreParam, nodeoptions *options.NodeOptions) (localcrypto.Keystore, *ethkeystore.Key, error) {
	signkeycount, err := localcrypto.InitKeystore(config.KeystoreName, config.KeystoreDir)
	ksi := localcrypto.GetKeystore()
	if err != nil {
		return nil, nil, err
	}

	ks, ok := ksi.(*localcrypto.DirKeyStore)
	//TODO: test other keystore type?
	//if there are no other keystores, exit and show error info.
	if ok == false {
		return nil, nil, fmt.Errorf("unknown keystore type")
	}

	password := config.KeystorePwd

	if signkeycount > 0 {
		if password == "" {
			password, err = localcrypto.PassphrasePromptForUnlock()
		}
		err = ks.Unlock(nodeoptions.SignKeyMap, password)
		if err != nil {
			return nil, nil, err
		}
	} else {
		if password == "" {
			password, err = localcrypto.PassphrasePromptForEncryption()
			if err != nil {
				return nil, nil, err
			}
			fmt.Println("Please keeping your password safe, We can't recover or reset your password.")
			fmt.Println("Your password:", password)
			fmt.Println("After saving the password, press any key to continue.")
			os.Stdin.Read(make([]byte, 1))
		}
		signkeyhexstr, err := localcrypto.LoadEncodedKeyFrom(config.ConfigDir, config.PeerName, "txt")
		if err != nil {
			return nil, nil, err
		}
		var addr string
		if signkeyhexstr != "" {
			addr, err = ks.Import(config.DefaultKeyName, signkeyhexstr, localcrypto.Sign, password)
		} else {
			addr, err = ks.NewKey(config.DefaultKeyName, localcrypto.Sign, password)
			if err != nil {
				return nil, nil, err
			}
		}

		if addr == "" {
			return nil, nil, fmt.Errorf("Load or create new signkey failed")
		}
		err = nodeoptions.SetSignKeyMap(config.DefaultKeyName, addr)
		if err != nil {
			return nil, nil, err
		}
		err = ks.Unlock(nodeoptions.SignKeyMap, password)
		if err != nil {
			return nil, nil, err
		}

		fmt.Printf("load signkey: %d press any key to continue...\n", signkeycount)

		_, err = ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(config.DefaultKeyName))
		signkeycount = ks.UnlockedKeyCount(localcrypto.Sign)
		if signkeycount == 0 {
			return nil, nil, fmt.Errorf("load signkey error, exit... %s", err)
		}
	}
	key, err := ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(config.DefaultKeyName))
	if err != nil {
		return nil, nil, fmt.Errorf("ks.GetKeyFromUnlocked failed: %s", err)
	}

	defaultkey, ok := key.(*ethkeystore.Key)
	if ok == false {
		return nil, nil, fmt.Errorf("load default key error, exit...")
	}
	return ks, defaultkey, nil
}

func InitRelayNodeKeystore(config cli.RelayNodeFlag, defaultKeyName string, relayNodeOpt *options.RelayNodeOptions) (localcrypto.Keystore, *ethkeystore.Key, error) {
	signkeycount, err := localcrypto.InitKeystore(config.KeyStoreName, config.KeyStoreDir)
	ksi := localcrypto.GetKeystore()
	if err != nil {
		return nil, nil, err
	}

	ks, ok := ksi.(*localcrypto.DirKeyStore)
	if ok == false {
		return nil, nil, fmt.Errorf("unknown keystore type")
	}

	password := config.KeyStorePwd

	if signkeycount > 0 {
		if password == "" {
			password, err = localcrypto.PassphrasePromptForUnlock()
		}
		err = ks.Unlock(relayNodeOpt.SignKeyMap, password)
		if err != nil {
			return nil, nil, err
		}
	} else {
		if password == "" {
			password, err = localcrypto.PassphrasePromptForEncryption()
			if err != nil {
				return nil, nil, err
			}
			fmt.Println("Please keeping your password safe, We can't recover or reset your password.")
			fmt.Println("Your password:", password)
			fmt.Println("After saving the password, press any key to continue.")
			os.Stdin.Read(make([]byte, 1))
		}
		signkeyhexstr, err := localcrypto.LoadEncodedKeyFrom(config.ConfigDir, config.PeerName, "txt")
		if err != nil {
			return nil, nil, err
		}
		var addr string
		if signkeyhexstr != "" {
			addr, err = ks.Import(defaultKeyName, signkeyhexstr, localcrypto.Sign, password)
		} else {
			addr, err = ks.NewKey(defaultKeyName, localcrypto.Sign, password)
			if err != nil {
				return nil, nil, err
			}
		}

		if addr == "" {
			return nil, nil, fmt.Errorf("Load or create new signkey failed")
		}

		relayNodeOpt.SetSignKeyMap(defaultKeyName, addr)

		if err != nil {
			return nil, nil, err
		}
		err = ks.Unlock(relayNodeOpt.SignKeyMap, password)
		if err != nil {
			return nil, nil, err
		}

		fmt.Printf("load signkey: %d press any key to continue...\n", signkeycount)

		_, err = ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(defaultKeyName))
		signkeycount = ks.UnlockedKeyCount(localcrypto.Sign)
		if signkeycount == 0 {
			return nil, nil, fmt.Errorf("load signkey error, exit... %s", err)
		}
	}
	key, err := ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(defaultKeyName))
	defaultkey, ok := key.(*ethkeystore.Key)
	if ok == false {
		return nil, nil, fmt.Errorf("load default key error, exit...")
	}
	return ks, defaultkey, nil
}
