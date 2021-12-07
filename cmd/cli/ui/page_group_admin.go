package ui

import (
	"runtime"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
)

var adminPage = cview.NewFlex()
var adminPageLeft = cview.NewList()  // users
var adminPageRight = cview.NewList() // producers

func adminPageInit() {
	adminPageLeft.SetTitle("User Requests")
	adminPageLeft.SetBorder(true)
	adminPageRight.SetTitle("Producer Requests")
	adminPageRight.SetBorder(true)

	initAdminPageInputHandler()

	// short cut
	adminPage.AddItem(adminPageLeft, 0, 1, false)
	adminPage.AddItem(adminPageRight, 0, 1, false)

	rootPanels.AddPanel("admin", adminPage, true, false)
}

var focusLeftView = func() { App.SetFocus(adminPageLeft) }
var focusRightView = func() { App.SetFocus(adminPageRight) }

func initAdminPageInputHandler() {
	leftViewHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		leftViewHandler.Set("Shift+L", wrapQuorumKeyFn(focusRightView))
	} else {
		leftViewHandler.Set("L", wrapQuorumKeyFn(focusRightView))
	}
	adminPageLeft.SetInputCapture(leftViewHandler.Capture)

	rightViewHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		rightViewHandler.Set("Shift+H", wrapQuorumKeyFn(focusLeftView))
	} else {
		rightViewHandler.Set("H", wrapQuorumKeyFn(focusLeftView))
	}
	adminPageRight.SetInputCapture(rightViewHandler.Capture)
}

func GroupAdminPage(groupId string) {
	lastPannel, lastView := rootPanels.GetFrontPanel()

	focusLastView := func() {
		rootPanels.ShowPanel(lastPannel)
		rootPanels.SendToFront(lastPannel)
		App.SetFocus(lastView)
	}

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusLeftView))
	pageInputHandler.Set("Esc", wrapQuorumKeyFn(focusLastView))
	adminPage.SetInputCapture(pageInputHandler.Capture)

	rootPanels.ShowPanel("admin")
	rootPanels.SendToFront("admin")
	App.SetFocus(adminPage)
}

func AdminRefreshAll(groupId string) {
	// TODO: read announced users and announced producers
}
