package handlers

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/options"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

type CreateSignKeyParams struct {
	KeyName string `from:"key_name"        json:"key_name"        validate:"required,max=100,min=2" example:"demo app"`
}

type CreateSignKeyResult struct {
	KeyAlias string `json:"key_alias" validate:"required,uuid"`
	KeyName  string `json:"key_name" validate:"required,max=100,min=2" example:"demo app"`
	Pubkey   string `json:"pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
}

func CreateSignKey(params *CreateSignKeyParams, nodeoptions *options.NodeOptions) (*CreateSignKeyResult, error) {
	pubkey, err := localcrypto.InitSignKeyWithKeyName(params.KeyName, nodeoptions)
	if err != nil {
		return nil, err
	}

	//create key alias
	KeyAlias := uuid.New().String()

	password := os.Getenv("RUM_KSPASSWD")
	ks := localcrypto.GetKeystore()
	err = ks.NewAlias(KeyAlias, params.KeyName, password)
	if err != nil {
		return nil, err
	}

	return &CreateSignKeyResult{
		KeyAlias: KeyAlias,
		KeyName:  params.KeyName,
		Pubkey:   pubkey,
	}, nil
}

type GetPubkeyByKeyNameParams struct {
	KeyName string `from:"key_name"        json:"key_name"        validate:"required,max=100,min=2" example:"demo app"`
}

type GetPubkeyByKeyNameResult struct {
	Pubkey  string   `json:"pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
	KeyName string   `json:"key_name" validate:"required,max=100,min=2" example:"demo app"`
	Alias   []string `json:"alias" validate:"required"`
}

func GetPubkeyByKeyName(params *GetPubkeyByKeyNameParams, nodeoptions *options.NodeOptions) (*GetPubkeyByKeyNameResult, error) {
	ks := localcrypto.GetKeystore()

	pubkey, err := ks.GetEncodedPubkey(params.KeyName, localcrypto.Sign)
	if err != nil {
		return nil, err
	}

	alias := ks.GetAlias(params.KeyName)

	return &GetPubkeyByKeyNameResult{
		Pubkey:  pubkey,
		KeyName: params.KeyName,
		Alias:   alias,
	}, nil
}

type GetAllKeysParams struct {
}

type GetAllKeysResult struct {
	KeysList []*GetPubkeyByKeyNameResult `json:"keys_list" validate:"required"`
}

func GetAllKeys(params *GetAllKeysParams, nodeoptions *options.NodeOptions) (*GetAllKeysResult, error) {
	ks := localcrypto.GetKeystore()
	keyItems, err := ks.ListAll()
	if err != nil {
		return nil, err
	}

	fmt.Println("%x", keyItems)

	var result []*GetPubkeyByKeyNameResult
	for _, keyItem := range keyItems {
		pubkey, err := ks.GetEncodedPubkey(keyItem.Keyname, localcrypto.Sign)
		if err != nil {
			pubkey = "GET PUBKEY FAILED"
		}

		resultItem := &GetPubkeyByKeyNameResult{
			Pubkey:  pubkey,
			KeyName: keyItem.Keyname,
			Alias:   keyItem.Alias,
		}
		result = append(result, resultItem)
	}

	return &GetAllKeysResult{
		KeysList: result,
	}, nil
}

/*

type RmSignKeyByKeyNameParams struct {
	KeyName string `from:"key_name"        json:"key_name"        validate:"required,max=100,min=2" example:"demo app"`
}

type RmSignKeyByAliasParams struct {
	KeyAlias string `from:"key_alias"        json:"key_alias"        validate:"required,uuid" example:"demo app"`
}

type RmSignKeyResult struct {
	KeyAlias []string `json:"key_alias" validate:"required,uuid"`
	KeyName  string   `json:"key_name" validate:"required,max=100,min=2" example:"demo app"`
	Pubkey   string   `json:"pubkey" validate:"required" example:"CAISIQO7ury6x7aWpwUVn6mj2dZFqme3BAY5xDkYjqW/EbFFcA=="`
}

func RmSignKeyByKeyName(params *RmSignKeyByKeyNameParams, nodeoptions *options.NodeOptions) (*RmSignKeyResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	ks := localcrypto.GetKeystore()
	password := os.Getenv("RUM_KSPASSWD")

	//remove all alias associated with this key
	alias := ks.GetAlias(params.KeyName)
	if len(alias) != 0 {
		for _, v := range alias {
			err := ks.UnAlias(v, password)
			if err != nil {
				return nil, err
			}
		}
	}

	err := ks.RemoveKey(params.KeyName, localcrypto.Sign)
	if err != nil {
		return nil, err
	}

	return &RmSignKeyResult{
		KeyAlias: alias,
		KeyName:  params.KeyName,
	}, nil

}
*/
