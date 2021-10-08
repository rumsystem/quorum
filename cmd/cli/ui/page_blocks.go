// blocks display in block mode, lower level than other modes
package ui

import (
	"encoding/hex"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/cmd/cli/config"
	"github.com/rumsystem/quorum/cmd/cli/model"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

var blocksPage = cview.NewFlex()
var blocksPageLeft = cview.NewList()      // groups
var blocksPageRight = cview.NewTextView() // blocks

var blocksData = model.BlocksDataModel{
	NextBlocks:    make(map[string][]string),
	Pager:         make(map[string]model.BlockRangeOpt),
	TickerRunning: false,
	Counter:       0,
	Cache:         make(map[string][]api.BlockStruct),
}

func Blocks() {
	rootPanels.ShowPanel("blocks")
	rootPanels.SendToFront("blocks")
	App.SetFocus(blocksPageRight)

	BlocksRefreshAll()

	blocksData.StartTicker(BlocksRefreshAll)
}

func BlocksExit() {
	blocksData.StopTicker()
}

func blocksPageInit() {
	blocksPageLeft.SetTitle("Groups")
	blocksPageLeft.SetBorder(true)

	blocksPageRight.SetBorder(true)
	blocksPageRight.SetRegions(true)
	blocksPageRight.SetDynamicColors(true)
	blocksPageRight.SetPadding(0, 1, 1, 1)

	initBlocksPageInputHandler()

	// short cut
	blocksPage.AddItem(blocksPageLeft, 0, 1, false)
	blocksPage.AddItem(blocksPageRight, 0, 2, false)

	rootPanels.AddPanel("blocks", blocksPage, true, false)
}

func initBlocksPageInputHandler() {
	focusGroupListView := func() { App.SetFocus(blocksPageLeft) }
	focusContentView := func() { App.SetFocus(blocksPageRight) }

	pageInputHandler := cbind.NewConfiguration()
	pageInputHandler.Set("Enter", wrapQuorumKeyFn(focusContentView))
	blocksPage.SetInputCapture(pageInputHandler.Capture)

	leftViewHandler := cbind.NewConfiguration()
	if runtime.GOOS == "windows" {
		// windows will set extra shift mod somehow
		leftViewHandler.Set("Shift+L", wrapQuorumKeyFn(focusContentView))
	} else {
		leftViewHandler.Set("L", wrapQuorumKeyFn(focusContentView))
	}
	blocksPageLeft.SetInputCapture(leftViewHandler.Capture)

	rightViewHandler := cbind.NewConfiguration()
	navNextPage := func() {
		curGroup := blocksData.GetCurrentGroup()
		if curGroup == "" {
			Error("No Group in selection", "Please select a group first.")
			return
		}
		blocks := blocksData.GetBlocks()
		blocksLen := len(blocks)
		if blocksLen == 0 {
			return
		}
		curOpt := blocksData.GetPager(curGroup)
		blocksData.SetPager(curGroup, model.BlockRangeOpt{Start: curOpt.End, End: curOpt.End + 20})
		go goBlocksContent()
		clearBlocksSelection()
		blocksPageRight.ScrollToBeginning()
	}
	navPreviousPage := func() {
		curGroup := blocksData.GetCurrentGroup()
		if curGroup == "" {
			Error("No Group in selection", "Please select a group first.")
			return
		}
		blocks := blocksData.GetBlocks()
		blocksLen := len(blocks)
		if blocksLen == 0 {
			return
		}
		curOpt := blocksData.GetPager(curGroup)
		start := curOpt.Start - 20
		if start < 0 {
			start = 0
		}
		end := curOpt.Start
		if start < end {
			blocksData.SetPager(curGroup, model.BlockRangeOpt{Start: start, End: end})
			go goBlocksContent()
			clearBlocksSelection()
			blocksPageRight.ScrollToBeginning()
		}
	}

	if runtime.GOOS == "windows" {
		rightViewHandler.Set("Shift+H", wrapQuorumKeyFn(focusGroupListView))
		rightViewHandler.Set("Shift+N", wrapQuorumKeyFn(navNextPage))
		rightViewHandler.Set("Shift+P", wrapQuorumKeyFn(navPreviousPage))
	} else {
		rightViewHandler.Set("H", wrapQuorumKeyFn(focusGroupListView))
		// N / P to navigate
		rightViewHandler.Set("N", wrapQuorumKeyFn(navNextPage))
		rightViewHandler.Set("P", wrapQuorumKeyFn(navPreviousPage))
	}
	blocksPageRight.SetInputCapture(rightViewHandler.Capture)

	selectNextBlock := func() {
		blocks := blocksData.GetBlocks()
		minBlockNum := 1
		maxBlockNum := 1
		if len(blocks) > 0 {
			minBlockNum = int(blocks[0].BlockNum)
			maxBlockNum = int(blocks[len(blocks)-1].BlockNum)
		}
		curSelection := blocksPageRight.GetHighlights()
		tag := minBlockNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag += 1
		}
		if tag >= minBlockNum && tag <= maxBlockNum {
			blocksPageRight.Highlight(strconv.Itoa(tag))
			blocksPageRight.ScrollToHighlight()
		}
	}
	selectLastBlock := func() {
		blocks := blocksData.GetBlocks()
		minBlockNum := 1
		maxBlockNum := 1
		if len(blocks) > 0 {
			minBlockNum = int(blocks[0].BlockNum)
			maxBlockNum = int(blocks[len(blocks)-1].BlockNum)
		}
		curSelection := blocksPageRight.GetHighlights()
		tag := minBlockNum
		if len(curSelection) > 0 {
			tag, _ = strconv.Atoi(curSelection[0])
			tag -= 1
		}
		if tag >= minBlockNum && tag <= maxBlockNum {
			blocksPageRight.Highlight(strconv.Itoa(tag))
			blocksPageRight.ScrollToHighlight()
		}
	}
	showBlockTrxInfo := func() {
		curSelection := blocksPageRight.GetHighlights()
		if len(curSelection) > 0 {
			tag, _ := strconv.Atoi(curSelection[0])
			block := blocksData.GetBlockByNum(tag)
			if block != nil {
				// b := *block
				// TODO: nothing todo yet
			}
		}
	}
	blocksPageRight.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEsc:
			clearBlocksSelection()
		case tcell.KeyEnter:
			showBlockTrxInfo()
		case tcell.KeyTab:
			selectNextBlock()
		case tcell.KeyBacktab:
			selectLastBlock()
		default:
		}
	})
}

