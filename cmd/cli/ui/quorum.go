package ui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/huo-ju/quorum/cmd/cli/api"
	"github.com/huo-ju/quorum/cmd/cli/config"
	"github.com/gdamore/tcell/v2"
)

// model
type quorumDataModel struct {
	users         map[string]api.ContentStruct
	node          api.NodeInfoStruct
	network       api.NetworkInfoStruct
	groups        api.GroupInfoListStruct
	contents      []api.ContentStruct
	contentFilter string
	cache         map[string][]api.ContentStruct
	curGroup      string
	tickerCh      chan struct{}
	tickerRunning bool

	sync.RWMutex
}

func (q *quorumDataModel) GetUserName(pubkey string) string {
	q.RLock()
	defer q.RUnlock()

	users := q.users
	content, hasKey := users[pubkey]
	if hasKey {
		name, ok := content.Content["name"]
		if ok {
			return fmt.Sprintf("%v", name)
		}
	}
	return pubkey
}
func (q *quorumDataModel) UpdateUserInfo(content api.ContentStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	users := q.users
	userPubKey := content.Publisher
	old, hasKey := users[userPubKey]
	if !hasKey {
		users[userPubKey] = content
	} else {
		if content.TimeStamp > old.TimeStamp {
			users[userPubKey] = content
		}
	}
}

func (q *quorumDataModel) SetNetworkInfo(network api.NetworkInfoStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.network = network
}

func (q *quorumDataModel) GetNetworkInfo() api.NetworkInfoStruct {
	q.RLock()
	defer q.RUnlock()

	return q.network
}

func (q *quorumDataModel) SetNodeInfo(node api.NodeInfoStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.node = node
}

func (q *quorumDataModel) GetNodeInfo() api.NodeInfoStruct {
	q.RLock()
	defer q.RUnlock()

	return q.node
}

func (q *quorumDataModel) SetGroups(groups api.GroupInfoListStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.groups = groups
}

func (q *quorumDataModel) GetGroups() api.GroupInfoListStruct {
	q.RLock()
	defer q.RUnlock()

	return q.groups
}

func (q *quorumDataModel) GetContents() []api.ContentStruct {
	q.RLock()
	defer q.RUnlock()

	return q.contents
}

func (q *quorumDataModel) SetContents(contents []api.ContentStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.contents = contents
}

func (q *quorumDataModel) GetCache(gid string) ([]api.ContentStruct, bool) {
	q.RLock()
	defer q.RUnlock()
	data, ok := q.cache[gid]
	return data, ok
}

func (q *quorumDataModel) UpdateCache(gid string, contents []api.ContentStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()
	q.cache[gid] = contents
}

func (q *quorumDataModel) GetCurrentGroup() string {
	q.RLock()
	defer q.RUnlock()
	return q.curGroup
}

func (q *quorumDataModel) SetCurrentGroup(gid string) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()
	q.curGroup = gid
}

const (
	CONTENT_FILTER_ALL       string = "ALL"
	CONTENT_FILTER_FOLLOWING string = "FOLLOWING"
	CONTENT_FILTER_MINE      string = "MINE"
)

var FILTERS = map[string]func(api.ContentStruct) bool{
	CONTENT_FILTER_ALL: func(content api.ContentStruct) bool {
		return true
	},
	CONTENT_FILTER_FOLLOWING: func(content api.ContentStruct) bool {
		return contains(config.RumConfig.Quorum.Following, content.Publisher)
	},
	CONTENT_FILTER_MINE: func(content api.ContentStruct) bool {
		return quorumData.GetNodeInfo().NodePubKey == content.Publisher
	},
}

var quorumData = quorumDataModel{
	users:         make(map[string]api.ContentStruct),
	contentFilter: CONTENT_FILTER_ALL,
	tickerRunning: false,
	cache:         make(map[string][]api.ContentStruct)}

var quorumPage = cview.NewFlex()

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

// Quorum function will init the main widget
// it will start a ticker to keep refreshing
// data from api server
func Quorum(apiServer string) {
	api.SetApiServer(apiServer)
	// set loading, fetch network, show error or status

	rootPanels.ShowPanel("quorum")
	rootPanels.SendToFront("quorum")
	App.SetFocus(contentView)

	go goQuorumRefresh()

	if !quorumData.tickerRunning {
		ticker := time.NewTicker(500 * time.Millisecond)
		quorumData.tickerCh = make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					if api.IsValidApiServer() {
						goQuorumRefresh()
					}
				case <-quorumData.tickerCh:
					ticker.Stop()
					quorumData.tickerRunning = false
					return
				}
			}
		}()
	}
}

