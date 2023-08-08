package handlers

import (
	"os"

	"github.com/go-playground/validator/v10"
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

func CreateSignKey(params *CreateSignKeyParams, nodeoptions *options.NodeOptions) (*CreateSignKeyResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

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

/*
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
