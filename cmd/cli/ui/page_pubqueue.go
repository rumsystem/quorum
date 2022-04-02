package ui

import (
	"fmt"
	"net"
	"runtime"
	"strconv"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/cmd/cli/config"
	"github.com/rumsystem/quorum/cmd/cli/model"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

var pubqueuePage = cview.NewFlex()
var pubqueuePageLeft = cview.NewList()      // groups
var pubqueuePageRight = cview.NewTextView() // trx

const PUBQUEUE_PAGE = "pubqueue"

var pubqueueData = model.PubqueueDataModel{
	Cache:         make(map[string][]*chain.PublishQueueItem),
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

	initPubqueuePageInputHandler()

	// short cut
	pubqueuePage.AddItem(pubqueuePageLeft, 0, 1, false)
	pubqueuePage.AddItem(pubqueuePageRight, 0, 2, false)

	rootPanels.AddPanel(PUBQUEUE_PAGE, pubqueuePage, true, false)
}

func initPubqueuePageInputHandler() {
	focusGroupListView := func() { App.SetFocus(pubqueuePageLeft) }
	focusContentView := func() { App.SetFocus(pubqueuePageRight) }

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusContentView))
	pubqueuePage.SetInputCapture(pageInputHandler.Capture)

	leftViewHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		leftViewHandler.Set("Shift+L", wrapQuorumKeyFn(focusContentView))
	} else {
		leftViewHandler.Set("L", wrapQuorumKeyFn(focusContentView))
	}
	pubqueuePageLeft.SetInputCapture(leftViewHandler.Capture)

	rightViewHandler := cbind.NewConfiguration()

	if runtime.GOOS == "windows" {
		rightViewHandler.Set("Shift+H", wrapQuorumKeyFn(focusGroupListView))
	} else {
		rightViewHandler.Set("H", wrapQuorumKeyFn(focusGroupListView))
	}
	pubqueuePageRight.SetInputCapture(rightViewHandler.Capture)

	selectNext := func() {
		trxs := pubqueueData.GetTrx()
		minBlockNum := 0
		maxBlockNum := len(trxs) - 1

		curSelection := pubqueuePageRight.GetHighlights()
		tag := minBlockNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag += 1
		}
		if tag >= minBlockNum && tag <= maxBlockNum {
			pubqueuePageRight.Highlight(strconv.Itoa(tag))
			pubqueuePageRight.ScrollToHighlight()
		}
	}
	selectLast := func() {
		trxs := pubqueueData.GetTrx()
		minBlockNum := 0
		maxBlockNum := len(trxs) - 1

		curSelection := pubqueuePageRight.GetHighlights()
		tag := minBlockNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag -= 1
		}
		if tag >= minBlockNum && tag <= maxBlockNum {
			pubqueuePageRight.Highlight(strconv.Itoa(tag))
			pubqueuePageRight.ScrollToHighlight()
		}
	}
	showTrxInfo := func() {
	}
	pubqueuePageRight.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			clearPubqueueSelection()
		case tcell.KeyEnter:
			showTrxInfo()
		case tcell.KeyTab:
			selectNext()
		case tcell.KeyBacktab:
			selectLast()
		default:
		}
	})
}

func Pubqueue() {
	rootPanels.ShowPanel(PUBQUEUE_PAGE)
	rootPanels.SendToFront(PUBQUEUE_PAGE)
	App.SetFocus(pubqueuePageRight)

	PubqueueRefreshAll()
	pubqueueData.StartTicker(PubqueueRefreshAll)
}

func PubqueueExit() {
	pubqueueData.StopTicker()
}

func PubqueueRefreshAll() {
	go func() {
		goPubqueueGroups()
		goPubqueueTrx()
	}()
}

func goPubqueueGroups() {
	groupsInfo, err := api.Groups()
	checkFatalError(err)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return
		}
		Error("Failed to get groups", err.Error())
	} else {
		oldGroups := pubqueueData.GetGroups().GroupInfos
		pubqueueData.SetGroups(*groupsInfo)
		if len(groupsInfo.GroupInfos) != len(oldGroups) {
			drawPubqueueGroups()
		}
		curGroup := pubqueueData.GetCurrentGroup()
		if curGroup == "" && len(groupsInfo.GroupInfos) > 0 {
			pubqueueData.SetCurrentGroup(groupsInfo.GroupInfos[0].GroupId)
		}
	}
}

func goPubqueueTrx() {
	curGroup := pubqueueData.GetCurrentGroup()
	if curGroup != "" {
		// get all trx in current queue
		trxs, err := api.GetPubQueue(curGroup, "", "")
		checkFatalError(err)
		pubqueueData.SetTrxs(trxs.Data)
		drawPubqueueContent()
	}
}

