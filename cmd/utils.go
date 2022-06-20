package main

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
			mainlog.Errorf(errStr)
			os.Exit(16)
		}
	}
}

func InitDefaultKeystore(config cli.Config, nodeoptions *options.NodeOptions) (localcrypto.Keystore, *ethkeystore.Key, error) {
	signkeycount, err := localcrypto.InitKeystore(config.KeyStoreName, config.KeyStoreDir)
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

	password := os.Getenv("RUM_KSPASSWD")

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
			addr, err = ks.Import(DEFAUT_KEY_NAME, signkeyhexstr, localcrypto.Sign, password)
		} else {
			addr, err = ks.NewKey(DEFAUT_KEY_NAME, localcrypto.Sign, password)
			if err != nil {
				return nil, nil, err
			}
		}

		if addr == "" {
			return nil, nil, fmt.Errorf("Load or create new signkey failed")
		}
		err = nodeoptions.SetSignKeyMap(DEFAUT_KEY_NAME, addr)
		if err != nil {
			return nil, nil, err
		}
		err = ks.Unlock(nodeoptions.SignKeyMap, password)
		if err != nil {
			return nil, nil, err
		}

		fmt.Printf("load signkey: %d press any key to continue...\n", signkeycount)

		_, err = ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(DEFAUT_KEY_NAME))
		signkeycount = ks.UnlockedKeyCount(localcrypto.Sign)
		if signkeycount == 0 {
			return nil, nil, fmt.Errorf("load signkey error, exit... %s", err)
		}
	}
	key, err := ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(DEFAUT_KEY_NAME))
	defaultkey, ok := key.(*ethkeystore.Key)
	if ok == false {
		return nil, nil, fmt.Errorf("load default key error, exit...")
	}
	return ks, defaultkey, nil
}

func InitRelayNodeKeystore(config cli.RelayNodeConfig, relayNodeOpt *options.RelayNodeOptions) (localcrypto.Keystore, *ethkeystore.Key, error) {
	signkeycount, err := localcrypto.InitKeystore(config.KeyStoreName, config.KeyStoreDir)
	ksi := localcrypto.GetKeystore()
	if err != nil {
		return nil, nil, err
	}

	ks, ok := ksi.(*localcrypto.DirKeyStore)
	if ok == false {
		return nil, nil, fmt.Errorf("unknown keystore type")
	}

	password := os.Getenv("RUM_KSPASSWD")

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
			addr, err = ks.Import(DEFAUT_KEY_NAME, signkeyhexstr, localcrypto.Sign, password)
		} else {
			addr, err = ks.NewKey(DEFAUT_KEY_NAME, localcrypto.Sign, password)
			if err != nil {
				return nil, nil, err
			}
		}

		if addr == "" {
			return nil, nil, fmt.Errorf("Load or create new signkey failed")
		}

		relayNodeOpt.SetSignKeyMap(DEFAUT_KEY_NAME, addr)

		if err != nil {
			return nil, nil, err
		}
		err = ks.Unlock(relayNodeOpt.SignKeyMap, password)
		if err != nil {
			return nil, nil, err
		}

		fmt.Printf("load signkey: %d press any key to continue...\n", signkeycount)

		_, err = ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(DEFAUT_KEY_NAME))
		signkeycount = ks.UnlockedKeyCount(localcrypto.Sign)
		if signkeycount == 0 {
			return nil, nil, fmt.Errorf("load signkey error, exit... %s", err)
		}
	}
	key, err := ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(DEFAUT_KEY_NAME))
	defaultkey, ok := key.(*ethkeystore.Key)
	if ok == false {
		return nil, nil, fmt.Errorf("load default key error, exit...")
	}
	return ks, defaultkey, nil
}
