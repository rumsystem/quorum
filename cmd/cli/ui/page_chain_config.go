package ui

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

var chainConfigPage = cview.NewFlex()
var chainAllowListView = cview.NewTextView()
var chainDenyListView = cview.NewTextView()

var PANNEL_CHAIN_CONFIG_PAGE = "chain.config"

var focusAllowListView = func() { App.SetFocus(chainAllowListView) }
var focusDenyListView = func() { App.SetFocus(chainDenyListView) }
var focusChainConfigRootView = func() { App.SetFocus(chainConfigPage) }

func chainConfigPageInit() {
	chainAllowListView.SetTitle("Allow List")
	chainAllowListView.SetBorder(true)
	chainAllowListView.SetRegions(true)
	chainAllowListView.SetDynamicColors(true)

	chainDenyListView.SetTitle("Deny List")
	chainDenyListView.SetBorder(true)
	chainDenyListView.SetRegions(true)
	chainDenyListView.SetDynamicColors(true)

	chainConfigPage.AddItem(chainAllowListView, 0, 1, false)
	chainConfigPage.AddItem(chainDenyListView, 0, 1, false)

	initChainConfigPageHandlers()
	rootPanels.AddPanel(PANNEL_CHAIN_CONFIG_PAGE, chainConfigPage, true, false)
}

func initChainConfigPageHandlers() {
	denyListHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		denyListHandler.Set("Shift+H", wrapQuorumKeyFn(focusAllowListView))
	} else {
		denyListHandler.Set("H", wrapQuorumKeyFn(focusAllowListView))
	}
	denyListHandler.Set("Esc", wrapQuorumKeyFn(focusChainConfigRootView))
	chainDenyListView.SetInputCapture(denyListHandler.Capture)

	allowListHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		allowListHandler.Set("Shift+L", wrapQuorumKeyFn(focusDenyListView))
	} else {
		allowListHandler.Set("L", wrapQuorumKeyFn(focusDenyListView))
	}
	allowListHandler.Set("Esc", wrapQuorumKeyFn(focusChainConfigRootView))
	chainAllowListView.SetInputCapture(allowListHandler.Capture)
}

func ChainConfigPage(groupId string) {
	lastPannel, lastView := rootPanels.GetFrontPanel()

	focusLastView := func() {
		rootPanels.ShowPanel(lastPannel)
		rootPanels.SendToFront(lastPannel)
		App.SetFocus(lastView)
	}

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusAllowListView))
	pageInputHandler.Set("Esc", wrapQuorumKeyFn(focusLastView))
	pageInputHandler.Set("r", wrapQuorumKeyFn(func() {
		ChainConfigRefreshAll(groupId)
	}))

	chainConfigPage.SetInputCapture(pageInputHandler.Capture)

	rootPanels.ShowPanel(PANNEL_CHAIN_CONFIG_PAGE)
	rootPanels.SendToFront(PANNEL_CHAIN_CONFIG_PAGE)
	App.SetFocus(chainConfigPage)

	ChainConfigRefreshAll(groupId)
}

func ChainConfigRefreshAll(groupId string) {
	go goGetChainAllowList(groupId)
	go goGetChainDenyList(groupId)
	go goGetChainAuthMode(groupId)
}

func goGetChainAllowList(groupId string) {
	data, err := api.GetChainAllowList(groupId)
	checkFatalError(err)

	renderListView(data, chainAllowListView)
}

func goGetChainDenyList(groupId string) {
	data, err := api.GetChainDenyList(groupId)
	checkFatalError(err)

	renderListView(data, chainDenyListView)
}

func renderListView(data []*handlers.ChainSendTrxRuleListItem, view *cview.TextView) {
	view.Clear()

	for i, each := range data {
		fmt.Fprintf(view, "[\"%d\"]%s\n", i, time.Unix(0, each.TimeStamp))
		fmt.Fprintf(view, "Pubkey: %s\n", each.Pubkey)
		fmt.Fprintf(view, "GroupOwnerPubkey: %s\n", each.GroupOwnerPubkey)
		fmt.Fprintf(view, "GroupOwnerSign: %s\n", each.GroupOwnerSign)
		fmt.Fprintf(view, "Trx Type: %s\n", each.TrxType)
		fmt.Fprintf(view, "Memo: %s\n", each.Memo)
		fmt.Fprintf(view, "\n\n")
	}

	focusLast := func() {
		minNum := 0
		maxNum := len(data) - 1

		curSelection := view.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag += 1
		}
		if tag >= minNum && tag <= maxNum {
			view.Highlight(strconv.Itoa(tag))
			view.ScrollToHighlight()
		}
	}
	focusNext := func() {
		minNum := 0
		maxNum := len(data) - 1

		curSelection := view.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag -= 1
		}
		if tag >= minNum && tag <= maxNum {
			view.Highlight(strconv.Itoa(tag))
			view.ScrollToHighlight()
		}
	}

	view.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			view.Highlight("")
			cmdInput.SetLabel("")
			cmdInput.SetText("")
		case tcell.KeyEnter:
			curSelection := view.GetHighlights()
			if len(curSelection) > 0 {

			}

		case tcell.KeyTab:
			focusNext()
		case tcell.KeyBacktab:
			focusLast()
		default:
		}
	})
}

func goGetChainAuthMode(groupId string) {
	// TODO:
	// POST
	// ANNOUNCE
	// REQ_BLOCK_FORWARD
	// REQ_BLOCK_BACKWARD
	// ASK_PEERID
	// BLOCK_SYNCED
	// BLOCK_PRODUCED
}
