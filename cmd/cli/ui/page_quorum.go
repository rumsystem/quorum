// default mode, twitter like UI

package ui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/cmd/cli/config"
	"github.com/rumsystem/quorum/cmd/cli/model"
	"github.com/rumsystem/quorum/cmd/cli/utils"
)

var quorumPage = cview.NewFlex()

var redrawControlCh = make(chan struct{})

// left
var networkStatusView = cview.NewTextView()
var groupListView = cview.NewList()

// center
var contentView = cview.NewTextView()

// right
var groupInfoView = cview.NewTextView()
var contentInfoView = cview.NewTextView()

func wrapQuorumKeyFn(f func()) func(ev *tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		if !cmdMode {
			f()
		}
		// propergate to default handler
		return ev
	}
}

const (
	CONTENT_FILTER_ALL   string = "ALL"
	CONTENT_FILTER_MUTED string = "MUTED"
	CONTENT_FILTER_MINE  string = "MINE"
)

var FILTERS = map[string]func(api.ContentStruct) bool{
	CONTENT_FILTER_ALL: func(content api.ContentStruct) bool {
		return !contains(config.RumConfig.Quorum.Muted, content.Publisher)
	},
	CONTENT_FILTER_MUTED: func(content api.ContentStruct) bool {
		return contains(config.RumConfig.Quorum.Muted, content.Publisher)
	},
	CONTENT_FILTER_MINE: func(content api.ContentStruct) bool {
		return quorumData.GetNodeInfo().NodePubKey == content.Publisher
	},
}

var quorumData = model.QuorumDataModel{
	ForceUpdate:   false,
	Pager:         make(map[string]api.PagerOpt),
	Users:         make(map[string]api.ContentStruct),
	ContentFilter: CONTENT_FILTER_ALL,
	RedrawCh:      make(chan bool),
	TickerRunning: false,
	Counter:       0,
	Cache:         make(map[string][]api.ContentStruct)}

// Quorum function will init the main widget
// it will start a ticker to keep refreshing
// data from api server
func Quorum(apiServer string) {
	api.SetApiServer(apiServer)
	// set loading, fetch network, show error or status

	rootPanels.ShowPanel("quorum")
	rootPanels.SendToFront("quorum")
	App.SetFocus(contentView)

	QuorumRefreshAll()

	quorumData.StartTicker(QuorumRefreshAll)

	if redrawControlCh == nil {
		redrawControlCh = make(chan struct{})
	}
	go debounce(time.Second, quorumData.RedrawCh, redrawControlCh, func() {
		drawQuorumContent()
		tryScrollToSelection()
		config.Logger.Info("redraw")
	})
}

func QuorumExit() {
	quorumData.StopTicker()
	if redrawControlCh != nil {
		close(redrawControlCh)
		redrawControlCh = nil
	}
}

func debounce(interval time.Duration, input chan bool, ctl chan struct{}, cb func()) {
	ok := false
	timer := time.NewTimer(interval)
	for {
		select {
		case <-ctl:
			config.Logger.Info("redraw listener exit")
			timer.Stop()
			return
		case ok = <-input:
			timer.Reset(interval)
		case <-timer.C:
			if ok {
				cb()
			}
		}
	}
}

// CMD /join handler
func QuorumCmdJoinHandler(cmd string) {
	seedStrOrFile := strings.Replace(cmd, model.CommandJoin.Cmd, "", -1)
	seedStrOrFile = strings.TrimSpace(seedStrOrFile)
	if strings.HasPrefix(seedStrOrFile, "@") {
		// read seed from file
		seedFile := seedStrOrFile[1:]
		seed, err := ioutil.ReadFile(seedFile)
		if err != nil {
			Error("Failed to read seed file", err.Error())
		}
		go goQuorumJoin(string(seed))
	} else {
		go goQuorumJoin(seedStrOrFile)
	}
}

// CMD /nick handler
func QuorumGroupNickHandler(cmd string) {
	nick := strings.Replace(cmd, model.CommandQuorumNick.Cmd, "", -1)
	nick = strings.TrimSpace(nick)
	go goQuorumNick(nick)
}

// CMD /token.apply handler
func QuorumApplyTokenHandler(cmd string) {
	go goQuorumJWT()
}

// CMD /send handler
func QuorumCmdSendHandler(cmd string) {
	msg := strings.Replace(cmd, model.CommandQuorumSend.Cmd, "", -1)
	msg = strings.TrimSpace(msg)
	go goQuorumCreateContent(msg)
}

// CMD /group.create handler
func QuorumNewGroupHandler() {
	CreateGroupForm()
}

func QuorumGetGroupSeedHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group", "Please select a group first.")
		return
	}
	go func() {
		seed, err := api.GetGroupSeed(quorumData.GetCurrentGroup())
		if err != nil {
			Error("Fetch Seed", err.Error())
			return
		}
		seedBytes, err := json.Marshal(seed)
		if err != nil {
			Error("Fetch Seed", err.Error())
			return
		}
		clipboard.WriteAll(string(seedBytes))
		tmpFile, err := SaveSeedToTmpFile(seedBytes)
		if err != nil {
			Error("Failed to cache group seed", err.Error())
			return
		}
		Info("Seed", "Seed is copied to your clipboard, if that is not working, check tmp file: "+tmpFile.Name())
	}()
}

func QuorumBackupHandler() {
	go func() {
		res, err := api.DoBackup()
		if err != nil {
			Error("Backup", err.Error())
			return
		}
		backupBytes, err := json.Marshal(res)
		if err != nil {
			Error("Backup", err.Error())
			return
		}
		tmpFile, err := SaveToTmpFile(backupBytes, "backup-")
		if err != nil {
			Error("Failed to copy backup file", err.Error())
			return
		}
		tmpFileName := tmpFile.Name()
		clipboard.WriteAll(tmpFileName)
		Info("Backup", fmt.Sprintf("Backup file is dumped to the tmp file: %s, use `quorum -restore` to restore", tmpFileName))
	}()
}

// CMD /group.admin
func QuorumGroupAdminHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group to Leave", "Please select a group first.")
		return
	}
	GroupAdminPage(quorumData.GetCurrentGroup())
}

func QuorumGroupChainConfigHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group to Leave", "Please select a group first.")
		return
	}
	ChainConfigPage(quorumData.GetCurrentGroup())
}

// CMD /group.leave handler
func QuorumLeaveGroupHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group to Leave", "Please select a group first.")
		return
	}
	curGroupOwner := ""
	for _, group := range quorumData.GetGroups().GroupInfos {
		if group.GroupId == quorumData.GetCurrentGroup() {
			curGroupOwner = group.OwnerPubKey
			break
		}
	}
	myPubKey := quorumData.GetNodeInfo().NodePubKey
	if myPubKey == curGroupOwner {
		Error("Failed to Leave", "Owner can't leave the group(but can delete).")
		return
	}
	go goQuorumLeaveGroup(quorumData.GetCurrentGroup())
}

// CMD /group.delete handler
func QuorumDelGroupHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group to Delete", "Please select a group first.")
		return
	}
	curGroupOwner := ""
	for _, group := range quorumData.GetGroups().GroupInfos {
		if group.GroupId == quorumData.GetCurrentGroup() {
			curGroupOwner = group.OwnerPubKey
			break
		}
	}
	myPubKey := quorumData.GetNodeInfo().NodePubKey
	if myPubKey != curGroupOwner {
		Error("No Permission to Delete", "Only the owner can delete this group.")
		return
	}
	go func() {
		if YesNo("Are you sure to delete current group?") {
			cmdInput.SetLabel("Deleting: ")
			cmdInput.SetText(quorumData.GetCurrentGroup())
			goQuorumDelGroup(quorumData.GetCurrentGroup())
		}
	}()
}

// CMD /group.sync handler
func QuorumForceSyncGroupHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group to Sync", "Please select a group first.")
		return
	}
	go goQuorumForceSyncGroup(quorumData.GetCurrentGroup())
}

func clearContentSelection() {
	// Stop highlighting
	contentView.Highlight("")
	cmdInput.SetLabel("")
	cmdInput.SetText("")
}

func tryScrollToSelection() {
	curSelection := contentView.GetHighlights()
	if len(curSelection) == 0 {
		return
	}
	contents := quorumData.GetContents()
	contentNum := len(contents)
	tag, _ := strconv.Atoi(curSelection[0])
	curContent := contents[contentNum-tag]
	filterFunc := FILTERS[quorumData.ContentFilter]
	if filterFunc(curContent) {
		contentView.ScrollToHighlight()
	}
}

