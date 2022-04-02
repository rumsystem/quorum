package ui

import (
	"runtime"

	"code.rocketnine.space/tslocum/cbind"
	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/cache"
	"github.com/rumsystem/quorum/cmd/cli/config"
)

var App = cview.NewApplication()

// root panels contains about page and main group page
var rootPanels = cview.NewPanels()

// root layout
var layout = cview.NewFlex()

// terminal size
var termW, termH int

func wrapGlobal(f func()) func(ev *tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		if !cmdMode && !formMode {
			f()
			return nil
		}
		return ev
	}
}

func Init() {
	cmdInputInit()
	modalInit()
	formInit()

	helpPageInit()
	welcomePageInit()
	blocksPageInit()
	quorumPageInit()
	adminPageInit()
	chainConfigPageInit()
	networkPageInit()
	pubqueuePageInit()

	// display groups
	App.EnableMouse(false)
	App.SetAfterResizeFunc(func(width int, height int) {
		termW = width
		termH = height
		// TODO: rerender
	})

	layout.SetDirection(cview.FlexRow)
	layout.AddItem(rootPanels, 0, 1, true)
	layout.AddItem(cmdInput, 1, 1, false)

	App.SetRoot(layout, true)

	gInputHandler := cbind.NewConfiguration()
	gInputHandler.SetRune(tcell.ModNone, 'q', wrapGlobal(shutdown))
	if runtime.GOOS == "windows" {
		gInputHandler.Set("Shift+?", wrapGlobal(Help))
	} else {
		gInputHandler.SetRune(tcell.ModNone, '?', wrapGlobal(Help))
	}
	gInputHandler.SetRune(tcell.ModNone, ' ', wrapGlobal(cmdActivate))

	App.SetInputCapture(gInputHandler.Capture)

	if config.RumConfig.Quorum.Server != "" {
		Quorum(config.RumConfig.Quorum.Server)
	} else {
		Welcome()
	}
}

func shutdown() {
	// Graceful shutdown
	cache.Shutdown()
	config.Save()
	App.Stop()
}
