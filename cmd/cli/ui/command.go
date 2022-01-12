package ui

import (
	"sort"
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/config"
)

// cmd input at the bottom
var cmdInput = cview.NewInputField()
var cmdMode = false

const (
	CMD_QUORUM_CONNECT        string = "/connect"
	CMD_QUORUM_BACKUP         string = "/backup"
	CMD_QUORUM_JOIN           string = "/join"
	CMD_QUORUM_APPLY_TOKEN    string = "/token.apply"
	CMD_QUORUM_SYNC_GROUP     string = "/group.sync"
	CMD_QUORUM_GET_GROUP_SEED string = "/group.seed"
	CMD_QUORUM_NEW_GROUP      string = "/group.create"
	CMD_QUORUM_LEAVE_GROUP    string = "/group.leave"
	CMD_QUORUM_DEL_GROUP      string = "/group.delete"
	CMD_QUORUM_GROUP_ADMIN    string = "/group.admin"
	CMD_CONFIG_RELOAD         string = "/config.reload"
	CMD_CONFIG_SAVE           string = "/config.save"
	CMD_MODE_BLOCKS           string = "/mode.blocks"
	CMD_MODE_QUORUM           string = "/mode.quorum"
	CMD_MODE_NETWORK          string = "/mode.network"

	// mode quorum only
	CMD_QUORUM_SEND string = "/send"
	CMD_QUORUM_NICK string = "/nick"

	// mode blocks only
	CMD_BLOCKS_JMP    string = "/blocks.jmp"
	CMD_BLOCKS_GENDOT string = "/blocks.gendot"

	// mode network
	CMD_NETWORK_PING string = "/network.ping"
)

func cmdInputInit() {
	baseCommands := []string{CMD_QUORUM_CONNECT, CMD_QUORUM_BACKUP, CMD_QUORUM_JOIN, CMD_QUORUM_APPLY_TOKEN, CMD_QUORUM_SYNC_GROUP, CMD_QUORUM_NEW_GROUP, CMD_QUORUM_GET_GROUP_SEED, CMD_QUORUM_LEAVE_GROUP, CMD_QUORUM_DEL_GROUP, CMD_QUORUM_GROUP_ADMIN, CMD_CONFIG_RELOAD, CMD_CONFIG_SAVE, CMD_MODE_BLOCKS, CMD_MODE_QUORUM, CMD_MODE_NETWORK}
	quorumCommands := []string{CMD_QUORUM_SEND, CMD_QUORUM_NICK}
	blocksCommands := []string{CMD_BLOCKS_JMP, CMD_BLOCKS_GENDOT}
	networkCommands := []string{CMD_NETWORK_PING}

	getCommands := func() []string {
		name, _ := rootPanels.GetFrontPanel()
		commands := baseCommands
		switch name {
		case "blocks":
			commands = append(commands, blocksCommands[:]...)
		case "quorum":
			commands = append(commands, quorumCommands[:]...)
		case "network":
			commands = append(commands, networkCommands[:]...)
		default:
		}
		sort.Strings(commands)
		return commands
	}

	cmdInput.SetAutocompleteFunc(func(currentText string) (entries []*cview.ListItem) {
		if len(currentText) == 0 {
			return
		}
		for _, word := range getCommands() {
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
			if strings.HasPrefix(cmdStr, CMD_QUORUM_CONNECT) {
				apiServer := strings.Replace(cmdStr, CMD_QUORUM_CONNECT, "", -1)
				apiServer = strings.TrimSpace(apiServer)
				reset("")
				Quorum(apiServer)
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_NICK) {
				reset("")
				QuorumGroupNickHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_APPLY_TOKEN) {
				reset("")
				QuorumApplyTokenHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_JOIN) {
				// join group
				reset("Joining: ")
				QuorumCmdJoinHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_SEND) {
				// send data
				reset("Loading: ")
				QuorumCmdSendHandler(cmdStr)
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_SYNC_GROUP) {
				reset("")
				QuorumForceSyncGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_NEW_GROUP) {
				// send data
				reset("Creating: ")
				QuorumNewGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_GET_GROUP_SEED) {
				reset("")
				QuorumGetGroupSeedHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_BACKUP) {
				reset("")
				QuorumBackupHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_LEAVE_GROUP) {
				reset("")
				QuorumLeaveGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_DEL_GROUP) {
				// send data
				reset("")
				QuorumDelGroupHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_GROUP_ADMIN) {
				reset("")
				QuorumGroupAdminHandler()
				return
			} else if strings.HasPrefix(cmdStr, CMD_CONFIG_RELOAD) {
				reset("")
				config.Init()
				if config.RumConfig.Quorum.Server != "" {
					Quorum(config.RumConfig.Quorum.Server)
				} else {
					Welcome()
				}
				return
			} else if strings.HasPrefix(cmdStr, CMD_CONFIG_SAVE) {
				configFilePath := config.Save()
				reset("")
				cmdInput.SetLabel("Config saved at:")
				cmdInput.SetText(configFilePath)
				return
			} else if strings.HasPrefix(cmdStr, CMD_MODE_BLOCKS) {
				reset("")
				QuorumExit()
				NetworkPageExit()

				Blocks()
			} else if strings.HasPrefix(cmdStr, CMD_MODE_QUORUM) {
				reset("")

				BlocksExit()
				NetworkPageExit()

				if config.RumConfig.Quorum.Server != "" {
					Quorum(config.RumConfig.Quorum.Server)
				} else {
					Welcome()
				}
			} else if strings.HasPrefix(cmdStr, CMD_MODE_NETWORK) {
				reset("")

				BlocksExit()
				QuorumExit()
				NetworkPage()
			} else if strings.HasPrefix(cmdStr, CMD_NETWORK_PING) {
				reset("")
				go NetworkRefreshAll()
			} else if strings.HasPrefix(cmdStr, CMD_BLOCKS_JMP) {
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