func quorumPageInit() {
	// left
	networkStatusView.SetBorder(true)
	networkStatusView.SetTitle("Network Status")

	groupListView.SetBorder(true)
	groupListView.SetTitle("Groups")

	leftFlex := cview.NewFlex()
	leftFlex.SetDirection(cview.FlexRow)
	leftFlex.AddItem(networkStatusView, 0, 1, false)
	leftFlex.AddItem(groupListView, 0, 3, false)

	// center
	contentView.SetBorder(true)
	contentView.SetRegions(true)
	contentView.SetDynamicColors(true)
	contentView.SetPadding(0, 1, 1, 1)
	contentView.SetTitle("Content")

	contentView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc {
			clearContentSelection()
			return
		}

		curSelection := contentView.GetHighlights()
		contents := quorumData.GetContents()
		contentNum := len(contents)

		setCmdInputLabel := func(idx int) {
			if idx >= 0 && idx < contentNum {
				trx := contents[idx].TrxId
				cmdInput.SetLabel("Press \"Enter\" to fetch trx infomation of: ")
				cmdInput.SetText(trx)
			}
		}

		if key == tcell.KeyEnter {
			if len(curSelection) > 0 {
				tag, _ := strconv.Atoi(curSelection[0])
				index := contentNum - tag
				trx := contents[index].TrxId
				cmdInput.SetLabel("Loading infomation of trx: ")
				cmdInput.SetText(trx)
				go goQuorumTrxInfo(trx)
			}
		}

		filterFunc := FILTERS[quorumData.ContentFilter]
		if key == tcell.KeyTab {
			if len(curSelection) == 0 {
				// no selection, go to the first one
				tag := contentNum // index in range [1, contentNum]
				for i, content := range contents {
					if filterFunc(content) {
						tag = contentNum - i
						break
					}
				}
				contentView.Highlight(strconv.Itoa(tag))
				contentView.ScrollToHighlight()
				setCmdInputLabel(contentNum - tag)
			} else {
				// go down, tag will decrease
				tag, _ := strconv.Atoi(curSelection[0])
				if contentNum-tag < 0 {
					clearContentSelection()
					return
				}
				curContent := contents[contentNum-tag]
				if !filterFunc(curContent) {
					// tab switch, choose first one
					for i, content := range contents {
						if filterFunc(content) {
							tag = contentNum - i
							break
						}
					}
				} else {
					for idx := contentNum - tag + 1; idx < contentNum; idx++ {
						content := contents[idx]
						if filterFunc(content) {
							tag = contentNum - idx
							break
						}
					}
				}
				if tag >= 1 {
					contentView.Highlight(strconv.Itoa(tag))
					contentView.ScrollToHighlight()
					setCmdInputLabel(contentNum - tag)
				}
			}
		}

		if key == tcell.KeyBacktab && len(curSelection) > 0 {
			// go up, tag will increase
			tag, _ := strconv.Atoi(curSelection[0])
			// reverse
			for idx := contentNum - tag - 1; idx >= 0; idx-- {
				content := contents[idx]
				if filterFunc(content) {
					tag = contentNum - idx
					break
				}
			}

			if tag <= contentNum {
				contentView.Highlight(strconv.Itoa(tag))
				contentView.ScrollToHighlight()
				setCmdInputLabel(tag)
			}
		}
	})

	// right
	groupInfoView.SetBorder(true)
	groupInfoView.SetTitle("Group Info")

	contentInfoView.SetBorder(true)
	contentInfoView.SetTitle("Content Info")

	rightFlex := cview.NewFlex()
	rightFlex.SetDirection(cview.FlexRow)
	rightFlex.AddItem(groupInfoView, 0, 1, false)
	rightFlex.AddItem(contentInfoView, 0, 3, false)

	// page root
	quorumPage.AddItem(leftFlex, 0, 1, false)
	quorumPage.AddItem(contentView, 0, 2, false)
	quorumPage.AddItem(rightFlex, 0, 1, false)

	// init input handlers
	quorumInputHandlersInit()
	rootPanels.AddPanel("quorum", quorumPage, true, false)
}

