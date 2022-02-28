package ui

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

var adminPageHelpText = strings.TrimSpace(
	"Shortcuts:\n" +
		"Enter to select left pannel.\n" +
		"Esc to go back.\n" +
		"Shift + h/j/k/l to naviagate between pannels.\n" +
		"Tab / Shift + Tab to scroll.\n" +
		"Enter to open operation modal of selected items.\n" +
		"Press `r` to refresh data.\n",
)

var adminPage = cview.NewFlex()
var adminPageAnnouncedUsersView = cview.NewTextView()     // announced users
var adminPageAnnouncedProducersView = cview.NewTextView() // announced producers
var adminAnnouncedApproveModal = cview.NewModal()         // announced users/producers approval modal
var adminGroupConfigView = cview.NewTable()               // Config List

func adminPageInit() {
	adminPageAnnouncedUsersView.SetTitle("Announced Users")
	adminPageAnnouncedUsersView.SetBorder(true)
	adminPageAnnouncedUsersView.SetRegions(true)
	adminPageAnnouncedUsersView.SetDynamicColors(true)

	adminPageAnnouncedProducersView.SetTitle("Announced Producers")
	adminPageAnnouncedProducersView.SetBorder(true)
	adminPageAnnouncedProducersView.SetRegions(true)
	adminPageAnnouncedProducersView.SetDynamicColors(true)

	adminGroupConfigView.SetBorders(true)

	adminAnnouncedApproveModal.SetBackgroundColor(tcell.ColorBlack)
	adminAnnouncedApproveModal.SetButtonBackgroundColor(tcell.ColorWhite)
	adminAnnouncedApproveModal.SetButtonTextColor(tcell.ColorBlack)
	adminAnnouncedApproveModal.SetTextColor(tcell.ColorWhite)
	adminAnnouncedApproveModal.SetTitle("Operation")
	adminAnnouncedApproveModal.SetText("Approve selected user?")
	adminAnnouncedApproveModal.AddButtons([]string{"Approve", "Deny", "Close"})

	rootPanels.AddPanel("adminAnnouncedApproveModal", adminAnnouncedApproveModal, false, false)
	rootPanels.HidePanel("adminAnnouncedApproveModal")

	initAdminPageInputHandler()

	leftFlex := cview.NewFlex()
	leftFlex.SetDirection(cview.FlexRow)
	leftFlex.AddItem(adminPageAnnouncedUsersView, 0, 1, false)
	leftFlex.AddItem(adminPageAnnouncedProducersView, 0, 1, false)

	rightFlex := cview.NewFlex()
	rightFlex.SetDirection(cview.FlexRow)
	rightFlex.SetBorder(true)
	rightFlex.SetTitle("Group Config List")
	rightHelpView := cview.NewTextView()
	rightHelpView.SetText(adminPageHelpText)
	rightFlex.AddItem(rightHelpView, 0, 1, false)
	rightFlex.AddItem(adminGroupConfigView, 0, 2, false)

	adminPage.AddItem(leftFlex, 0, 1, false)
	adminPage.AddItem(rightFlex, 0, 1, false)

	rootPanels.AddPanel("admin", adminPage, true, false)
}

var focusAnnouncedUsersViewView = func() { App.SetFocus(adminPageAnnouncedUsersView) }
var focusAnnouncedProducersView = func() { App.SetFocus(adminPageAnnouncedProducersView) }
var focusGroupConfigView = func() { App.SetFocus(adminGroupConfigView) }
var focusAdminRootView = func() { App.SetFocus(rootPanels) }

