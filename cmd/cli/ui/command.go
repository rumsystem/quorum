package ui

import (
	"sort"
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/config"
	"github.com/rumsystem/quorum/cmd/cli/model"
)

// cmd input at the bottom
var cmdInput = cview.NewInputField()
var cmdMode = false

func cmdInputInit() {

	getCommands := func() model.CommandList {
		name, _ := rootPanels.GetFrontPanel()
		commands := model.CommandList{}
		commands = append(commands, model.BaseCommands[:]...)
		switch name {
		case "blocks":
			commands = append(commands, model.BlocksCommands[:]...)
		case "quorum":
			commands = append(commands, model.QuorumCommands[:]...)
		case "network":
			commands = append(commands, model.NetworkCommands[:]...)
		default:
		}
		sort.Sort(commands)
		return commands
	}

	cmdInput.SetAutocompleteFunc(func(currentText string) (entries []*cview.ListItem) {
		if len(currentText) == 0 {
			return
		}
		for _, item := range getCommands() {
			word := item.Cmd
			if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
				entries = append(entries, cview.NewListItem(word))
			}
		}
		if len(entries) == 0 {
			entries = nil
		}
		return
	})

	// cmdInput handler
	cmdInput.SetDoneFunc(func(key tcell.Key) {
		reset := func(label string) {
			cmdInput.SetLabel(label)
			cmdMode = false
			App.SetFocus(rootPanels)
		}
		switch key {
		case tcell.KeyEnter:
			cmdStr := cmdInput.GetText()
			cmdStr = strings.TrimSpace(cmdStr)
			if cmdStr == "" {
				reset("")
				return
			}
			if strings.HasPrefix(cmdStr, model.CommandConnect.Cmd) {
				apiServer := strings.Replace(cmdStr, model.CommandConnect.Cmd, "", -1)
				apiServer = strings.TrimSpace(apiServer)
				reset("")
				Quorum(apiServer)
			} else if strings.HasPrefix(cmdStr, model.CommandQuorumNick.Cmd) {
				reset("")
				QuorumGroupNickHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, model.CommandTokenApply.Cmd) {
				reset("")
				QuorumApplyTokenHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, model.CommandJoin.Cmd) {
				// join group
				reset("Joining: ")
				QuorumCmdJoinHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, model.CommandQuorumSend.Cmd) {
				// send data
				reset("Loading: ")
				QuorumCmdSendHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupSync.Cmd) {
				reset("")
				QuorumForceSyncGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupCreate.Cmd) {
				// send data
				reset("Creating: ")
				QuorumNewGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupSeed.Cmd) {
				reset("")
				QuorumGetGroupSeedHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandBackup.Cmd) {
				reset("")
				QuorumBackupHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupLeave.Cmd) {
				reset("")
				QuorumLeaveGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupDelete.Cmd) {
				// send data
				reset("")
				QuorumDelGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupAdmin.Cmd) {
				reset("")
				QuorumGroupAdminHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandGroupChainConfig.Cmd) {
				reset("")
				QuorumGroupChainConfigHandler()
				return
			} else if strings.HasPrefix(cmdStr, model.CommandConfigReload.Cmd) {
				reset("")
				config.Init()
				if config.RumConfig.Quorum.Server != "" {
					Quorum(config.RumConfig.Quorum.Server)
				} else {
					Welcome()
				}
				return
			} else if strings.HasPrefix(cmdStr, model.CommandConfigSave.Cmd) {
				configFilePath := config.Save()
				reset("")
				cmdInput.SetLabel("Config saved at:")
				cmdInput.SetText(configFilePath)
				return
			} else if strings.HasPrefix(cmdStr, model.CommandModeBlocks.Cmd) {
				reset("")
				exitAll()

				Blocks()
			} else if strings.HasPrefix(cmdStr, model.CommandModePubqueue.Cmd) {
				reset("")
				exitAll()

				Pubqueue()
			} else if strings.HasPrefix(cmdStr, model.CommandModeQuorum.Cmd) {
				reset("")

				exitAll()

				if config.RumConfig.Quorum.Server != "" {
					Quorum(config.RumConfig.Quorum.Server)
				} else {
					Welcome()
				}
			} else if strings.HasPrefix(cmdStr, model.CommandModeNetwork.Cmd) {
				reset("")

				exitAll()
				NetworkPage()
			} else if strings.HasPrefix(cmdStr, model.CommandNetworkPing.Cmd) {
				reset("")
				go NetworkRefreshAll()
			} else if strings.HasPrefix(cmdStr, model.CommandBlocksJmp.Cmd) {
				reset("")
				BlockCMDJump(cmdStr)
			} else {
				reset("")
				return
			}
		case tcell.KeyEsc:
			reset("")
			return
		}
	})

	// cmd input style
	cmdInput.SetBackgroundColor(tcell.ColorWhite)
	cmdInput.SetLabelColor(tcell.ColorBlack)
	cmdInput.SetFieldBackgroundColor(tcell.ColorWhite)
	cmdInput.SetFieldTextColor(tcell.ColorBlack)
}

func cmdActivate() {
	cmdMode = true
	cmdInput.SetLabel("[::b]/CMD <..args>: [::-]")
	cmdInput.SetText("")
	App.SetFocus(cmdInput)
}

func exitAll() {
	BlocksExit()
	QuorumExit()
	NetworkPageExit()
}