func quorumInputHandlersInit() {
	focusGroupListView := func() { App.SetFocus(groupListView) }
	focusContentView := func() { App.SetFocus(contentView) }
	focusGroupInfoView := func() { App.SetFocus(groupInfoView) }
	focusContentInfoView := func() { App.SetFocus(contentInfoView) }
	focusNetworkStatusView := func() { App.SetFocus(networkStatusView) }

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusContentView))
	quorumPage.SetInputCapture(pageInputHandler.Capture)

	networkStatusInputHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		networkStatusInputHandler.Set("Shift+J", wrapQuorumKeyFn(focusGroupListView))
		networkStatusInputHandler.Set("Shift+L", wrapQuorumKeyFn(focusContentView))
	} else {
		networkStatusInputHandler.Set("J", wrapQuorumKeyFn(focusGroupListView))
		networkStatusInputHandler.Set("L", wrapQuorumKeyFn(focusContentView))
	}
	networkStatusView.SetInputCapture(networkStatusInputHandler.Capture)

	groupListInputHandler := cbind.NewConfiguration()
	// Tab to naviagate items
	groupListInputHandler.Set("Tab", wrapQuorumKeyFn(func() {
		itemIdx := groupListView.GetCurrentItemIndex()
		nextIdx := (itemIdx + 1) % groupListView.GetItemCount()
		groupListView.SetCurrentItem(nextIdx)
	}))
	groupListInputHandler.Set("Shift+Tab", wrapQuorumKeyFn(func() {
		itemIdx := groupListView.GetCurrentItemIndex()
		lastIdx := (itemIdx - 1) % groupListView.GetItemCount()
		groupListView.SetCurrentItem(lastIdx)
	}))
	if runtime.GOOS == "windows" {
		groupListInputHandler.Set("Shift+K", wrapQuorumKeyFn(focusNetworkStatusView))
		groupListInputHandler.Set("Shift+L", wrapQuorumKeyFn(focusContentView))
	} else {
		groupListInputHandler.Set("K", wrapQuorumKeyFn(focusNetworkStatusView))
		groupListInputHandler.Set("L", wrapQuorumKeyFn(focusContentView))
	}
	groupListView.SetInputCapture(groupListInputHandler.Capture)

	contentInputHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		contentInputHandler.Set("Shift+P", wrapQuorumKeyFn(inputhandlerPrevPage))
		contentInputHandler.Set("Shift+N", wrapQuorumKeyFn(inputhandlerNextPage))
		contentInputHandler.Set("Shift+H", wrapQuorumKeyFn(focusNetworkStatusView))
		contentInputHandler.Set("Shift+L", wrapQuorumKeyFn(focusGroupInfoView))
		contentInputHandler.Set("Shift+M", wrapQuorumKeyFn(inputhandlerMute))
		contentInputHandler.Set("Shift+U", wrapQuorumKeyFn(inputhandlerUnMute))
		contentInputHandler.Set("Shift+!", wrapQuorumKeyFn(func() {
			quorumData.ContentFilter = CONTENT_FILTER_ALL
			drawQuorumContent()
			tryScrollToSelection()
		}))
		contentInputHandler.Set("Shift+@", wrapQuorumKeyFn(func() {
			quorumData.ContentFilter = CONTENT_FILTER_MUTED
			drawQuorumContent()
			tryScrollToSelection()
		}))
		contentInputHandler.Set("Shift+#", wrapQuorumKeyFn(func() {
			quorumData.ContentFilter = CONTENT_FILTER_MINE
			drawQuorumContent()
			tryScrollToSelection()
		}))
	} else {
		contentInputHandler.Set("P", wrapQuorumKeyFn(inputhandlerPrevPage))
		contentInputHandler.Set("N", wrapQuorumKeyFn(inputhandlerNextPage))
		contentInputHandler.Set("H", wrapQuorumKeyFn(focusNetworkStatusView))
		contentInputHandler.Set("L", wrapQuorumKeyFn(focusGroupInfoView))
		contentInputHandler.Set("M", wrapQuorumKeyFn(inputhandlerMute))
		contentInputHandler.Set("U", wrapQuorumKeyFn(inputhandlerUnMute))
		contentInputHandler.Set("!", wrapQuorumKeyFn(func() {
			quorumData.ContentFilter = CONTENT_FILTER_ALL
			drawQuorumContent()
			tryScrollToSelection()
		}))
		contentInputHandler.Set("@", wrapQuorumKeyFn(func() {
			quorumData.ContentFilter = CONTENT_FILTER_MUTED
			drawQuorumContent()
			tryScrollToSelection()
		}))
		contentInputHandler.Set("#", wrapQuorumKeyFn(func() {
			quorumData.ContentFilter = CONTENT_FILTER_MINE
			drawQuorumContent()
			tryScrollToSelection()
		}))
	}
	contentView.SetInputCapture(contentInputHandler.Capture)

	groupInfoInputHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		groupInfoInputHandler.Set("Shift+H", wrapQuorumKeyFn(focusContentView))
		groupInfoInputHandler.Set("Shift+J", wrapQuorumKeyFn(focusContentInfoView))
	} else {
		groupInfoInputHandler.Set("H", wrapQuorumKeyFn(focusContentView))
		groupInfoInputHandler.Set("J", wrapQuorumKeyFn(focusContentInfoView))
	}
	groupInfoView.SetInputCapture(groupInfoInputHandler.Capture)

	contentInfoInputHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		contentInfoInputHandler.Set("Shift+H", wrapQuorumKeyFn(focusContentView))
		contentInfoInputHandler.Set("Shift+K", wrapQuorumKeyFn(focusGroupInfoView))
	} else {
		contentInfoInputHandler.Set("H", wrapQuorumKeyFn(focusContentView))
		contentInfoInputHandler.Set("K", wrapQuorumKeyFn(focusGroupInfoView))
	}
	contentInfoView.SetInputCapture(contentInfoInputHandler.Capture)

}

