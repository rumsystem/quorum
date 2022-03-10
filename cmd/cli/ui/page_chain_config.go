package ui

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

var chainConfigPage = cview.NewFlex()
var chainAuthModeView = cview.NewTable()
var chainAllowListView = cview.NewTextView()
var chainDenyListView = cview.NewTextView()

var PANNEL_CHAIN_CONFIG_PAGE = "chain.config"

var chainConfigPageHelpText = strings.TrimSpace(
	"Shortcuts:\n" +
		"Enter to select left pannel.\n" +
		"Esc to go back.\n" +
		"Shift + h/j/k/l to naviagate between pannels.\n" +
		"Tab / Shift + Tab to scroll.\n" +
		"Enter to open operation modal of selected items.\n" +
		"Press `r` to refresh data.\n",
)
var chainAuthModes sync.Map

var chainConfigTrxTypes []string = []string{"POST", "ANNOUNCE", "REQ_BLOCK_FORWARD", "REQ_BLOCK_BACKWARD", "ASK_PEERID", "BLOCK_SYNCED", "BLOCK_PRODUCED"}

var focusAllowListView = func() { App.SetFocus(chainAllowListView) }
var focusDenyListView = func() { App.SetFocus(chainDenyListView) }
var focusAuthModeView = func() { App.SetFocus(chainAuthModeView) }
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

	chainAuthModeView.SetBorder(true)

	rightFlex := cview.NewFlex()
	rightFlex.SetDirection(cview.FlexRow)
	rightFlex.AddItem(chainAllowListView, 0, 1, false)
	rightFlex.AddItem(chainDenyListView, 0, 1, false)

	leftFlex := cview.NewFlex()
	leftFlex.SetDirection(cview.FlexRow)
	leftFlex.SetBorder(true)
	leftFlex.SetTitle("Chain Auth Mode")
	helpView := cview.NewTextView()
	helpView.SetText(chainConfigPageHelpText)
	leftFlex.AddItem(helpView, 0, 1, false)
	leftFlex.AddItem(chainAuthModeView, 0, 2, false)

	chainConfigPage.AddItem(leftFlex, 0, 1, false)
	chainConfigPage.AddItem(rightFlex, 0, 2, false)

	initChainConfigPageHandlers()
	rootPanels.AddPanel(PANNEL_CHAIN_CONFIG_PAGE, chainConfigPage, true, false)
}

func initChainConfigPageHandlers() {
	denyListHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		denyListHandler.Set("Shift+K", wrapQuorumKeyFn(focusAllowListView))
		denyListHandler.Set("Shift+H", wrapQuorumKeyFn(focusAuthModeView))
	} else {
		denyListHandler.Set("K", wrapQuorumKeyFn(focusAllowListView))
		denyListHandler.Set("H", wrapQuorumKeyFn(focusAuthModeView))
	}
	denyListHandler.Set("Esc", wrapQuorumKeyFn(focusChainConfigRootView))
	chainDenyListView.SetInputCapture(denyListHandler.Capture)

	allowListHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		allowListHandler.Set("Shift+J", wrapQuorumKeyFn(focusDenyListView))
		allowListHandler.Set("Shift+H", wrapQuorumKeyFn(focusAuthModeView))
	} else {
		allowListHandler.Set("J", wrapQuorumKeyFn(focusDenyListView))
		allowListHandler.Set("H", wrapQuorumKeyFn(focusAuthModeView))
	}
	allowListHandler.Set("Esc", wrapQuorumKeyFn(focusChainConfigRootView))
	chainAllowListView.SetInputCapture(allowListHandler.Capture)

	authModeViewHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		authModeViewHandler.Set("Shift+L", wrapQuorumKeyFn(focusAllowListView))
	} else {
		authModeViewHandler.Set("L", wrapQuorumKeyFn(focusAllowListView))
	}
	authModeViewHandler.Set("Esc", wrapQuorumKeyFn(focusChainConfigRootView))
	chainAuthModeView.SetInputCapture(authModeViewHandler.Capture)
}