// CMD /join handler
func QuorumCmdJoinHandler(cmd string) {
	seedStrOrFile := strings.Replace(cmd, CMD_QUORUM_JOIN, "", -1)
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

func QuorumGroupNickHandler(cmd string) {
	nick := strings.Replace(cmd, CMD_QUORUM_NICK, "", -1)
	nick = strings.TrimSpace(nick)
	go goQuorumNick(nick)
}

// CMD /send handler
func QuorumCmdSendHandler(cmd string) {
	msg := strings.Replace(cmd, CMD_QUORUM_SEND, "", -1)
	msg = strings.TrimSpace(msg)
	go goQuorumCreateContent(msg)
}

// CMD /group.create handler
func QuorumNewGroupHandler(cmd string) {
	groupName := strings.Replace(cmd, CMD_QUORUM_NEW_GROUP, "", -1)
	groupName = strings.TrimSpace(groupName)
	go goQuorumCreateGroup(groupName)
}

// CMD /group.leave handler
func QuorumLeaveGroupHandler() {
	if quorumData.GetCurrentGroup() == "" {
		Error("No Group to Leave", "Please select a group first.")
		return
	}
	curGroupOwner := ""
	for _, group := range quorumData.GetGroups().Groups {
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
	for _, group := range quorumData.GetGroups().Groups {
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
		clearSelection := func() {
			// Stop highlighting
			contentView.Highlight("")
			cmdInput.SetLabel("")
			cmdInput.SetText("")
		}
		if key == tcell.KeyEsc {
			clearSelection()
			return
		}

		curSelection := contentView.GetHighlights()
		contentNum := len(quorumData.contents)

		setCmdInputLabel := func(idx int) {
			if idx >= 0 && idx < len(quorumData.contents) {
				trx := quorumData.contents[idx].TrxId
				cmdInput.SetLabel("Press \"Enter\" to fetch trx infomation of: ")
				cmdInput.SetText(trx)
			}
		}

		if key == tcell.KeyEnter {
			if len(curSelection) > 0 {
				tag, _ := strconv.Atoi(curSelection[0])
				index := contentNum - tag
				trx := quorumData.contents[index].TrxId
				cmdInput.SetLabel("Loading infomation of trx: ")
				cmdInput.SetText(trx)
				go goQuorumTrxInfo(trx)
			}
		}

		if key == tcell.KeyTab {
			if len(curSelection) == 0 {
				// no selection, go to the first one
				tag := contentNum // index in range [1, contentNum]
				contentView.Highlight(strconv.Itoa(tag))
				contentView.ScrollToHighlight()
				setCmdInputLabel(contentNum - tag)
			} else {
				// go down
				tag, _ := strconv.Atoi(curSelection[0])
				tag = (tag - 1)
				if tag >= 1 {
					contentView.Highlight(strconv.Itoa(tag))
					contentView.ScrollToHighlight()
					setCmdInputLabel(contentNum - tag)
				}
			}
		}

		if key == tcell.KeyBacktab && len(curSelection) > 0 {
			// go up
			index, _ := strconv.Atoi(curSelection[0])
			index = (index + 1)
			if index <= contentNum {
				contentView.Highlight(strconv.Itoa(index))
				contentView.ScrollToHighlight()
				setCmdInputLabel(index)
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
		contentInputHandler.Set("Shift+H", wrapQuorumKeyFn(focusNetworkStatusView))
		contentInputHandler.Set("Shift+L", wrapQuorumKeyFn(focusGroupInfoView))
		contentInputHandler.Set("Shift+F", wrapQuorumKeyFn(inputhandlerFollow))
		contentInputHandler.Set("Shift+U", wrapQuorumKeyFn(inputhandlerUnFollow))
		contentInputHandler.Set("Shift+!", wrapQuorumKeyFn(func() {
			quorumData.contentFilter = CONTENT_FILTER_ALL
			drawQuorumContent()
		}))
		contentInputHandler.Set("Shift+@", wrapQuorumKeyFn(func() {
			quorumData.contentFilter = CONTENT_FILTER_FOLLOWING
			drawQuorumContent()
		}))
		contentInputHandler.Set("Shift+#", wrapQuorumKeyFn(func() {
			quorumData.contentFilter = CONTENT_FILTER_MINE
			drawQuorumContent()
		}))
	} else {
		contentInputHandler.Set("H", wrapQuorumKeyFn(focusNetworkStatusView))
		contentInputHandler.Set("L", wrapQuorumKeyFn(focusGroupInfoView))
		contentInputHandler.Set("F", wrapQuorumKeyFn(inputhandlerFollow))
		contentInputHandler.Set("U", wrapQuorumKeyFn(inputhandlerUnFollow))
		contentInputHandler.Set("!", wrapQuorumKeyFn(func() {
			quorumData.contentFilter = CONTENT_FILTER_ALL
			drawQuorumContent()
		}))
		contentInputHandler.Set("@", wrapQuorumKeyFn(func() {
			quorumData.contentFilter = CONTENT_FILTER_FOLLOWING
			drawQuorumContent()
		}))
		contentInputHandler.Set("#", wrapQuorumKeyFn(func() {
			quorumData.contentFilter = CONTENT_FILTER_MINE
			drawQuorumContent()
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
	for i, group := range quorumData.GetGroups().Groups {
		item := cview.NewListItem(fmt.Sprintf("%s(%s)", group.GroupName, group.GroupStatus))
		item.SetShortcut(rune('a' + i))
		groupListView.AddItem(item)
	}

	groupListView.SetSelectedFunc(func(idx int, group *cview.ListItem) {
		targetGroup := quorumData.GetGroups().Groups[idx]
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
		}
	})
	drawQuorumCurrentGroup()
	App.Draw()
}

// called by drawQuorumGroups
func drawQuorumCurrentGroup() {
	if quorumData.GetCurrentGroup() == "" && len(quorumData.GetGroups().Groups) > 0 {
		// set default to the first group
		quorumData.SetCurrentGroup(quorumData.GetGroups().Groups[0].GroupId)
	}

	if quorumData.GetCurrentGroup() != "" && len(quorumData.GetGroups().Groups) > 0 {
		// draw current group info including
		// 1. contentView's title
		// 2. groupInfoView's content
		// update contentView's title
		contentTitle := "Group Content"
		for _, group := range quorumData.GetNetworkInfo().Groups {
			if group.GroupId == quorumData.GetCurrentGroup() {
				contentTitle = fmt.Sprintf("%s (%d peers connected)", group.GroupName, len(group.Peers))
				break
			}
		}
		contentView.SetTitle(contentTitle)

		// update groupInfoView
		groupInfoView.Clear()
		for _, group := range quorumData.GetGroups().Groups {
			if group.GroupId == quorumData.GetCurrentGroup() {
				fmt.Fprintf(groupInfoView, "Name:   %s\n", group.GroupName)
				fmt.Fprintf(groupInfoView, "ID:     %s\n", group.GroupId)
				fmt.Fprintf(groupInfoView, "Owner:  %s\n", group.OwnerPubKey)
				fmt.Fprintf(groupInfoView, "Blocks: %d\n", group.LatestBlockNum)
				fmt.Fprintf(groupInfoView, "Status: %s\n", group.GroupStatus)
				fmt.Fprintf(groupInfoView, "\n")
				fmt.Fprintf(groupInfoView, "Last Update:  %s\n", time.Unix(0, group.LastUpdate))
				fmt.Fprintf(groupInfoView, "Latest Block: %s\n", group.LatestBlockId)
				break
			}
		}
	}
	App.Draw()
}

func drawQuorumContent() {
	contentView.Clear()
	totalLen := len(quorumData.contents)

	nodePubkey := quorumData.GetNodeInfo().NodePubKey
	filterFunc := FILTERS[quorumData.contentFilter]

	tabPager := fmt.Sprintf("%s(!) %s(@) %s(#)", CONTENT_FILTER_ALL, CONTENT_FILTER_FOLLOWING, CONTENT_FILTER_MINE)

	if quorumData.contentFilter == CONTENT_FILTER_ALL {
		tabPager = fmt.Sprintf("[yellow::b][-:yellow]%s(!)[yellow:-] %s(@) %s(#)[-:-:-]", CONTENT_FILTER_ALL, CONTENT_FILTER_FOLLOWING, CONTENT_FILTER_MINE)
	} else if quorumData.contentFilter == CONTENT_FILTER_FOLLOWING {
		tabPager = fmt.Sprintf("[yellow::b]%s(!) [-:yellow]%s(@)[yellow:-] %s(#)[-:-:-]", CONTENT_FILTER_ALL, CONTENT_FILTER_FOLLOWING, CONTENT_FILTER_MINE)
	} else if quorumData.contentFilter == CONTENT_FILTER_MINE {
		tabPager = fmt.Sprintf("[yellow::b]%s(!) %s(@) [-:yellow]%s(#)[:-][-:-:-]", CONTENT_FILTER_ALL, CONTENT_FILTER_FOLLOWING, CONTENT_FILTER_MINE)
	}
	fmt.Fprintf(contentView, "%s\n\n", tabPager)
	for i, content := range quorumData.contents {
		if !filterFunc(content) {
			continue
		}
		followingStatus := ""
		if contains(config.RumConfig.Quorum.Following, content.Publisher) {
			followingStatus = "(following)"
		}
		if nodePubkey == content.Publisher {
			followingStatus = "(me)"
		}
		name := quorumData.GetUserName(content.Publisher)
		fmt.Fprintf(contentView, "[\"%d\"][::b]%s[-:-:-]\n", totalLen-i /*keep the order*/, name)
		fmt.Fprintf(contentView, "%s\n", time.Unix(0, content.TimeStamp))
		if followingStatus != "" {
			fmt.Fprintf(contentView, "%s\n\n", followingStatus)
		} else {
			fmt.Fprintf(contentView, "\n")
		}
		fmt.Fprintf(contentView, "%s\n", content.Content["content"])
		fmt.Fprintf(contentView, "\n\n")
	}
	drawQuorumCurrentGroup()
	App.Draw()
}

func drawQuorumContentInfo(trx api.TrxStruct) {
	contentInfoView.Clear()
	fmt.Fprintf(contentInfoView, "TrxId:     %s\n", trx.TrxId)
	fmt.Fprintf(contentInfoView, "GroupId:   %s\n", trx.GroupId)
	fmt.Fprintf(contentInfoView, "Sender:    %s\n", trx.Sender)
	fmt.Fprintf(contentInfoView, "Pubkey:    %s\n", trx.Pubkey)
	fmt.Fprintf(contentInfoView, "Signature: %s\n", trx.Signature)
	fmt.Fprintf(contentInfoView, "TimeStamp: %s\n", time.Unix(0, trx.TimeStamp))
	fmt.Fprintf(contentInfoView, "Version:   %s\n", trx.Version)

	cmdInput.SetLabel("")
	App.Draw()
}

// Connect to an api server
// All actions will be operated via the api server
func goQuorumNetwork() {
	networkInfo, err := api.Network()
	if err != nil {
		Error("Failed to connect api server", err.Error())
	} else {
		quorumData.SetNetworkInfo(*networkInfo)
		drawQuorumNetwork()
	}
}

func goQuorumNode() {
	nodeInfo, err := api.Node()
	if err != nil {
		Error("Failed to connect api server", err.Error())
	} else {
		quorumData.SetNodeInfo(*nodeInfo)
	}
}

func goQuorumGroups() {
	groupsInfo, err := api.Groups()
	if err != nil {
		Error("Failed to get groups", err.Error())
	} else {
		sort.Sort(groupsInfo)
		oldGroups := quorumData.GetGroups().Groups
		quorumData.SetGroups(*groupsInfo)
		if len(groupsInfo.Groups) != len(oldGroups) {
			drawQuorumGroups()
		}
		drawQuorumCurrentGroup()
	}
}

func goQuorumContent() {
	curGroup := quorumData.GetCurrentGroup()
	if curGroup != "" {
		contents, err := api.Content(curGroup)
		if err != nil {
			Error("Failed to get content", err.Error())
		} else {
			posts := []api.ContentStruct{}
			for _, c := range *contents {
				if api.IsQuorumContentMessage(c) {
					posts = append(posts, c)
				} else if api.IsQuorumContentUserInfo(c) {
					quorumData.UpdateUserInfo(c)
				}
			}
			curContents := quorumData.GetContents()
			if len(posts) > len(curContents) {
				sort.Sort(sort.Reverse(api.ContentList(posts)))
				cmdInput.SetLabel(fmt.Sprintf("[%d] New posts come: ", len(posts)-len(curContents)))
				firstContentInterface := posts[0].Content["content"]
				firstContent := fmt.Sprintf("%v", firstContentInterface)
				if len(firstContent) > 20 {
					previewContentRunes := []rune(firstContent)
					previewStr := string(previewContentRunes[0:20])
					cmdInput.SetText(previewStr + "...")
				} else {
					cmdInput.SetText(firstContent)
				}
				thisGroup := quorumData.GetCurrentGroup()
				if thisGroup == curGroup {
					quorumData.SetContents(posts)
				}
				drawQuorumContent()
			}
		}
	}
}

// Refresh all data
// update ui if new content comes in
// run in a goroutine
func goQuorumRefresh() {
	// must in order
	goQuorumNetwork()
	goQuorumNode()
	goQuorumGroups()
	goQuorumContent()
}

func goQuorumTrxInfo(trxId string) {
	trxInfo, err := api.TrxInfo(trxId)
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
			// validate the trx to check if it is succeed
			go func() {
				select {
				case <-time.After(30 * time.Second):
					trxInfo, err := api.TrxInfo(ret.TrxId)
					if err != nil {
						Error("Timed Out", fmt.Sprintf(
							"Content not found in 30s.\nTRX: %s\n%s\n%s", ret.TrxId, content, err.Error()))
					} else {
						if trxInfo.TrxId != ret.TrxId {
							Error("Timed Out", fmt.Sprintf("Content not found in 30s.\nTRX: %s\n%s", ret.TrxId, content))
						}
					}
				}
			}()
			App.SetFocus(contentView)
		}
	}
}

func goQuorumCreateGroup(name string) {
	ret, err := api.CreateGroup(name)
	if err != nil {
		Error("Failed to create group", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", name))
		cmdInput.SetText("Created")
		seedInfo, _ := json.Marshal(ret)

		tmpFile, err := ioutil.TempFile(os.TempDir(), "quorum-seed-")
		if err != nil {
			Error("Cannot create temporary file to save seed", err.Error())
			return
		}
		if _, err = tmpFile.Write(seedInfo); err != nil {
			Error("Failed to write to seed file", err.Error())
			return
		}

		if err := tmpFile.Close(); err != nil {
			Error("Failed to close the seed file", err.Error())
			return
		}
		Info(fmt.Sprintf("Group %s created", name), fmt.Sprintf("Seed saved at: %s. Be sure to keep it well.", tmpFile.Name()))
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

// Join into a group, seed can be a string or file path
// should be called in a goroutine
func goQuorumJoin(seed string) {
	ret, err := api.JoinGroup(seed)
	if err != nil {
		Error("Failed to send", err.Error())
	} else {
		cmdInput.SetLabel("Success: ")
		cmdInput.SetText(fmt.Sprintf("Group %s joined", ret.GroupId))
		App.SetFocus(contentView)
	}
}

// input handlers
func inputhandlerFollow() {
	curSelection := contentView.GetHighlights()
	curContents := quorumData.GetContents()
	contentNum := len(curContents)

	if len(curSelection) > 0 {
		tag, _ := strconv.Atoi(curSelection[0])
		index := contentNum - tag
		publisher := curContents[index].Publisher

		if !contains(config.RumConfig.Quorum.Following, publisher) {
			config.RumConfig.Quorum.Following = append(config.RumConfig.Quorum.Following, publisher)
			cmdInput.SetLabel("Followed: ")
			cmdInput.SetText(publisher)
			drawQuorumContent()
		}
	}
}

func inputhandlerUnFollow() {
	curSelection := contentView.GetHighlights()
	curContents := quorumData.GetContents()
	contentNum := len(curContents)

	if len(curSelection) > 0 {
		tag, _ := strconv.Atoi(curSelection[0])
		index := contentNum - tag
		publisher := curContents[index].Publisher

		if contains(config.RumConfig.Quorum.Following, publisher) {
			newFollowing := []string{}
			for _, item := range config.RumConfig.Quorum.Following {
				if item != publisher {
					newFollowing = append(newFollowing, item)
				}
			}
			config.RumConfig.Quorum.Following = newFollowing
			cmdInput.SetLabel("UnFollowed: ")
			cmdInput.SetText(publisher)
			drawQuorumContent()
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