func drawQuorumNetwork() {
	networkInfo := quorumData.GetNetworkInfo()
	// update node status view
	networkStatusView.SetText("")
	fmt.Fprintf(networkStatusView, "Server:   %s\n", api.ApiServer)
	fmt.Fprintf(networkStatusView, "ID:       %s\n", networkInfo.Node.PeerId)
	fmt.Fprintf(networkStatusView, "NAT Type: %s\n", networkInfo.Node.NatType)

	nodeInfo := quorumData.GetNodeInfo()
	fmt.Fprintf(networkStatusView, "Status:   %s\n", nodeInfo.NodeStatus)
	fmt.Fprintf(networkStatusView, "Version:  %s\n", nodeInfo.NodeVersion)
	fmt.Fprintf(networkStatusView, "Peers:\n")
	keys := []string{}
	for k := range nodeInfo.Peers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(networkStatusView, "\t%s: %d\n", k, len(nodeInfo.Peers[k]))
	}

	App.Draw()
}

func drawQuorumGroups() {
	// draw groupListView's content
	groupListView.Clear()
	for i, group := range quorumData.GetGroups().GroupInfos {
		item := cview.NewListItem(fmt.Sprintf("%s(%s)", group.GroupName, group.GroupStatus))
		item.SetShortcut(rune('a' + i))
		groupListView.AddItem(item)
	}

	groupListView.SetSelectedFunc(func(idx int, group *cview.ListItem) {
		targetGroup := quorumData.GetGroups().GroupInfos[idx]
		contents, ok := quorumData.GetCache(targetGroup.GroupId)
		if !ok {
			contents = []api.ContentStruct{}
		}
		// cache current contents
		curGroup := quorumData.GetCurrentGroup()
		curContents := quorumData.GetContents()
		quorumData.UpdateCache(curGroup, curContents)
		if curGroup != targetGroup.GroupId {
			// switch to new group
			quorumData.SetCurrentGroup(targetGroup.GroupId)
			quorumData.SetContents(contents)
			// should rerender all widget
			drawQuorumCurrentGroup()
			drawQuorumContent()
			if len(contents) == 0 {
				go goQuorumContent()
			}
		}
	})
	drawQuorumCurrentGroup()
	App.Draw()
}

// called by drawQuorumGroups
func drawQuorumCurrentGroup() {
	if quorumData.GetCurrentGroup() == "" && len(quorumData.GetGroups().GroupInfos) > 0 {
		// set default to the first group
		quorumData.SetCurrentGroup(quorumData.GetGroups().GroupInfos[0].GroupId)
	}

	if quorumData.GetCurrentGroup() != "" && len(quorumData.GetGroups().GroupInfos) > 0 {
		// draw current group info including
		// 1. contentView's title
		// 2. groupInfoView's content
		// update contentView's title
		contentTitle := "Group Content"
		for _, group := range quorumData.GetNetworkInfo().Groups {
			curGroup := quorumData.GetCurrentGroup()
			if group.GroupId == curGroup {
				pageOpt := quorumData.GetPager(curGroup)
				contentTitle = fmt.Sprintf("%s (%d peers connected) page[%d]", group.GroupName, len(group.Peers), pageOpt.Page)
				break
			}
		}
		contentView.SetTitle(contentTitle)

		// update groupInfoView
		groupInfoView.Clear()
		for _, group := range quorumData.GetGroups().GroupInfos {
			if group.GroupId == quorumData.GetCurrentGroup() {
				fmt.Fprintf(groupInfoView, "Name:   %s\n", group.GroupName)
				fmt.Fprintf(groupInfoView, "ID:     %s\n", group.GroupId)
				fmt.Fprintf(groupInfoView, "Owner:  %s\n", group.OwnerPubKey)
				fmt.Fprintf(groupInfoView, "HighestHeight: %d\n", group.HighestHeight)
				fmt.Fprintf(groupInfoView, "Status: %s\n", group.GroupStatus)
				fmt.Fprintf(groupInfoView, "\n")
				fmt.Fprintf(groupInfoView, "Last Update:  %s\n", time.Unix(0, group.LastUpdated))
				fmt.Fprintf(groupInfoView, "Latest Block: %s\n", group.HighestBlockId)
				break
			}
		}
	}
	App.Draw()
}

