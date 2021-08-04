package ui

import (
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/huo-ju/quorum/cmd/cli/config"
	"github.com/gdamore/tcell/v2"
)

// cmd input at the bottom
var cmdInput = cview.NewInputField()
var cmdMode = false

const (
	CMD_QUORUM_CONNECT     string = "/connect"
	CMD_QUORUM_JOIN        string = "/join"
	CMD_QUORUM_SEND        string = "/send"
	CMD_QUORUM_NICK        string = "/nick"
	CMD_QUORUM_NEW_GROUP   string = "/group.create"
	CMD_QUORUM_LEAVE_GROUP string = "/group.leave"
	CMD_QUORUM_DEL_GROUP   string = "/group.delete"
	CMD_CONFIG_RELOAD      string = "/config.reload"
	CMD_CONFIG_SAVE        string = "/config.save"
)

func cmdInputInit() {
	commands := []string{CMD_QUORUM_CONNECT, CMD_QUORUM_JOIN, CMD_QUORUM_SEND, CMD_QUORUM_NICK, CMD_QUORUM_NEW_GROUP, CMD_QUORUM_LEAVE_GROUP, CMD_QUORUM_DEL_GROUP, CMD_CONFIG_RELOAD, CMD_CONFIG_SAVE}
	cmdInput.SetAutocompleteFunc(func(currentText string) (entries []*cview.ListItem) {
		if len(currentText) == 0 {
			return
		}
		for _, word := range commands {
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
			} else if strings.HasPrefix(cmdStr, CMD_QUORUM_NEW_GROUP) {
				// send data
				reset("Creating: ")
				QuorumNewGroupHandler(cmdStr)
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