func BlocksRefreshAll() {
	go func() {
		goBlocksGroups()
		if blocksData.Counter%10 == 0 {
			// get current group info first, it will be rendered at the top
			goBlocksContent()
		}
	}()
}

func goBlocksGroups() {
	groupsInfo, err := api.Groups()
	checkFatalError(err)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return
		}
		Error("Failed to get groups", err.Error())
	} else {
		oldGroups := blocksData.GetGroups().GroupInfos
		blocksData.SetGroups(*groupsInfo)
		if len(groupsInfo.GroupInfos) != len(oldGroups) {
			drawBlocksGroups()
		}
		curGroup := blocksData.GetCurrentGroup()
		if curGroup == "" && len(groupsInfo.GroupInfos) > 0 {
			blocksData.SetCurrentGroup(groupsInfo.GroupInfos[0].GroupId)
		}
	}
}

func goBlocksContent() {
	// TODO: no block id info
	curGroup := blocksData.GetCurrentGroup()
	if curGroup != "" {
		var blocks []api.BlockStruct = []api.BlockStruct{}
		opt := blocksData.GetPager(curGroup)
		var start = opt.Start
		var end = opt.End
		for i := start; i <= end; i++ {
			curOpt := blocksData.GetPager(curGroup)
			curG := blocksData.GetCurrentGroup()
			if curG != curGroup || curOpt.Start != start || curOpt.End != end {
				config.Logger.Warnf("Abort blocks fetching due to opts change or group change\n")
				return
			}
			block, err := api.GetBlockByNum(curGroup, int(i))
			checkFatalError(err)
			if block != nil {
				blocks = append(blocks, *block)
				if block.PrevBlockId != "" {
					nextBlockLen := blocksData.SetNextBlock(block.PrevBlockId, block.BlockId)
					if nextBlockLen > 1 {
						// show message
						Error("Multiple children detected", fmt.Sprintf("Block %s has %d children", block.BlockId, nextBlockLen))
						config.Logger.Errorf("Multiple children detected: Block %s has %d children", block.BlockId, nextBlockLen)
					}
				}
			}
		}
		curOpt := blocksData.GetPager(curGroup)
		if blocksData.GetCurrentGroup() == curGroup && curOpt.Start == start && curOpt.End == end {
			// safe to update
			blocksData.SetBlocks(blocks)
			drawBlocksContent()
		}
	}
}

