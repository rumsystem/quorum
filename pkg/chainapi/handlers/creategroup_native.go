//go:build !js
// +build !js

package handlers

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	guuid "github.com/google/uuid"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	"github.com/rumsystem/rumchaindata/pkg/pb"
)

func CreateGroupUrl(baseUrl string, params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*CreateGroupResult, error) {
	createGrpResult, err := CreateGroup(params, nodeoptions, appdb)
	if err != nil {
		return nil, err
	}

	// get chain api url
	jwtName := fmt.Sprintf("allow-%s", groupid.String())
	jwt, err := utils.NewJWTToken(
		jwtName,
		"node",
		[]string{groupid.String()},
		nodeoptions.JWTKey,
		time.Now().Add(time.Hour*24*365*5), // 5 years
	)
	if err != nil {
		return nil, err
	}
	if err := nodeoptions.SetJWTTokenMap(jwtName, jwt); err != nil {
		return nil, err
	}
	chainapiUrl, err := utils.GetChainapiURL(baseUrl, jwt)
	if err != nil {
		return nil, err
	}

	// convert group seed to url
	seedurl, err := GroupSeedToUrl(1, []string{chainapiUrl}, createGrpResult)
	result := CreateGroupResult{
		Seed:    seedurl,
		GroupId: groupid.String(),
	}
	return &result, err
}