func initAdminPageInputHandler() {
	announcedUsersHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		announcedUsersHandler.Set("Shift+J", wrapQuorumKeyFn(focusAnnouncedProducersView))
		announcedUsersHandler.Set("Shift+L", wrapQuorumKeyFn(focusGroupConfigView))
	} else {
		announcedUsersHandler.Set("J", wrapQuorumKeyFn(focusAnnouncedProducersView))
		announcedUsersHandler.Set("L", wrapQuorumKeyFn(focusGroupConfigView))
	}
	announcedUsersHandler.Set("Esc", wrapQuorumKeyFn(focusAdminRootView))
	adminPageAnnouncedUsersView.SetInputCapture(announcedUsersHandler.Capture)

	announcedProducersHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		announcedProducersHandler.Set("Shift+K", wrapQuorumKeyFn(focusAnnouncedUsersViewView))
		announcedProducersHandler.Set("Shift+L", wrapQuorumKeyFn(focusGroupConfigView))
	} else {
		announcedProducersHandler.Set("K", wrapQuorumKeyFn(focusAnnouncedUsersViewView))
		announcedProducersHandler.Set("L", wrapQuorumKeyFn(focusGroupConfigView))
	}
	announcedProducersHandler.Set("Esc", wrapQuorumKeyFn(focusAdminRootView))
	adminPageAnnouncedProducersView.SetInputCapture(announcedProducersHandler.Capture)

	groupConfigHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		groupConfigHandler.Set("Shift+H", wrapQuorumKeyFn(focusAnnouncedUsersViewView))
	} else {
		groupConfigHandler.Set("H", wrapQuorumKeyFn(focusAnnouncedUsersViewView))
	}
	groupConfigHandler.Set("Esc", wrapQuorumKeyFn(focusAdminRootView))
	adminGroupConfigView.SetInputCapture(groupConfigHandler.Capture)
}

func GroupAdminPage(groupId string) {
	lastPannel, lastView := rootPanels.GetFrontPanel()

	focusLastView := func() {
		rootPanels.ShowPanel(lastPannel)
		rootPanels.SendToFront(lastPannel)
		App.SetFocus(lastView)
	}

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusAnnouncedUsersViewView))
	pageInputHandler.Set("Esc", wrapQuorumKeyFn(focusLastView))
	pageInputHandler.Set("r", wrapQuorumKeyFn(func() {
		AdminRefreshAll(groupId)
	}))

	adminPage.SetInputCapture(pageInputHandler.Capture)

	rootPanels.ShowPanel("admin")
	rootPanels.SendToFront("admin")
	App.SetFocus(adminPage)

	AdminRefreshAll(groupId)
}

func AdminRefreshAll(groupId string) {
	go goGetAnnouncedUsers(groupId)
	go goGetAnnouncedProducers(groupId)
	go goGetGroupKeyList(groupId)
}

func goGetGroupKeyList(groupId string) {
	keys, err := api.GetGroupConfigList(groupId)
	checkFatalError(err)

	adminGroupConfigView.Clear()

	// init table header
	color := tcell.ColorYellow.TrueColor()
	cell := cview.NewTableCell("key")
	cell.SetTextColor(color)
	cell.SetAlign(cview.AlignCenter)
	adminGroupConfigView.SetCell(0, 0, cell)
	cell = cview.NewTableCell("type")
	cell.SetTextColor(color)
	cell.SetAlign(cview.AlignCenter)
	adminGroupConfigView.SetCell(0, 1, cell)
	cell = cview.NewTableCell("value")
	cell.SetTextColor(color)
	cell.SetAlign(cview.AlignCenter)
	adminGroupConfigView.SetCell(0, 2, cell)
	cell = cview.NewTableCell("memo")
	cell.SetTextColor(color)
	cell.SetAlign(cview.AlignCenter)
	adminGroupConfigView.SetCell(0, 3, cell)

	adminGroupConfigView.SetFixed(0, 0)
	adminGroupConfigView.SetFixed(0, 1)
	adminGroupConfigView.SetFixed(0, 2)
	adminGroupConfigView.SetFixed(0, 3)

	adminGroupConfigView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			adminGroupConfigView.SetSelectable(true, false)
		}
		if key == tcell.KeyTab {
			// select next row
			row, col := adminGroupConfigView.GetSelection()
			row += 1
			if row >= 0 && row <= len(keys) {
				adminGroupConfigView.Select(row, col)
			}
		}
		if key == tcell.KeyBacktab {
			// select last row
			row, col := adminGroupConfigView.GetSelection()
			row -= 1
			if row >= 0 && row <= len(keys) {
				adminGroupConfigView.Select(row, col)
			}
		}
	})
	adminGroupConfigView.SetSelectedFunc(func(row int, column int) {
		if row == 0 {
			// create new config
			item := &handlers.AppConfigKeyItem{}
			item.Type = "string"
			GroupConfigFormShow(groupId, item)
		} else {
			for i := 0; i < adminGroupConfigView.GetColumnCount(); i++ {
				adminGroupConfigView.GetCell(row, i).SetTextColor(tcell.ColorRed.TrueColor())
			}
			idx := row - 1
			if idx >= 0 && idx < len(keys) {
				key := keys[idx].Name
				item, _ := api.GetGroupConfig(groupId, key)
				GroupConfigFormShow(groupId, item)
			}
		}
		adminGroupConfigView.SetSelectable(false, false)
	})

	for i, each := range keys {
		row := i + 1
		k := each.Name
		color := tcell.ColorWhite.TrueColor()
		cell := cview.NewTableCell(k)
		cell.SetTextColor(color)
		cell.SetAlign(cview.AlignCenter)
		adminGroupConfigView.SetCell(row, 0, cell)
		cell = cview.NewTableCell(each.Type)
		cell.SetTextColor(color)
		cell.SetAlign(cview.AlignCenter)
		adminGroupConfigView.SetCell(row, 1, cell)
		go func() {
			detail, err := api.GetGroupConfig(groupId, k)
			if err != nil {
				return
			}
			cell := cview.NewTableCell(detail.Value)
			cell.SetTextColor(color)
			cell.SetAlign(cview.AlignCenter)
			adminGroupConfigView.SetCell(row, 2, cell)
			cell = cview.NewTableCell(detail.Memo)
			cell.SetTextColor(color)
			cell.SetAlign(cview.AlignCenter)
			adminGroupConfigView.SetCell(row, 3, cell)
		}()
	}

}

