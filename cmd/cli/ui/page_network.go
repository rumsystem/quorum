package ui

import (
	"fmt"
	"net"
	"runtime"
	"sort"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/cmd/cli/model"
)

var networkPage = cview.NewFlex()
var networkPageLeft = cview.NewList()
var networkPageRight = cview.NewTextView()

var networkData = model.NetworkDataModel{
	TickerRunning: false,
	Counter:       0,
}

func NetworkPage() {
	rootPanels.ShowPanel("network")
	rootPanels.SendToFront("network")
	App.SetFocus(networkPageRight)

	NetworkRefreshAll()

	networkData.StartTicker(NetworkRefreshAll)
}

func NetworkPageExit() {
	networkData.StopTicker()
}

func networkPageInit() {
	networkPageLeft.SetTitle("Peers")
	networkPageLeft.SetBorder(true)

	networkPageRight.SetBorder(true)
	networkPageRight.SetRegions(true)
	networkPageRight.SetDynamicColors(true)
	networkPageRight.SetPadding(0, 1, 1, 1)

	initNetworkPageInputHandler()

	// short cut
	networkPage.AddItem(networkPageLeft, 0, 1, false)
	networkPage.AddItem(networkPageRight, 0, 2, false)

	rootPanels.AddPanel("network", networkPage, true, false)
}

func initNetworkPageInputHandler() {
	focusLeftView := func() { App.SetFocus(networkPageLeft) }
	focusRightView := func() { App.SetFocus(networkPageRight) }

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusLeftView))
	networkPage.SetInputCapture(pageInputHandler.Capture)

	leftViewHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		leftViewHandler.Set("Shift+L", wrapQuorumKeyFn(focusRightView))
	} else {
		leftViewHandler.Set("L", wrapQuorumKeyFn(focusRightView))
	}
	networkPageLeft.SetInputCapture(leftViewHandler.Capture)

	rightViewHandler := cbind.NewConfiguration()

	if runtime.GOOS == "windows" {
		rightViewHandler.Set("Shift+H", wrapQuorumKeyFn(focusLeftView))
	} else {
		rightViewHandler.Set("H", wrapQuorumKeyFn(focusLeftView))
	}

	networkPageRight.SetInputCapture(rightViewHandler.Capture)

}

func NetworkRefreshAll() {
	go func() {
		goNetworkPing()
	}()
}

func goNetworkPing() {
	pingInfo, err := api.Ping()
	checkFatalError(err)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return
		}
		Error("Failed to ping", err.Error())
	} else {
		oldLen := len(networkData.GetPeers())
		networkData.SetData(pingInfo)
		newLen := len(networkData.GetPeers())
		if oldLen != newLen {
			drawLeft()
		}
		drawRight()
	}
}

func drawLeft() {
	networkPageLeft.Clear()

	peers := networkData.GetPeers()
	sort.Strings(peers)
	for i, peer := range peers {
		item := cview.NewListItem(fmt.Sprintf("%s", peer))
		item.SetShortcut(rune('a' + i))
		networkPageLeft.AddItem(item)
	}
	if networkData.GetCurrentPeer() == "" && len(peers) > 0 {
		networkData.SetCurrentPeer(peers[0])
	}

	networkPageLeft.SetSelectedFunc(func(idx int, group *cview.ListItem) {
		networkData.SetCurrentPeer(peers[idx])
		drawRight()
	})
	App.Draw()
}

func drawRight() {
	peer := networkData.GetCurrentPeer()
	peerData := networkData.GetPeerData(peer)
	if peer != "" && peerData != nil {
		networkPageRight.Clear()
		fmt.Fprintf(networkPageRight, "[::b]%s[-:-:-]\n", peer)
		fmt.Fprintf(networkPageRight, "[::b]ping(ms): ")
		for _, rtt := range peerData.RTT {
			fmt.Fprintf(networkPageRight, "%d ", rtt)
		}
		fmt.Fprintf(networkPageRight, "[-:-:-]\n")
		fmt.Fprintf(networkPageRight, "\n")
		fmt.Fprintf(networkPageRight, "addrs:\n")
		for _, addr := range peerData.Addrs {
			fmt.Fprintf(networkPageRight, "\t%s\n", addr)

		}
		fmt.Fprintf(networkPageRight, "\n")
		fmt.Fprintf(networkPageRight, "protocols:\n")
		for _, protocol := range peerData.Protocols {
			if len(protocol) > 0 {
				fmt.Fprintf(networkPageRight, "\t%s\n", protocol)
			}
		}

		fmt.Fprintf(networkPageRight, "\n")
		fmt.Fprintf(networkPageRight, "connections:\n")
		for _, conn := range peerData.Connections {
			if len(conn.Protocol) > 0 {
				fmt.Fprintf(networkPageRight, "\t%s -> %s (%s)\n", conn.Local, conn.Remote, conn.Protocol)
			}
		}
		App.Draw()
	}
}