func drawQuorumContent() {
	contentView.Clear()
	contents := quorumData.GetContents()
	totalLen := len(contents)

	nodePubkey := quorumData.GetNodeInfo().NodePubKey
	filterFunc := FILTERS[quorumData.ContentFilter]

	tabPager := fmt.Sprintf("%s(!) %s(@) %s(#)", CONTENT_FILTER_ALL, CONTENT_FILTER_MUTED, CONTENT_FILTER_MINE)

	if quorumData.ContentFilter == CONTENT_FILTER_ALL {
		tabPager = fmt.Sprintf("[yellow::b][-:yellow]%s(!)[yellow:-] %s(@) %s(#)[-:-:-]", CONTENT_FILTER_ALL, CONTENT_FILTER_MUTED, CONTENT_FILTER_MINE)
	} else if quorumData.ContentFilter == CONTENT_FILTER_MUTED {
		tabPager = fmt.Sprintf("[yellow::b]%s(!) [-:yellow]%s(@)[yellow:-] %s(#)[-:-:-]", CONTENT_FILTER_ALL, CONTENT_FILTER_MUTED, CONTENT_FILTER_MINE)
	} else if quorumData.ContentFilter == CONTENT_FILTER_MINE {
		tabPager = fmt.Sprintf("[yellow::b]%s(!) %s(@) [-:yellow]%s(#)[:-][-:-:-]", CONTENT_FILTER_ALL, CONTENT_FILTER_MUTED, CONTENT_FILTER_MINE)
	}
	fmt.Fprintf(contentView, "%s\n\n", tabPager)
	for i, content := range contents {
		if !filterFunc(content) {
			continue
		}
		userStatus := ""
		if contains(config.RumConfig.Quorum.Muted, content.Publisher) {
			userStatus = "(muted)"
		}
		if nodePubkey == content.Publisher {
			userStatus = "(me)"
		}
		name := quorumData.GetUserName(content.Publisher, quorumData.GetCurrentGroup())
		if api.IsQuorumContentMessage(content) {
			msgBlock := api.ContentInnerMsgStruct{}
			jsonStr, _ := json.Marshal(content.Content)
			json.Unmarshal(jsonStr, &msgBlock)
			if msgBlock.ReplyTo.TrxId != "" {
				fmt.Fprintf(contentView, "[\"%d\"].---[::b]%s[-:-:-]\n", totalLen-i /*keep the order*/, msgBlock.ReplyTo.TrxId)
				fmt.Fprintf(contentView, "|\n")
				fmt.Fprintf(contentView, "`--->[::b]%s[-:-:-]\n", name)
			} else {
				fmt.Fprintf(contentView, "[\"%d\"][::b]%s[-:-:-]\n", totalLen-i /*keep the order*/, name)
			}

			fmt.Fprintf(contentView, "%s\n", time.Unix(0, content.TimeStamp))
			if userStatus != "" {
				fmt.Fprintf(contentView, "%s\n\n", userStatus)
			} else {
				fmt.Fprintf(contentView, "\n")
			}
			fmt.Fprintf(contentView, "%s\n", msgBlock.Content)
			fmt.Fprintf(contentView, "\n\n")
		} else if api.IsQuorumContentUserInfo(content) {
			fmt.Fprintf(contentView, "[\"%d\"][::b]%s[-:-:-]\n", totalLen-i /*keep the order*/, name)
			fmt.Fprintf(contentView, "%s\n", time.Unix(0, content.TimeStamp))
			fmt.Fprintf(contentView, "<profile update>\n")
			fmt.Fprintf(contentView, "\n\n")
		}
	}
	drawQuorumCurrentGroup()
	App.Draw()
}

func drawQuorumContentInfo(trx api.TrxStruct) {
	contentInfoView.Clear()
	fmt.Fprintf(contentInfoView, "TrxId:     %s\n", trx.TrxId)
	fmt.Fprintf(contentInfoView, "GroupId:   %s\n", trx.GroupId)
	fmt.Fprintf(contentInfoView, "Sender:    %s\n", trx.SenderPubkey)
	fmt.Fprintf(contentInfoView, "Signature: %s\n", trx.SenderSign)
	ts, err := strconv.Atoi(trx.TimeStamp)
	if err != nil {
		fmt.Fprintf(contentInfoView, "TimeStamp: %s\n", time.Unix(0, int64(ts)))
	}
	fmt.Fprintf(contentInfoView, "Version:   %s\n", trx.Version)

	mixinUID := quorumData.GetUserMixinUID(trx.SenderPubkey, trx.GroupId)
	if mixinUID != "" {
		fmt.Fprintf(contentInfoView, "MixinUID:   %s\n", mixinUID)
	}

	cmdInput.SetLabel("")
	App.Draw()
}

// Connect to an api server
// All actions will be operated via the api server
func goQuorumNetwork() {
	networkInfo, err := api.Network()
	checkFatalError(err)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return
		}
		Error("Failed to connect api server", err.Error())
	} else {
		quorumData.SetNetworkInfo(*networkInfo)
		drawQuorumNetwork()
	}
}

