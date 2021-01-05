package ui

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

var helpCells = strings.TrimSpace(
	"Shortcuts:\n" +
		"?\tBring up this help.\n" +
		"Esc\tLeave the help.\n" +
		"\tIn Content View, it will deselect item.\n" +
		"q\tQuit.\n" +
		"left/down/up/right, h/j/k/l\tScroll.\n" +
		"Shift + h/j/k/l\tMove between widgets.\n" +
		"Ctrl + b\tScroll back half screen.\n" +
		"Ctrl + f\tScroll forward half screen.\n" +
		"g\tGo to the top of document.\n" +
		"G\tGo to the bottom of document.\n" +
		"N\tGo next page.\n" +
		"P\tGo previous page.\n" +
		"M\tMute user.\n" +
		"U\tUnmute user.\n" +
		"Tab\tNavigate to the next item.\n" +
		"Shift-Tab\tNavigate to the previous item.\n" +
		"Enter\tIn Content View, it will fetch the detail about the item you select.\n" +
		"\tIn GroupList View, it will switch to the group you select.\n" +
		"\tIn Root View, it will set focus on the Content View.\n" +
		"a-z\tIn GroupList View, to quick-switch between groups.\n" +
		"Space\tOpen the command prompt.\n" +
		"Commands:\n" +
		fmt.Sprintf("%s ip:port\t Connect to your API server.\n", CMD_QUORUM_CONNECT) +
		fmt.Sprintf("%s @seed.json\t Join into a group from seed file.\n", CMD_QUORUM_JOIN) +
		fmt.Sprintf("%s xxxxxxxxxx\t Join into a group from raw seed string.\n", CMD_QUORUM_JOIN) +
		fmt.Sprintf("%s msg\t Send message to cuerrent group.\n", CMD_QUORUM_SEND) +
		fmt.Sprintf("%s <nickname>\t Change your nickname in cuerrent group.\n", CMD_QUORUM_NICK) +
		fmt.Sprintf("%s\t Apply JWT if you don't applied before.\n", CMD_QUORUM_APPLY_TOKEN) +
		fmt.Sprintf("%s\t Trigger a sync on current group manually.\n", CMD_QUORUM_SYNC_GROUP) +
		fmt.Sprintf("%s <group_name>\t Create a new group.\n", CMD_QUORUM_NEW_GROUP) +
		fmt.Sprintf("%s\t Delete cuerrent group(you are owner).\n", CMD_QUORUM_DEL_GROUP) +
		fmt.Sprintf("%s\t Leave cuerrent group.\n", CMD_QUORUM_LEAVE_GROUP) +
		fmt.Sprintf("%s \t Reload the config.\n", CMD_CONFIG_RELOAD) +
		fmt.Sprintf("%s \t Save the config.\n", CMD_CONFIG_SAVE) +
		fmt.Sprintf("%s \t Switch to blocks mode.\n", CMD_MODE_BLOCKS) +
		fmt.Sprintf("%s <block_num>\t Jump to block (available in blocks mode).\n", CMD_BLOCKS_JMP) +
		fmt.Sprintf("%s \t Switch to network mode.\n", CMD_MODE_NETWORK) +
		fmt.Sprintf("%s \t Ping connected peers(available in network mode).\n", CMD_NETWORK_PING) +
		fmt.Sprintf("%s \t Switch to quorum mode(default).\n", CMD_MODE_QUORUM) +
		"",
)

var helpTable = cview.NewTextView()

// Help displays the help and keybindings.
func Help() {
	helpTable.ScrollToBeginning()
	rootPanels.ShowPanel("help")
	rootPanels.SendToFront("help")
	App.SetFocus(helpTable)
}

func helpPageInit() {
	helpTable.SetPadding(0, 0, 1, 1)
	helpTable.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc || key == tcell.KeyEnter {
			rootPanels.HidePanel("help")
			App.Draw()
		}
	})

	lines := strings.Split(helpCells, "\n")
	w := tabwriter.NewWriter(helpTable, 0, 8, 2, ' ', 0)
	for i, line := range lines {
		if i > 0 && line[0] != '\t' {
			fmt.Fprintln(w, "\t")
		}
		fmt.Fprintln(w, line)
	}

	w.Flush()

	rootPanels.AddPanel("help", helpTable, true, false)
}
