package ui

import (
	"code.rocketnine.space/tslocum/cview"
	"github.com/rumsystem/quorum/cmd/cli/model"
)

var pubqueuePage = cview.NewFlex()
var pubqueuePageLeft = cview.NewList()      // groups
var pubqueuePageRight = cview.NewTextView() // trx

const PUBQUEUE_PAGE = "pubqueue"

var pubqueueData = model.PubqueueDataModel{
	TickerCh:      make(chan struct{}),
	TickerRunning: false,
}

func pubqueuePageInit() {
	pubqueuePageLeft.SetTitle("Groups")
	pubqueuePageLeft.SetBorder(true)

	pubqueuePageRight.SetBorder(true)
	pubqueuePageRight.SetRegions(true)
	pubqueuePageRight.SetDynamicColors(true)
	pubqueuePageRight.SetPadding(0, 1, 1, 1)

	initBlocksPageInputHandler()

	// short cut
	pubqueuePage.AddItem(pubqueuePageLeft, 0, 1, false)
	pubqueuePage.AddItem(pubqueuePageRight, 0, 2, false)

	rootPanels.AddPanel(PUBQUEUE_PAGE, pubqueuePage, true, false)
}

func Pubqueue() {
	rootPanels.ShowPanel(PUBQUEUE_PAGE)
	rootPanels.SendToFront(PUBQUEUE_PAGE)
	App.SetFocus(pubqueuePageRight)

	PubqueueRefreshAll()
}

func PubqueueExit() {
}

func PubqueueRefreshAll() {

}
