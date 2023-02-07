package api

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func clearGroup(api string, payload handlers.ClearGroupDataParam) (*handlers.ClearGroupDataResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := string(payloadBytes[:])
	urlSuffix := fmt.Sprintf("/api/v1/group/clear")
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.ClearGroupDataResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		logger.Errorf("clear group response: %s, result: %+v", resp, result)
		return nil, err
	}

	return &result, nil
}
