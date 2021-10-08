package ui

import (
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/rumsystem/quorum/cmd/cli/config"
)

// global modal
var errorModal = cview.NewModal()
var infoModal = cview.NewModal()
var yesNoModal = cview.NewModal()

var yesNoCh = make(chan bool)

func modalInit() {
	infoModalInit()
	errorModalInit()
	yesNoModalInit()
}

func infoModalInit() {
	infoModal.AddButtons([]string{"Ok"})
	infoModal.SetBackgroundColor(tcell.ColorBlack)
	infoModal.SetButtonBackgroundColor(tcell.ColorWhite)
	infoModal.SetButtonTextColor(tcell.ColorBlack)
	infoModal.SetTextColor(tcell.ColorWhite)
	form := infoModal.GetForm()
	form.SetButtonBackgroundColorFocused(tcell.ColorBlack)
	form.SetButtonTextColorFocused(tcell.ColorWhite)
	frame := infoModal.GetFrame()
	frame.SetBorderColor(tcell.ColorWhite)
	frame.SetTitleColor(tcell.ColorWhite)

	rootPanels.AddPanel("info", infoModal, false, false)

	infoModal.SetBorder(true)
	infoModal.GetFrame().SetTitleAlign(cview.AlignCenter)
	infoModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		rootPanels.HidePanel("info")
		App.Draw()
	})
}

func errorModalInit() {
	errorModal.AddButtons([]string{"Ok"})
	errorModal.SetBackgroundColor(tcell.ColorBlack)
	errorModal.SetButtonBackgroundColor(tcell.ColorWhite)
	errorModal.SetButtonTextColor(tcell.ColorBlack)
	errorModal.SetTextColor(tcell.ColorWhite)
	form := errorModal.GetForm()
	form.SetButtonBackgroundColorFocused(tcell.ColorBlack)
	form.SetButtonTextColorFocused(tcell.ColorWhite)
	frame := errorModal.GetFrame()
	frame.SetBorderColor(tcell.ColorWhite)
	frame.SetTitleColor(tcell.ColorWhite)

	rootPanels.AddPanel("error", errorModal, false, false)

	errorModal.SetBorder(true)
	errorModal.GetFrame().SetTitleAlign(cview.AlignCenter)
	errorModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		rootPanels.HidePanel("error")
		App.Draw()
	})
}

func yesNoModalInit() {
	yesNoModal.AddButtons([]string{"Yes", "No"})
	yesNoModal.SetBackgroundColor(tcell.ColorBlack)
	yesNoModal.SetButtonBackgroundColor(tcell.ColorBlack)
	yesNoModal.SetButtonTextColor(tcell.ColorBlack)
	yesNoModal.SetTextColor(tcell.ColorWhite)
	form := yesNoModal.GetForm()
	form.SetButtonBackgroundColor(tcell.ColorBlack)
	form.SetButtonBackgroundColorFocused(tcell.ColorBlack)
	form.SetButtonTextColorFocused(tcell.ColorYellow)
	form.SetButtonTextColor(tcell.ColorWhite)
	frame := yesNoModal.GetFrame()
	frame.SetBorderColor(tcell.ColorWhite)
	frame.SetTitleColor(tcell.ColorWhite)

	yesNoModal.SetBorder(true)
	yesNoModal.GetFrame().SetTitleAlign(cview.AlignCenter)
	yesNoModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Yes" {
			yesNoCh <- true
			return
		}
		yesNoCh <- false
	})

	rootPanels.AddPanel("yesno", yesNoModal, false, false)
}

func Error(title, text string) {
	if text == "" {
		text = "No additional information."
	} else {
		text = strings.ToUpper(string([]rune(text)[0])) + text[1:]
	}
	// Add spaces to title for aesthetic reasons
	title = " " + strings.TrimSpace(title) + " "

	config.Logger.Errorf("%s: %s\n", title, text)

	errorModal.GetFrame().SetTitle(title)
	errorModal.SetText(text)
	rootPanels.ShowPanel("error")
	rootPanels.SendToFront("error")
	App.SetFocus(errorModal)
	App.Draw()

}

func Info(title, text string) {
	if text == "" {
		text = "No additional information."
	} else {
		text = strings.ToUpper(string([]rune(text)[0])) + text[1:]
	}
	// Add spaces to title for aesthetic reasons
	title = " " + strings.TrimSpace(title) + " "

	infoModal.GetFrame().SetTitle(title)
	infoModal.SetText(text)
	rootPanels.ShowPanel("info")
	rootPanels.SendToFront("info")
	App.SetFocus(infoModal)
	App.Draw()
}

// YesNo will block on channel
// so better to run in a goroutine
func YesNo(text string) bool {
	yesNoModal.GetFrame().SetTitle("")
	yesNoModal.SetText(text)
	rootPanels.ShowPanel("yesno")
	rootPanels.SendToFront("yesno")
	App.SetFocus(yesNoModal)
	App.Draw()

	resp := <-yesNoCh
	rootPanels.HidePanel("yesno")
	App.Draw()
	return resp
}
