//go:build !js
// +build !js

package handlers

import (
	"fmt"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

func CreateGroupUrl(baseUrl string, params *CreateGroupParam, nodeoptions *options.NodeOptions, appdb *appdata.AppDb) (*CreateGroupResult, error) {
	createGrpResult, err := CreateGroup(params, nodeoptions, appdb)
	if err != nil {
		return nil, err
	}

	// get chain api url
	jwtName := fmt.Sprintf("allow-%s", createGrpResult.GroupId)
	jwt, err := utils.NewJWTToken(
		jwtName,
		"node",
		[]string{createGrpResult.GroupId},
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
		GroupId: createGrpResult.GroupId,
	}
	return &result, err
}
