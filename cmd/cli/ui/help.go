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
		"Ctrl + b\tGo up a page.\n" +
		"Ctrl + f\tGo down a page.\n" +
		"g\tGo to the top of document.\n" +
		"G\tGo to the bottom of document.\n" +
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
		fmt.Sprintf("%s <group_name>\t Create a new group.\n", CMD_QUORUM_NEW_GROUP) +
		fmt.Sprintf("%s\t Delete cuerrent group.\n", CMD_QUORUM_DEL_GROUP) +
		fmt.Sprintf("%s \t Reload the config.\n", CMD_CONFIG_RELOAD) +
		fmt.Sprintf("%s \t Save the config.\n", CMD_CONFIG_SAVE) +
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

func helpInit() {
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
