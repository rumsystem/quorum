package ui

import (
	"runtime"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
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
		denyListHandler.Set("Shift+L", wrapQuorumKeyFn(focusAllowListView))
	} else {
		denyListHandler.Set("L", wrapQuorumKeyFn(focusAllowListView))
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
	// TODO:
}
