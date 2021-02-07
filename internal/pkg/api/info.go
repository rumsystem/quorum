package api

import (
	"fmt"
	"net/http"
	//"encoding/json"
    //kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/labstack/echo/v4"
)



func (h *Handler) Info(c echo.Context) (err error) {
	//h.Node.Ddht.LAN.RoutingTable().Print()
	//h.Node.Ddht.WAN.RoutingTable().Print()
    output := make(map[string]string)
    lanpeerids := h.Node.Ddht.LAN.RoutingTable().ListPeers()
    for idx , peerid := range lanpeerids {
        output[fmt.Sprintf("LAN_%d",idx)] = peerid.Pretty()
    }

    wanpeerids := h.Node.Ddht.WAN.RoutingTable().ListPeers()
    for idx , peerid := range wanpeerids {
        output[fmt.Sprintf("WAN_%d",idx)] = peerid.Pretty()
    }
    return c.JSON(http.StatusOK, output)
}