func goQuorumNode() {
	nodeInfo, err := api.Node()
	checkFatalError(err)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return
		}
		Error("Failed to connect api server", err.Error())
	} else {
		quorumData.SetNodeInfo(*nodeInfo)
	}
}

func goQuorumGroups() {
	groupsInfo, err := api.Groups()
	checkFatalError(err)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return
		}
		Error("Failed to get groups", err.Error())
	} else {
		oldGroups := quorumData.GetGroups().GroupInfos
		quorumData.SetGroups(*groupsInfo)
		if len(groupsInfo.GroupInfos) != len(oldGroups) {
			drawQuorumGroups()
		}
		drawQuorumCurrentGroup()
	}
}

func goQuorumContent() {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup != "" {
		var contents *[]api.ContentStruct = &[]api.ContentStruct{}
		var err error
		opt := quorumData.GetPager(curGroup)
		contents, err = api.Content(curGroup, opt)
		if !opt.Reverse {
			sort.Sort(sort.Reverse(api.ContentList(*contents)))
		}
		checkFatalError(err)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				return
			}
			Error("Failed to get content", err.Error())
		} else {
			curContents := quorumData.GetContents()
			curContentsLen := len(curContents)
			shouldUpdate := false
			if curContentsLen == 0 && len(*contents) > 0 {
				shouldUpdate = true
			} else {
				contentsLen := len(*contents)
				if contentsLen > 0 && (*contents)[0].TimeStamp > curContents[0].TimeStamp {
					shouldUpdate = true
				}
			}

			if shouldUpdate || quorumData.ForceUpdate {
				if quorumData.ForceUpdate {
					quorumData.SetForceUpdate(false)
				}

				if len(*contents) == 0 {
					cmdInput.SetLabel(fmt.Sprintf("No more posts"))
					return
				}

				cmdInput.SetLabel(fmt.Sprintf("[%d] New posts come: ", len(*contents)))

				var firstContent = ""
				for _, c := range *contents {
					if api.IsQuorumContentMessage(c) {
						if firstContent == "" {
							firstContentInterface := c.Content["content"]
							firstContent = fmt.Sprintf("%v", firstContentInterface)
						}
					} else if api.IsQuorumContentUserInfo(c) {
						quorumData.UpdateUserInfo(c, curGroup)
					}
				}

				if len(firstContent) > 20 {
					previewContentRunes := []rune(firstContent)
					previewStr := string(previewContentRunes[0:20])
					cmdInput.SetText(previewStr + "...")
				} else {
					cmdInput.SetText(firstContent)
				}

				thisGroup := quorumData.GetCurrentGroup()
				if thisGroup == curGroup {
					quorumData.SetContents(*contents)
				}

				drawQuorumContent()
			}
		}
	}
}

// Refresh all data
// update ui if new content comes in
// run in a goroutine
func QuorumRefreshAll() {
	go goQuorumNetwork()
	go goQuorumNode()

	go func() {
		goQuorumGroups()
		if quorumData.Counter%10 == 0 {
			goQuorumContent()
		}
	}()
}

func goQuorumTrxInfo(trxId string) {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup == "" {
		Error("Failed to set nickname", "Select a group first.")
		return
	}
	trxInfo, err := api.TrxInfo(curGroup, trxId)
	if err != nil {
		Error("Failed to get trx", err.Error())
	} else {
		drawQuorumContentInfo(*trxInfo)
	}
}