func drawPubqueueGroups() {
	pubqueuePageLeft.Clear()
	for i, group := range pubqueueData.GetGroups().GroupInfos {
		item := cview.NewListItem(fmt.Sprintf("%s(%s)", group.GroupName, group.GroupStatus))
		item.SetShortcut(rune('a' + i))
		pubqueuePageLeft.AddItem(item)
	}

	pubqueuePageLeft.SetSelectedFunc(func(idx int, group *cview.ListItem) {
		targetGroup := pubqueueData.GetGroups().GroupInfos[idx]
		trxs, ok := pubqueueData.GetCache(targetGroup.GroupId)
		if !ok {
			trxs = []*chain.PublishQueueItem{}
		}
		curGroup := pubqueueData.GetCurrentGroup()
		curTrxs := pubqueueData.GetTrx()
		pubqueueData.UpdateCache(curGroup, curTrxs)
		if curGroup != targetGroup.GroupId {
			// switch to new group
			clearPubqueueSelection()
			pubqueueData.SetCurrentGroup(targetGroup.GroupId)
			pubqueueData.SetTrxs(trxs)

			drawPubqueueContent()
			if len(trxs) == 0 {
				go goPubqueueTrx()
			}
			App.SetFocus(pubqueuePageRight)
		}
	})
	App.Draw()
}

func drawPubqueueContent() {
	pubqueuePageRight.Clear()
	cipherKey := ""
	groupType := "PUBLIC"

	for _, group := range pubqueueData.GetGroups().GroupInfos {
		if group.GroupId == pubqueueData.GetCurrentGroup() {
			cipherKey = group.CipherKey
			groupType = group.EncryptionType
			fmt.Fprintf(pubqueuePageRight, "Name:   %s\n", group.GroupName)
			fmt.Fprintf(pubqueuePageRight, "ID:     %s\n", group.GroupId)
			fmt.Fprintf(pubqueuePageRight, "Owner:  %s\n", group.OwnerPubKey)
			fmt.Fprintf(pubqueuePageRight, "HighestHeight: %d\n", group.HighestHeight)
			fmt.Fprintf(pubqueuePageRight, "Status: %s\n", group.GroupStatus)
			fmt.Fprintf(pubqueuePageRight, "\n")
			fmt.Fprintf(pubqueuePageRight, "Last Update:  %s\n", time.Unix(0, group.LastUpdated))
			fmt.Fprintf(pubqueuePageRight, "Highest Block: %s\n", group.HighestBlockId)
			break
		}
	}
	fmt.Fprintf(pubqueuePageRight, "\n\n")

	// trx
	trxs := pubqueueData.GetTrx()
	for i, trx := range trxs {
		fmt.Fprintf(pubqueuePageRight, "[\"%d\"][::b]%s[-:-:-]\n", i, trx.Trx.TrxId)
		fmt.Fprintf(pubqueuePageRight, "Updated: %s\n", time.Unix(0, trx.UpdateAt))
		fmt.Fprintf(pubqueuePageRight, "State: %s\n", trx.State)
		fmt.Fprintf(pubqueuePageRight, "RetryCount: %d\n", trx.RetryCount)

		fmt.Fprintf(pubqueuePageRight, "\t- trx %s\n", trx.Trx.TrxId)
		trxData, err := decodeTrxData(groupType, cipherKey, trx.Trx.Data)
		if err != nil {
			fmt.Fprintf(pubqueuePageRight, "\t[red:]Failed to decode: %s[-:-:-]\n", err.Error())
		} else {
			trxContent, typeUrl, err := quorumpb.BytesToMessage(trx.Trx.TrxId, trxData)
			if err != nil {
				config.Logger.Error(err)
				fmt.Fprintf(pubqueuePageRight, "\t[red:]Failed to decode: %s[-:-:-]\n", err.Error())
			} else {
				fmt.Fprintf(pubqueuePageRight, "\t\t - TypeUrl: %s\n", typeUrl)
				contentStr := fmt.Sprintf("%s", trxContent)
				if len(contentStr) > 1024 {
					contentStr = contentStr[0:1024] + "..."
				}
				fmt.Fprintf(pubqueuePageRight, "\t\t - Content: %s\n", contentStr)
			}
		}

		fmt.Fprintf(pubqueuePageRight, "\n\n")
	}
	App.Draw()
}

func clearPubqueueSelection() {
	pubqueuePageRight.Highlight("")
	cmdInput.SetLabel("")
	cmdInput.SetText("")
}
