//go:build !js
// +build !js

package handlers

import (
	"fmt"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

func GetOrCreateGroupNodeJwt(groupId string) (string, error) {
	nodeoptions := options.GetNodeOptions()

	jwtName := fmt.Sprintf("allow-%s", groupId)
	jwt := nodeoptions.GetJWTTokenMap(jwtName)
	if jwt == "" {
		var err error
		jwt, err = utils.NewJWTToken(
			jwtName,
			"node",
			[]string{groupId},
			nodeoptions.JWTKey,
			time.Now().Add(time.Hour*24*365*5), // 5 years
		)
		if err != nil {
			return "", err
		}
		if err := nodeoptions.SetJWTTokenMap(jwtName, jwt); err != nil {
			return "", err
		}
	}

	if jwt == "" {
		return "", rumerrors.ErrInvalidJWT
	}

	return jwt, nil
}

func CreateGroupUrl(baseUrl string, params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*CreateGroupResult, error) {
	createGrpResult, err := CreateGroup(params, nodeoptions, appdb)
	if err != nil {
		return nil, err
	}

	jwt, err := GetOrCreateGroupNodeJwt(createGrpResult.GroupId)
	if err != nil {
		return nil, err
	}
	if jwt == "" {
		return nil, rumerrors.ErrInvalidJWT
	}

	// get chain api url
	chainapiUrl, err := utils.GetChainapiURL(baseUrl, jwt)
	if err != nil {
		return nil, err
	}

	// convert group seed to url
	seedurl, err := GroupSeedToUrl(1, []string{chainapiUrl}, createGrpResult)
	if err != nil {
		return nil, err
	}

	result := CreateGroupResult{
		Seed:    seedurl,
		GroupId: createGrpResult.GroupId,
	}
	return &result, nil
}