func goQuorumNick(nick string) {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup == "" {
		Error("Failed to set nickname", "Select a group first.")
		return
	}
	ret, err := api.Nick(curGroup, nick)
	if err != nil {
		Error("Failed to update nickname", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Nickname TRX %s Sent: ", ret.TrxId))
		cmdInput.SetText("Wait for syncing...")
		go goAutoAckGroupTrx(curGroup)
	}
}

func goQuorumJWT() {
	ret, err := api.TokenApply()
	if err != nil {
		Error("Failed to apply jwt", err.Error())
	} else {
		if ret.Token != "" {
			cmdInput.SetLabel(fmt.Sprintf("JWT saved: "))
			cmdInput.SetText(ret.Token)
			config.RumConfig.Quorum.JWT = ret.Token
			quorumData.StartTicker(QuorumRefreshAll)
		}
	}
}

func goQuorumCreateContent(content string) {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup != "" {
		ret, err := api.CreateContent(curGroup, content)
		if err != nil {
			Error("Failed to send", err.Error())
		} else {
			cmdInput.SetLabel(fmt.Sprintf("TRX %s: ", ret.TrxId))
			cmdInput.SetText("Syncing with peers..")
			go goAutoAckGroupTrx(curGroup)
			App.SetFocus(contentView)
		}
	}
}

func goAutoAckGroupTrx(groupId string) {
	trxIds, err := utils.CheckTrx(groupId, "", "")
	if err != nil {
		Error("Failed to check trx info from pubqueue", err.Error())
		return
	}
	if len(trxIds) > 0 {
		cmdInput.SetLabel(fmt.Sprintf("[%d] TRX ACKED: ", len(trxIds)))
		info := ""
		for idx, tId := range trxIds {
			if idx >= 2 {
				info += ".."
				break
			}
			info += fmt.Sprintf("%s ", tId)
		}
		cmdInput.SetText(info)
	}
}

func goQuorumLeaveGroup(gid string) {
	_, err := api.LeaveGroup(gid)
	if err != nil {
		Error("Failed to leave group", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", gid))
		cmdInput.SetText("Left")
	}
}

func goQuorumDelGroup(gid string) {
	_, err := api.DelGroup(gid)
	if err != nil {
		Error("Failed to delete group", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", gid))
		cmdInput.SetText("Deleted")
	}
}

func goQuorumForceSyncGroup(gid string) {
	_, err := api.ForceSyncGroup(gid)
	if err != nil {
		Error("Failed to force sync group", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", gid))
		cmdInput.SetText("syncing")
	}
}

// Join into a group, seed can be a string or file path
// should be called in a goroutine
func goQuorumJoin(seed string) {
	ret, err := api.JoinGroup(seed)
	if err != nil {
		Error("Failed to join", err.Error())
	} else {
		cmdInput.SetLabel("Success: ")
		cmdInput.SetText(fmt.Sprintf("Group %s joined", ret.GroupId))
		App.SetFocus(contentView)
	}
}

// input handlers
func inputhandlerMute() {
	curSelection := contentView.GetHighlights()
	curContents := quorumData.GetContents()
	contentNum := len(curContents)

	if len(curSelection) > 0 {
		tag, _ := strconv.Atoi(curSelection[0])
		index := contentNum - tag
		publisher := curContents[index].Publisher

		if !contains(config.RumConfig.Quorum.Muted, publisher) {
			config.RumConfig.Quorum.Muted = append(config.RumConfig.Quorum.Muted, publisher)
			cmdInput.SetLabel("Followed: ")
			cmdInput.SetText(publisher)
			drawQuorumContent()
		}
	}
}

func inputhandlerUnMute() {
	curSelection := contentView.GetHighlights()
	curContents := quorumData.GetContents()
	contentNum := len(curContents)

	if len(curSelection) > 0 {
		tag, _ := strconv.Atoi(curSelection[0])
		index := contentNum - tag
		publisher := curContents[index].Publisher

		if contains(config.RumConfig.Quorum.Muted, publisher) {
			newMuted := []string{}
			for _, item := range config.RumConfig.Quorum.Muted {
				if item != publisher {
					newMuted = append(newMuted, item)
				}
			}
			config.RumConfig.Quorum.Muted = newMuted
			cmdInput.SetLabel("UnFollowed: ")
			cmdInput.SetText(publisher)
			drawQuorumContent()
		}
	}
}

func inputhandlerNextPage() {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup == "" {
		Error("No Group in selection", "Please select a group first.")
		return
	}
	curContents := quorumData.GetContents()
	curLen := len(curContents)
	if curLen == 0 {
		return
	}
	trx := curContents[curLen-1].TrxId
	curOpt := quorumData.GetPager(curGroup)
	quorumData.SetPager(curGroup, api.PagerOpt{StartTrxId: trx, Reverse: true, Page: curOpt.Page + 1})
	quorumData.SetForceUpdate(true)
	go goQuorumContent()
	clearContentSelection()
	contentView.ScrollToBeginning()
}

func inputhandlerPrevPage() {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup == "" {
		Error("No Group in selection", "Please select a group first.")
		return
	}
	curContents := quorumData.GetContents()
	curLen := len(curContents)
	if curLen == 0 {
		return
	}
	trx := curContents[0].TrxId
	curOpt := quorumData.GetPager(curGroup)
	if curOpt.Page == 0 {
		return
	}
	quorumData.SetPager(curGroup, api.PagerOpt{StartTrxId: trx, Reverse: false, Page: curOpt.Page - 1})
	quorumData.SetForceUpdate(true)
	go goQuorumContent()
	clearContentSelection()
	contentView.ScrollToBeginning()
}

// to handle fatal error, like jwt missing
// this will stop the ticker
func checkFatalError(err error) {
	if err != nil {
		if strings.Contains(err.Error(), "missing or malformed jwt") {
			quorumData.StopTicker()
		}
	}
}

func contains(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}
