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
)

var adminPage = cview.NewFlex()
var adminPageLeft = cview.NewTextView()  // users
var adminPageRight = cview.NewTextView() // producers

func adminPageInit() {
	adminPageLeft.SetTitle("Announced Users")
	adminPageLeft.SetBorder(true)
	adminPageLeft.SetRegions(true)
	adminPageLeft.SetDynamicColors(true)

	adminPageRight.SetTitle("Announced Producers")
	adminPageRight.SetBorder(true)
	adminPageRight.SetRegions(true)
	adminPageRight.SetDynamicColors(true)

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

	AdminRefreshAll(groupId)
}

func AdminRefreshAll(groupId string) {
	go goGetAnnouncedUsers(groupId)
	go goGetAnnouncedProducers(groupId)
}

func goGetAnnouncedUsers(groupId string) {
	aUsers, err := api.AnnouncedUsers(groupId)
	checkFatalError(err)

	adminPageLeft.Clear()

	for i, each := range aUsers {
		fmt.Fprintf(adminPageLeft, "[\"%d\"]%s %s\n", i, each.Result, time.Unix(0, each.TimeStamp))
		fmt.Fprintf(adminPageLeft, "AnnouncedSignPubkey: %s\n", each.AnnouncedSignPubkey)
		fmt.Fprintf(adminPageLeft, "AnnouncedEncryptPubkey: %s\n", each.AnnouncedEncryptPubkey)
		fmt.Fprintf(adminPageLeft, "AnnouncerSign: %s\n", each.AnnouncerSign)
		fmt.Fprintf(adminPageLeft, "Memo: %s\n", each.Memo)
		fmt.Fprintf(adminPageLeft, "\n\n")
	}

	selectNextUser := func() {
		minNum := 0
		maxNum := len(aUsers) - 1

		curSelection := adminPageLeft.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag += 1
		}
		if tag >= minNum && tag <= maxNum {
			adminPageLeft.Highlight(strconv.Itoa(tag))
			adminPageLeft.ScrollToHighlight()
		}
	}
	selectLastUser := func() {
		minNum := 0
		maxNum := len(aUsers) - 1

		curSelection := adminPageLeft.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag -= 1
		}
		if tag >= minNum && tag <= maxNum {
			adminPageLeft.Highlight(strconv.Itoa(tag))
			adminPageLeft.ScrollToHighlight()
		}
	}
	adminPageLeft.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			adminPageLeft.Highlight("")
			cmdInput.SetLabel("")
			cmdInput.SetText("")
		case tcell.KeyEnter:
			// TODO: show modal approve ?
		case tcell.KeyTab:
			selectNextUser()
		case tcell.KeyBacktab:
			selectLastUser()
		default:
		}
	})
}

func goGetAnnouncedProducers(groupId string) {
	aProducers, err := api.AnnouncedProducers(groupId)
	checkFatalError(err)

	adminPageRight.Clear()

	for i, each := range aProducers {
		fmt.Fprintf(adminPageRight, "[\"%d\"]%s %s\n", i, each.Result, time.Unix(0, each.TimeStamp))
		fmt.Fprintf(adminPageRight, "AnnouncedPubkey: %s\n", each.AnnouncedPubkey)
		fmt.Fprintf(adminPageRight, "AnnouncerSign: %s\n", each.AnnouncerSign)
		fmt.Fprintf(adminPageRight, "Memo: %s\n", each.Memo)
		fmt.Fprintf(adminPageRight, "\n\n")

	}
}
