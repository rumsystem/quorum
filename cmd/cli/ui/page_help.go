package ui

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/model"
)

func getCommandHelpMessages() string {
	help := ""
	for _, item := range model.BaseCommands {
		help += strings.Replace(item.Help, "$cmd", item.Cmd, -1)
	}

	for _, item := range model.QuorumCommands {
		help += strings.Replace(item.Help, "$cmd", item.Cmd, -1)
	}

	for _, item := range model.BlocksCommands {
		help += strings.Replace(item.Help, "$cmd", item.Cmd, -1)
	}

	for _, item := range model.NetworkCommands {
		help += strings.Replace(item.Help, "$cmd", item.Cmd, -1)
	}
	return help
}

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
		getCommandHelpMessages() +
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