func ChainConfigPage(groupId string) {
	lastPannel, lastView := rootPanels.GetFrontPanel()

	focusLastView := func() {
		rootPanels.ShowPanel(lastPannel)
		rootPanels.SendToFront(lastPannel)
		App.SetFocus(lastView)
	}

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusAuthModeView))
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

	chainAuthModeView.Clear()

	for i, trxType := range chainConfigTrxTypes {
		// refresh
		go goGetChainAuthMode(groupId, trxType)
		color := tcell.ColorYellow.TrueColor()
		cell := cview.NewTableCell(trxType)
		cell.SetTextColor(color)
		cell.SetAlign(cview.AlignLeft)
		chainAuthModeView.SetCell(i, 0, cell)
	}

	chainAuthModeView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			chainAuthModeView.SetSelectable(true, false)
		}
		if key == tcell.KeyTab {
			// select next row
			row, col := chainAuthModeView.GetSelection()
			row += 1
			if row >= 0 && row < len(chainConfigTrxTypes) {
				chainAuthModeView.Select(row, col)
			}
		}
		if key == tcell.KeyBacktab {
			// select last row
			row, col := chainAuthModeView.GetSelection()
			row -= 1
			if row >= 0 && row <= len(chainConfigTrxTypes) {
				chainAuthModeView.Select(row, col)
			}
		}
	})
	chainAuthModeView.SetSelectedFunc(func(row int, column int) {
		for i := 0; i < chainAuthModeView.GetColumnCount(); i++ {
			chainAuthModeView.GetCell(row, i).SetTextColor(tcell.ColorRed.TrueColor())
		}
		if row >= 0 && row < len(chainConfigTrxTypes) {
			trxType := chainConfigTrxTypes[row]
			authTypeI, _ := chainAuthModes.Load(trxType)
			authType := authTypeI.(string)
			if len(trxType) > 0 && len(authType) > 0 {
				ChainAuthModeForm(groupId, trxType, authType)
			}
		}
		chainAuthModeView.SetSelectable(false, false)
	})
}

func goGetChainAllowList(groupId string) {
	data, err := api.GetChainAllowList(groupId)
	checkFatalError(err)

	renderListView(groupId, "upd_alw_list", data, chainAllowListView)
}

func goGetChainDenyList(groupId string) {
	data, err := api.GetChainDenyList(groupId)
	checkFatalError(err)

	renderListView(groupId, "upd_dny_list", data, chainDenyListView)
}

func renderListView(groupId, listType string, data []*handlers.ChainSendTrxRuleListItem, view *cview.TextView) {
	view.Clear()

	for i, each := range data {
		fmt.Fprintf(view, "[\"%d\"]%s\n", i, time.Unix(0, each.TimeStamp))
		fmt.Fprintf(view, "Pubkey: %s\n", each.Pubkey)
		fmt.Fprintf(view, "GroupOwnerPubkey: %s\n", each.GroupOwnerPubkey)
		fmt.Fprintf(view, "GroupOwnerSign: %s\n", each.GroupOwnerSign)
		fmt.Fprintf(view, "Trx Types: %v\n", each.TrxType)
		fmt.Fprintf(view, "Memo: %s\n", each.Memo)
		fmt.Fprintf(view, "\n\n")
	}

	focusNext := func() {
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
	focusLast := func() {
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
				idx, _ := strconv.Atoi(curSelection[0])
				curData := data[idx]
				defaultAction := "add"
				ChainAuthListForm(groupId, listType, defaultAction, curData.Pubkey, curData.TrxType)
			}

		case tcell.KeyTab:
			focusNext()
		case tcell.KeyBacktab:
			focusLast()
		default:
		}
	})
}

func goGetChainAuthMode(groupId string, trxType string) {
	data, err := api.GetChainAuthMode(groupId, trxType)
	checkFatalError(err)

	idx := -1
	for i, trxType := range chainConfigTrxTypes {
		if trxType == data.TrxType {
			idx = i
			break
		}
	}
	if idx >= 0 {
		chainAuthModes.Store(data.TrxType, data.AuthType)
		color := tcell.ColorWhite.TrueColor()
		cell := cview.NewTableCell(data.AuthType)
		cell.SetTextColor(color)
		chainAuthModeView.SetCell(idx, 1, cell)
	}
}
