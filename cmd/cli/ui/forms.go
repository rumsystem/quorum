package ui

import (
	"fmt"
	"io/ioutil"
	"os"

	"code.rocketnine.space/tslocum/cview"
	"github.com/rumsystem/quorum/cmd/cli/api"
)

// global forms

var formMode = false

var groupForm = cview.NewForm()
var groupReqStruct = api.CreateGroupReqStruct{
	Name:           "",
	ConsensusType:  "poa",
	EncryptionType: "public",
	AppKey:         "",
}

func formInit() {
	createGroupFormInit()
}

func createGroupFormInit() {
	groupForm.AddDropDownSimple("Encryption Type", 0, func(index int, option *cview.DropDownOption) {
		groupReqStruct.EncryptionType = option.GetText()
	}, "public", "private")

	groupForm.AddInputField("Group Name", "", 20, nil, func(name string) {
		groupReqStruct.Name = name
	})
	groupForm.AddInputField("App Key", "", 20, nil, func(key string) {
		groupReqStruct.AppKey = key
	})
	groupForm.AddButton("Save", func() {
		if groupReqStruct.Name == "" || groupReqStruct.AppKey == "" {
			Error("invalid parameter", "Name and AppKey should not be empty")
			return
		}
		go goQuorumCreateGroup()
		rootPanels.HidePanel("form.group")
		rootPanels.SendToBack("form.group")
		formMode = false
	})
	groupForm.AddButton("Cancel", func() {
		// backto last
		rootPanels.HidePanel("form.group")
		rootPanels.SendToBack("form.group")
		formMode = false
	})
	groupForm.SetBorder(true)
	groupForm.SetTitle("Create Group")
	groupForm.SetTitleAlign(cview.AlignCenter)

	rootPanels.AddPanel("form.group", groupForm, true, false)
}

func CreateGroupForm() {
	formMode = true
	rootPanels.ShowPanel("form.group")
	rootPanels.SendToFront("form.group")
	App.SetFocus(groupForm)
}

func goQuorumCreateGroup() {
	seedBytes, err := api.CreateGroup(groupReqStruct)
	if err != nil {
		Error("Failed to create group", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", groupReqStruct.Name))
		cmdInput.SetText("Created")

		tmpFile, err := ioutil.TempFile(os.TempDir(), "quorum-seed-")
		if err != nil {
			Error("Cannot create temporary file to save seed", err.Error())
			return
		}
		if _, err = tmpFile.Write(seedBytes); err != nil {
			Error("Failed to write to seed file", err.Error())
			return
		}

		if err := tmpFile.Close(); err != nil {
			Error("Failed to close the seed file", err.Error())
			return
		}
		Info(fmt.Sprintf("Group %s created", groupReqStruct.Name), fmt.Sprintf("Seed saved at: %s. Be sure to keep it well.", tmpFile.Name()))
	}
}