func drawBlocksGroups() {
	blocksPageLeft.Clear()
	for i, group := range blocksData.GetGroups().GroupInfos {
		item := cview.NewListItem(fmt.Sprintf("%s(%s)", group.GroupName, group.GroupStatus))
		item.SetShortcut(rune('a' + i))
		blocksPageLeft.AddItem(item)
	}

	blocksPageLeft.SetSelectedFunc(func(idx int, group *cview.ListItem) {
		targetGroup := blocksData.GetGroups().GroupInfos[idx]
		blocks, ok := blocksData.GetCache(targetGroup.GroupId)
		if !ok {
			blocks = []api.BlockStruct{}
		}
		// cache current contents
		curGroup := blocksData.GetCurrentGroup()
		curBlocks := blocksData.GetBlocks()
		blocksData.UpdateCache(curGroup, curBlocks)
		if curGroup != targetGroup.GroupId {
			// switch to new group
			clearBlocksSelection()
			blocksData.SetCurrentGroup(targetGroup.GroupId)
			blocksData.SetBlocks(blocks)

			drawBlocksContent()
			if len(blocks) == 0 {
				go goBlocksContent()
			}
			App.SetFocus(blocksPageRight)
		}
	})
	App.Draw()
}

func drawBlocksContent() {
	blocksPageRight.Clear()
	for _, group := range blocksData.GetGroups().GroupInfos {
		if group.GroupId == blocksData.GetCurrentGroup() {
			fmt.Fprintf(blocksPageRight, "Name:   %s\n", group.GroupName)
			fmt.Fprintf(blocksPageRight, "ID:     %s\n", group.GroupId)
			fmt.Fprintf(blocksPageRight, "Owner:  %s\n", group.OwnerPubKey)
			fmt.Fprintf(blocksPageRight, "HighestHeight: %d\n", group.HighestHeight)
			fmt.Fprintf(blocksPageRight, "Status: %s\n", group.GroupStatus)
			fmt.Fprintf(blocksPageRight, "\n")
			fmt.Fprintf(blocksPageRight, "Last Update:  %s\n", time.Unix(0, group.LastUpdated))
			fmt.Fprintf(blocksPageRight, "Highest Block: %s\n", group.HighestBlockId)
			break
		}
	}
	fmt.Fprintf(blocksPageRight, "\n\n")

	blocks := blocksData.GetBlocks()
	for _, block := range blocks {
		fmt.Fprintf(blocksPageRight, "[\"%d\"][::b]%d. %s[-:-:-]\n", block.BlockNum, block.BlockNum, block.BlockId)
		fmt.Fprintf(blocksPageRight, "%s\n", time.Unix(0, block.Timestamp))
		fmt.Fprintf(blocksPageRight, "Hash: %s\n", hex.EncodeToString(block.Hash))
		fmt.Fprintf(blocksPageRight, "Signature: %s\n", hex.EncodeToString(block.Signature))
		fmt.Fprintf(blocksPageRight, "Trxs:\n")
		for _, trx := range block.Trxs {
			fmt.Fprintf(blocksPageRight, "\t- trx %s\n", trx.TrxId)

			blockContent, typeUrl, err := quorumpb.BytesToMessage(trx.TrxId, trx.Data)
			if err != nil {
				config.Logger.Error(err)
				fmt.Fprintf(blocksPageRight, "\t[red:]Failed to decode: %s[-:-:-]\n", err.Error())
			} else {
				fmt.Fprintf(blocksPageRight, "\t\t - TypeUrl: %s\n", typeUrl)
				contentStr := fmt.Sprintf("%s", blockContent)
				if len(contentStr) > 1024 {
					contentStr = contentStr[0:1024] + "..."
				}
				fmt.Fprintf(blocksPageRight, "\t\t - Content: %s\n", contentStr)
			}
		}

		fmt.Fprintf(blocksPageRight, "Children:\n")
		for _, child := range blocksData.GetNextBlocks(block.BlockId) {
			fmt.Fprintf(blocksPageRight, "\t- blk %s\n", child)
		}
		fmt.Fprintf(blocksPageRight, "\n\n")
	}
	App.Draw()
}

func clearBlocksSelection() {
	blocksPageRight.Highlight("")
	cmdInput.SetLabel("")
	cmdInput.SetText("")
}

func jumpToBlock(num int) {
	curGroup := blocksData.GetCurrentGroup()
	if curGroup == "" {
		Error("No Group in selection", "Please select a group first.")
		return
	}
	curOpt := blocksData.GetPager(curGroup)
	if curOpt.Start == uint64(num) {
		return
	}
	start := uint64(num)
	end := uint64(num) + 20

	blocksData.SetPager(curGroup, model.BlockRangeOpt{Start: start, End: end})
	go goBlocksContent()
	clearBlocksSelection()
	blocksPageRight.ScrollToBeginning()
}

func BlockCMDJump(cmd string) {
	numStr := strings.Replace(cmd, CMD_BLOCKS_JMP, "", -1)
	numStr = strings.TrimSpace(numStr)
	num, err := strconv.Atoi(numStr)
	if err == nil {
		jumpToBlock(num)
	}
}
