package ui

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	appKeyInput := groupForm.GetFormItemByLabel("App Key").(*cview.InputField)
	DefaultAppKeys := []string{"group_timeline", "group_post", "group_note"}

	appKeyInput.SetAutocompleteFunc(func(currentText string) (entries []*cview.ListItem) {
		if len(currentText) == 0 {
			return
		}
		for _, word := range DefaultAppKeys {
			if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
				entries = append(entries, cview.NewListItem(word))
			}
		}
		if len(entries) == 0 {
			entries = nil
		}
		return
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

func SaveSeedToTmpFile(seedBytes []byte) (*os.File, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "quorum-seed-")
	if err != nil {
		return nil, err
	}
	if _, err = tmpFile.Write(seedBytes); err != nil {
		return nil, err
	}

	if err := tmpFile.Close(); err != nil {
		return nil, err
	}
	return tmpFile, nil
}

func goQuorumCreateGroup() {
	seedBytes, err := api.CreateGroup(groupReqStruct)
	if err != nil {
		Error("Failed to create group", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", groupReqStruct.Name))
		cmdInput.SetText("Created")
		tmpFile, err := SaveSeedToTmpFile(seedBytes)
		if err != nil {
			Error("Failed to cache group seed", err.Error())
			return
		}
		Info(fmt.Sprintf("Group %s created", groupReqStruct.Name), fmt.Sprintf("Seed saved at: %s. Be sure to keep it well.", tmpFile.Name()))
	}
}
