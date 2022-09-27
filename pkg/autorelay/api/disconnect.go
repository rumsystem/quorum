package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/autorelay/handlers"
)

func (h *RelayServerHandler) Disconnect(c echo.Context) (err error) {
	param := handlers.DelBlacklistParam{}
	if err := c.Bind(&param); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	result := handlers.DisconnectResult{Ok: true}

	relay := h.node.GetRelay()
	if relay != nil {
		//TODO upgrade the DisconnectByPeerID
		//type *relay.Relay has no field or method DisconnectByPeerID

		//from, err := peer.Decode(param.FromPeer)
		//if err != nil {
		//	return rumerrors.NewBadRequestError(err.Error())
		//}
		//to, err := peer.Decode(param.ToPeer)
		//if err != nil {
		//	return rumerrors.NewBadRequestError(err.Error())
		//}
		//relay.DisconnectByPeerID(from, to)
	}

	return c.JSON(http.StatusOK, result)
}
