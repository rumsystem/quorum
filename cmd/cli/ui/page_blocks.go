// blocks display in block mode, lower level than other modes
package ui

import (
	"encoding/hex"
	"errors"
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
	qCrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
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

func decodeTrxData(gType, key string, data []byte) ([]byte, error) {
	if gType == "PUBLIC" {
		k, err := hex.DecodeString(key)
		if err != nil {
			return nil, err
		}
		return qCrypto.AesDecode(data, k)
	} else if gType == "PRIVATE" {
		// FIXME: need to know UserEncryptPubkey
		// have to save it when join/create the private group ?
		ks := qCrypto.GetKeystore()
		return ks.Decrypt(key, data)
	}
	return nil, errors.New(fmt.Sprintf("Unknown type: %s", gType))
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
		if curOpt.NextBlockId != "" {
			blocksData.SetPager(curGroup,
				model.BlockRangeOpt{
					CurBlockId:  curOpt.NextBlockId,
					NextBlockId: "",
					Count:       curOpt.Count,
					Done:        false,
				})
			//clearBlocksSelection()
			//blocksPageRight.ScrollToBeginning()
		}

	}

	if runtime.GOOS == "windows" {
		rightViewHandler.Set("Shift+H", wrapQuorumKeyFn(focusGroupListView))
		rightViewHandler.Set("Shift+N", wrapQuorumKeyFn(navNextPage))
	} else {
		rightViewHandler.Set("H", wrapQuorumKeyFn(focusGroupListView))
		// N / P to navigate
		rightViewHandler.Set("N", wrapQuorumKeyFn(navNextPage))
	}
	blocksPageRight.SetInputCapture(rightViewHandler.Capture)

	selectNextBlock := func() {
		blocks := blocksData.GetBlocks()
		minBlockNum := 0
		maxBlockNum := len(blocks) - 1

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
		minBlockNum := 0
		maxBlockNum := len(blocks) - 1

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
		config.Logger.Info(curSelection)
		if len(curSelection) > 0 {
			tag, _ := strconv.Atoi(curSelection[0])
			block := blocksData.GetBlockByIndex(tag)
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
		goBlocksContent()
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
	curGroup := blocksData.GetCurrentGroup()
	if curGroup != "" {
		var blocks []api.BlockStruct = []api.BlockStruct{}
		opt := blocksData.GetPager(curGroup)
		if opt.Done {
			return
		}
		if opt.CurBlockId == "" {
			// Get the latest block id first
			for _, group := range blocksData.GetGroups().GroupInfos {
				if group.GroupId == curGroup {
					if len(group.HighestBlockId) > 0 {
						opt.CurBlockId = group.HighestBlockId
						blocksData.SetPager(curGroup, model.BlockRangeOpt{CurBlockId: opt.CurBlockId, NextBlockId: opt.NextBlockId, Count: opt.Count})
					}
				}
			}
		}
		if opt.CurBlockId == "" {
			Error("Failed to get blocks", "Can not get HighestBlockId of this group")
			return
		}
		var startBlockId = opt.CurBlockId
		var curBlockId = opt.CurBlockId
		var count = opt.Count
		for i := 0; i < count; i++ {
			curOpt := blocksData.GetPager(curGroup)
			curG := blocksData.GetCurrentGroup()
			if curG != curGroup || curOpt.CurBlockId != startBlockId || curOpt.Count != count {
				config.Logger.Warnf("Abort blocks fetching due to opts change or group change\n")
				return
			}
			block, err := api.GetBlockById(curGroup, curBlockId)
			checkFatalError(err)
			if block == nil {
				config.Logger.Infof("Abort blocks fetching, nil block found\n")
				break
			}
			if block.PrevBlockId != "" {
				blocks = append(blocks, *block)
				nextBlockLen := blocksData.SetNextBlock(block.PrevBlockId, block.BlockId)
				if nextBlockLen > 1 {
					// show message
					Error("Multiple children detected", fmt.Sprintf("Block %s has %d children", block.BlockId, nextBlockLen))
					config.Logger.Errorf("Multiple children detected: Block %s has %d children", block.BlockId, nextBlockLen)
				}
				curBlockId = block.PrevBlockId
			} else {
				config.Logger.Infof("blocks fetched, no prev block\n")
				break
			}
		}
		curOpt := blocksData.GetPager(curGroup)
		if blocksData.GetCurrentGroup() == curGroup && curOpt.CurBlockId == startBlockId && curOpt.Count == count {
			blocksData.SetPager(curGroup, model.BlockRangeOpt{CurBlockId: startBlockId, NextBlockId: curBlockId, Count: curOpt.Count, Done: true})
			// safe to update
			curBlocks := blocksData.GetBlocks()
			blocksData.SetBlocks(append(curBlocks, blocks...))
			drawBlocksContent()
			if len(blocksPageRight.GetHighlights()) > 0 {
				blocksPageRight.ScrollToHighlight()
			} else {
				blocksPageRight.ScrollToBeginning()
			}
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
	cipherKey := ""
	groupType := "PUBLIC"
	for _, group := range blocksData.GetGroups().GroupInfos {
		if group.GroupId == blocksData.GetCurrentGroup() {
			cipherKey = group.CipherKey
			groupType = group.EncryptionType
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
	for i, block := range blocks {
		fmt.Fprintf(blocksPageRight, "[\"%d\"][::b]%s[-:-:-]\n", i, block.BlockId)
		ts, err := strconv.Atoi(block.TimeStamp)
		if err != nil {
			fmt.Fprintf(blocksPageRight, "%s\n", time.Unix(0, int64(ts)))
		}
		fmt.Fprintf(blocksPageRight, "Hash: %s\n", hex.EncodeToString(block.Hash))
		fmt.Fprintf(blocksPageRight, "Signature: %s\n", hex.EncodeToString(block.Signature))
		fmt.Fprintf(blocksPageRight, "Trxs:\n")
		for _, trx := range block.Trxs {
			fmt.Fprintf(blocksPageRight, "\t- trx %s\n", trx.TrxId)
			trxData, err := decodeTrxData(groupType, cipherKey, trx.Data)
			if err != nil {
				fmt.Fprintf(blocksPageRight, "\t[red:]Failed to decode: %s[-:-:-]\n", err.Error())
			} else {
				blockContent, typeUrl, err := quorumpb.BytesToMessage(trx.TrxId, trxData)
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

func jumpToBlock(id string) {
	curGroup := blocksData.GetCurrentGroup()
	if curGroup == "" {
		Error("No Group in selection", "Please select a group first.")
		return
	}
	curOpt := blocksData.GetPager(curGroup)
	if curOpt.CurBlockId == id {
		return
	}
	blocksData.SetPager(curGroup, model.BlockRangeOpt{CurBlockId: id, Count: curOpt.Count})
	go goBlocksContent()
	clearBlocksSelection()
	blocksPageRight.ScrollToBeginning()
}

func BlockCMDJump(cmd string) {
	blockId := strings.Replace(cmd, model.CommandBlocksJmp.Cmd, "", -1)
	jumpToBlock(blockId)
}
