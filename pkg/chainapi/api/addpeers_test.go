package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func addPeers(api string, payload handlers.AddPeerParam) (*handlers.AddPeerResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadStr := string(payloadByte)

	urlSuffix := "/api/v1/network/peers"
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	var result handlers.AddPeerResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	if result.ErrCount > 0 {
		return nil, fmt.Errorf("%s", result.Errs)
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestAddPeers(t *testing.T) {
	t.Parallel()

	payload := handlers.AddPeerParam{"/ip4/192.99.101.35/tcp/10666/p2p/16Uiu2HAm8FNWrm1Zqe29Xe4EoHTzKG5UnVD2gCDB7D4b3SM5AvGi"}

	if _, err := addPeers(peerapi, payload); err != nil {
		t.Errorf("addPeers failed: %s, payload: %+v", err, payload)
	}
}