func goGetAnnouncedUsers(groupId string) {
	aUsers, err := api.AnnouncedUsers(groupId)
	checkFatalError(err)

	adminPageAnnouncedUsersView.Clear()

	for i, each := range aUsers {
		fmt.Fprintf(adminPageAnnouncedUsersView, "[\"%d\"]%s %s\n", i, each.Result, time.Unix(0, each.TimeStamp))
		fmt.Fprintf(adminPageAnnouncedUsersView, "AnnouncedSignPubkey: %s\n", each.AnnouncedSignPubkey)
		fmt.Fprintf(adminPageAnnouncedUsersView, "AnnouncedEncryptPubkey: %s\n", each.AnnouncedEncryptPubkey)
		fmt.Fprintf(adminPageAnnouncedUsersView, "AnnouncerSign: %s\n", each.AnnouncerSign)
		fmt.Fprintf(adminPageAnnouncedUsersView, "Memo: %s\n", each.Memo)
		fmt.Fprintf(adminPageAnnouncedUsersView, "\n\n")
	}

	selectNextUser := func() {
		minNum := 0
		maxNum := len(aUsers) - 1

		curSelection := adminPageAnnouncedUsersView.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag += 1
		}
		if tag >= minNum && tag <= maxNum {
			adminPageAnnouncedUsersView.Highlight(strconv.Itoa(tag))
			adminPageAnnouncedUsersView.ScrollToHighlight()
		}
	}
	selectLastUser := func() {
		minNum := 0
		maxNum := len(aUsers) - 1

		curSelection := adminPageAnnouncedUsersView.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag -= 1
		}
		if tag >= minNum && tag <= maxNum {
			adminPageAnnouncedUsersView.Highlight(strconv.Itoa(tag))
			adminPageAnnouncedUsersView.ScrollToHighlight()
		}
	}
	adminPageAnnouncedUsersView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			adminPageAnnouncedUsersView.Highlight("")
			cmdInput.SetLabel("")
			cmdInput.SetText("")
		case tcell.KeyEnter:
			curSelection := adminPageAnnouncedUsersView.GetHighlights()
			if len(curSelection) > 0 {
				idx, _ := strconv.Atoi(curSelection[0])
				user := aUsers[idx]
				adminAnnouncedApproveModal.SetText("Approve user with memo: " + user.Memo + "?")
				adminAnnouncedApproveModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonIndex {
					case 0:
						// approve
						go goApprove(groupId, user, false)
						rootPanels.HidePanel("adminAnnouncedApproveModal")
						Info("Syncing...", "Keep waiting and press `r` to refresh")
					case 1:
						// delete
						go goApprove(groupId, user, true)
						rootPanels.HidePanel("adminAnnouncedApproveModal")
						Info("Syncing...", "Keep waiting and press `r` to refresh")
					case 2:
						// abort operation
						rootPanels.HidePanel("adminAnnouncedApproveModal")
					}
				})
				rootPanels.ShowPanel("adminAnnouncedApproveModal")
				rootPanels.SendToFront("adminAnnouncedApproveModal")
				App.SetFocus(adminAnnouncedApproveModal)
				App.Draw()
			}

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

	adminPageAnnouncedProducersView.Clear()

	for i, each := range aProducers {
		fmt.Fprintf(adminPageAnnouncedProducersView, "[\"%d\"]%s %s\n", i, each.Result, time.Unix(0, each.TimeStamp))
		fmt.Fprintf(adminPageAnnouncedProducersView, "AnnouncedPubkey: %s\n", each.AnnouncedPubkey)
		fmt.Fprintf(adminPageAnnouncedProducersView, "AnnouncerSign: %s\n", each.AnnouncerSign)
		fmt.Fprintf(adminPageAnnouncedProducersView, "Memo: %s\n", each.Memo)
		fmt.Fprintf(adminPageAnnouncedProducersView, "\n\n")
	}

	selectNextUser := func() {
		minNum := 0
		maxNum := len(aProducers) - 1

		curSelection := adminPageAnnouncedProducersView.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag += 1
		}
		if tag >= minNum && tag <= maxNum {
			adminPageAnnouncedProducersView.Highlight(strconv.Itoa(tag))
			adminPageAnnouncedProducersView.ScrollToHighlight()
		}
	}
	selectLastUser := func() {
		minNum := 0
		maxNum := len(aProducers) - 1

		curSelection := adminPageAnnouncedProducersView.GetHighlights()
		tag := minNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag -= 1
		}
		if tag >= minNum && tag <= maxNum {
			adminPageAnnouncedProducersView.Highlight(strconv.Itoa(tag))
			adminPageAnnouncedProducersView.ScrollToHighlight()
		}
	}
	adminPageAnnouncedProducersView.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			adminPageAnnouncedProducersView.Highlight("")
			cmdInput.SetLabel("")
			cmdInput.SetText("")
		case tcell.KeyEnter:
			curSelection := adminPageAnnouncedProducersView.GetHighlights()
			if len(curSelection) > 0 {
				idx, _ := strconv.Atoi(curSelection[0])
				user := aProducers[idx]
				adminAnnouncedApproveModal.SetText("Approve producer with memo: " + user.Memo + "?")
				adminAnnouncedApproveModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonIndex {
					case 0:
						// approve
						go goApproveProducer(groupId, user, false)
						rootPanels.HidePanel("adminAnnouncedApproveModal")
						Info("Syncing...", "Keep waiting and press `r` to refresh")
					case 1:
						// delete
						go goApproveProducer(groupId, user, true)
						rootPanels.HidePanel("adminAnnouncedApproveModal")
						Info("Syncing...", "Keep waiting and press `r` to refresh")
					case 2:
						// abort operation
						rootPanels.HidePanel("adminAnnouncedApproveModal")
					}
				})
				rootPanels.ShowPanel("adminAnnouncedApproveModal")
				rootPanels.SendToFront("adminAnnouncedApproveModal")
				App.SetFocus(adminAnnouncedApproveModal)
				App.Draw()
			}

		case tcell.KeyTab:
			selectNextUser()
		case tcell.KeyBacktab:
			selectLastUser()
		default:
		}
	})

}

func goApprove(groupId string, user *handlers.AnnouncedUserListItem, removal bool) {
	_, err := api.ApproveAnnouncedUser(groupId, user, removal)
	checkFatalError(err)
	if err != nil {
		Error("Failed to call user API: ", err.Error())
	}
}

func goApproveProducer(groupId string, user *handlers.AnnouncedProducerListItem, removal bool) {
	_, err := api.ApproveAnnouncedProducer(groupId, user, removal)
	checkFatalError(err)
	if err != nil {
		Error("Failed to call user API: ", err.Error())
	}
}
