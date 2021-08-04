package ui

import (
	"fmt"
	"strings"

	"code.rocketnine.space/tslocum/cview"
)

var welcomePage = cview.NewFlex()
var welcomePageLeft = cview.NewTextView()
var welcomePageRight = cview.NewTextView()

var quorumIcon = `                                        
       -2ooojojoojooooojjooojjjj^       
      =Bdr@R~~>@R~~~~~~n@1~~1@2u@\      
     _BB|N@x^^}@z^^^^^^/@5^^rBBrE@\     
     8Byyyyyyyyyyyyyyyyyyyyyyyyyyg@-    
    :@dxxxxxxxxxxxxxxxxxxxxxxxxxxs@c    
    y@YxBBxxxxBBxxxxxxxxd@nxxxR@uxQQ    
    8g  @W    Bd        v@:   r@~ l@.   
    @K .@1   .@a        -y.   ,@x ^@^   
   .@u ,@v   -@K        .N-   .@l ,@x   
   _@i =@\   _@2         ~     @y .@i   
   .@} :@v   -@K        ;@^   .@l ,@x   
    @I .@n   .@a        r@~   ,@x >@^   
    Qg  @P    Bd        v@:   r@~ 1@.   
    k@xvBBvvvvBBvvvvvvvv5@ivvvb@ivQQ    
    :@RYYYYYYYYYYYYYYYYYYYYYYYYYYH@y    
     QQnnnnnnnnnnnnnnnnnnnnnnnnnnE@-    
     ,BB/d@xrru@Irrrrrrv@NrrrBBrE@x     
      ~BNr@d===@d======}@}==}@yi@v      
       _zfffffffffffffffffffffffr       
                                        `
var welcomeHeading = "Welcom to the rumcli!\n\n\nBefore you start using it\nMake sure you have completed the following steps\n\n"
var welcomeContent = "1. Connect to an API server\n" +
	"2. Join into a group\n" +
	"3. Syncronize content\n\n"
var welcomeFooter = "Press <space> to activate the command prompt\n\n" +
	"Always press ? to show more help\n"

func Welcome() {
	rootPanels.ShowPanel("welcome")
	rootPanels.SendToFront("welcome")
	App.SetFocus(welcomePage)
}

func welcomePageInit() {
	welcomePage.SetTitle("Welcome")

	var welcomText string
	welcomText += welcomeHeading
	contentLines := strings.Split(welcomeContent, "\n")
	maxLen := 0
	for _, line := range contentLines {
		if maxLen < len(line) {
			maxLen = len(line)
		}
	}
	welcomePageLeft.SetText(quorumIcon)
	welcomePageLeft.SetTextAlign(cview.AlignRight)
	welcomePageLeft.SetVerticalAlign(cview.AlignMiddle)
	welcomePageLeft.SetPadding(0, 0, 1, 1)

	for _, line := range contentLines {
		welcomText += fmt.Sprintf("%*s\n", -(maxLen + 1), line)
	}
	welcomText += welcomeFooter
	welcomePageRight.SetText(welcomText)
	welcomePageRight.SetTextAlign(cview.AlignLeft)
	welcomePageRight.SetVerticalAlign(cview.AlignMiddle)
	welcomePageRight.SetPadding(0, 0, 1, 1)

	welcomePage.AddItem(welcomePageLeft, 0, 1, false)
	welcomePage.AddItem(welcomePageRight, 0, 1, false)

	rootPanels.AddPanel("welcome", welcomePage, true, false)
}
