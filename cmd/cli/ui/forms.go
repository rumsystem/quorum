package ui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
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

var PANNEL_GROUP_FORM = "form.group"
var PANNEL_GROUP_CONFIG_FORM = "form.group.config"
var PANNEL_CHAIN_AUTH_MODE_FORM = "form.chain.auth.mode"

var groupConfigForm = cview.NewForm()
var groupConfigParam = handlers.AppConfigParam{
	Action:  "add",
	GroupId: "",
	Name:    "",
	Type:    "",
	Value:   "",
	Memo:    "",
}

var chainAuthModeForm = cview.NewForm()

type ChainAuthModeFormParam struct {
	GroupId string
	TrxType string
	Mode    string
}

var chainAuthModeParam = ChainAuthModeFormParam{
	GroupId: "",
	TrxType: "",
	Mode:    "",
}

func formInit() {
	createGroupFormInit()
	groupConfigFormInit()
	chainConfigFormInit()
}

func chainConfigFormInit() {
	chainAuthModeForm.AddInputField("GroupId", "", 40, nil, func(string) {
	})
	chainAuthModeForm.AddInputField("TrxType", "", 40, nil, func(string) {
	})
	chainAuthModeForm.AddDropDownSimple("Mode", 0, func(index int, option *cview.DropDownOption) {
		chainAuthModeParam.Mode = option.GetText()
	}, "FOLLOW_DNY_LIST", "FOLLOW_ALW_LIST")

	chainAuthModeForm.AddButton("OK", func() {
		if chainAuthModeParam.GroupId == "" || chainAuthModeParam.TrxType == "" {
			Error("invalid parameter", "GroupId and TrxType should not be empty")
			return
		}
		go goQuorumUpdateChainAuthMode()
		rootPanels.HidePanel(PANNEL_CHAIN_AUTH_MODE_FORM)
		rootPanels.SendToBack(PANNEL_CHAIN_AUTH_MODE_FORM)
		formMode = false
	})
	chainAuthModeForm.AddButton("Cancel", func() {
		// backto last
		rootPanels.HidePanel(PANNEL_CHAIN_AUTH_MODE_FORM)
		rootPanels.SendToBack(PANNEL_CHAIN_AUTH_MODE_FORM)
		formMode = false
	})
	chainAuthModeForm.SetBorder(true)
	chainAuthModeForm.SetTitle("Group Config")
	chainAuthModeForm.SetTitleAlign(cview.AlignCenter)

	rootPanels.AddPanel(PANNEL_CHAIN_AUTH_MODE_FORM, chainAuthModeForm, true, false)
}

func goQuorumUpdateChainAuthMode() {
	config := handlers.TrxAuthModeParams{
		strings.ToLower(chainAuthModeParam.TrxType),
		strings.ToLower(chainAuthModeParam.Mode),
	}
	configBytes, _ := json.Marshal(&config)
	res, err := api.UpdateChainConfig(
		chainAuthModeParam.GroupId,
		"set_trx_auth_mode",
		string(configBytes),
		"updated by cli",
	)
	if err != nil {
		Error("Failed to update chain config", err.Error())
		return
	}
	Info(fmt.Sprintf("Chain config updated"), fmt.Sprintf("TrxId: %s\nSign: %s\n", res.TrxId, res.Sign))

}

func ChainAuthModeForm(groupId, trxType, mode string) {
	var indexOf = func(word string, data []string) int {
		for k, v := range data {
			if strings.ToUpper(word) == v {
				return k
			}
		}
		return -1
	}
	chainAuthModeParam.GroupId = groupId
	chainAuthModeParam.TrxType = trxType
	chainAuthModeParam.Mode = mode

	chainAuthModeForm.GetFormItemByLabel("GroupId").(*cview.InputField).SetText(groupId)
	chainAuthModeForm.GetFormItemByLabel("TrxType").(*cview.InputField).SetText(trxType)
	options := []string{"FOLLOW_DNY_LIST", "FOLLOW_ALW_LIST"}
	chainAuthModeForm.GetFormItemByLabel("Mode").(*cview.DropDown).SetCurrentOption(indexOf(mode, options))
	formMode = true
	rootPanels.ShowPanel(PANNEL_CHAIN_AUTH_MODE_FORM)
	rootPanels.SendToFront(PANNEL_CHAIN_AUTH_MODE_FORM)
	App.SetFocus(chainAuthModeForm)
}

func groupConfigFormInit() {
	groupConfigForm.AddDropDownSimple("Action", 0, func(index int, option *cview.DropDownOption) {
		groupConfigParam.Action = option.GetText()
	}, "add", "del")

	groupConfigForm.AddInputField("Group Id", "", 40, nil, func(groupId string) {
		groupConfigParam.GroupId = groupId
	})
	groupConfigForm.AddInputField("Name", "", 20, nil, func(name string) {
		groupConfigParam.Name = name
	})
	groupConfigForm.AddDropDownSimple("Type", 0, func(index int, option *cview.DropDownOption) {
		groupConfigParam.Type = option.GetText()
	}, "string", "int", "bool")
	groupConfigForm.AddInputField("Value", "", 20, nil, func(value string) {
		groupConfigParam.Value = value
	})
	groupConfigForm.AddInputField("Memo", "", 20, nil, func(memo string) {
		groupConfigParam.Memo = memo
	})

	groupConfigForm.AddButton("OK", func() {
		if groupConfigParam.GroupId == "" || groupConfigParam.Name == "" {
			Error("invalid parameter", "GroupId and Name should not be empty")
			return
		}
		go goQuorumUpdateGroupConfig()
		rootPanels.HidePanel(PANNEL_GROUP_CONFIG_FORM)
		rootPanels.SendToBack(PANNEL_GROUP_CONFIG_FORM)
		formMode = false
	})
	groupConfigForm.AddButton("Cancel", func() {
		// backto last
		rootPanels.HidePanel(PANNEL_GROUP_CONFIG_FORM)
		rootPanels.SendToBack(PANNEL_GROUP_CONFIG_FORM)
		formMode = false
	})
	groupConfigForm.SetBorder(true)
	groupConfigForm.SetTitle("Group Config")
	groupConfigForm.SetTitleAlign(cview.AlignCenter)

	rootPanels.AddPanel(PANNEL_GROUP_CONFIG_FORM, groupConfigForm, true, false)
}

func GroupConfigFormShow(groupId string, item *handlers.AppConfigKeyItem) {
	groupConfigParam.GroupId = groupId
	groupConfigParam.Name = item.Name
	groupConfigParam.Type = item.Type
	groupConfigParam.Value = item.Value
	groupConfigParam.Memo = item.Memo
	groupConfigForm.GetFormItemByLabel("Group Id").(*cview.InputField).SetText(groupId)
	groupConfigForm.GetFormItemByLabel("Name").(*cview.InputField).SetText(item.Name)
	options := []string{"string", "int", "bool"}
	var indexOf = func(word string, data []string) int {
		for k, v := range data {
			if strings.ToLower(word) == v {
				return k
			}
		}
		return -1
	}
	groupConfigForm.GetFormItemByLabel("Type").(*cview.DropDown).SetCurrentOption(indexOf(item.Type, options))
	groupConfigForm.GetFormItemByLabel("Value").(*cview.InputField).SetText(item.Value)
	groupConfigForm.GetFormItemByLabel("Memo").(*cview.InputField).SetText(item.Memo)

	formMode = true
	rootPanels.ShowPanel(PANNEL_GROUP_CONFIG_FORM)
	rootPanels.SendToFront(PANNEL_GROUP_CONFIG_FORM)
	App.SetFocus(groupConfigForm)
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
		rootPanels.HidePanel(PANNEL_GROUP_FORM)
		rootPanels.SendToBack(PANNEL_GROUP_FORM)
		formMode = false
	})
	groupForm.AddButton("Cancel", func() {
		// backto last
		rootPanels.HidePanel(PANNEL_GROUP_FORM)
		rootPanels.SendToBack(PANNEL_GROUP_FORM)
		formMode = false
	})
	groupForm.SetBorder(true)
	groupForm.SetTitle("Create Group")
	groupForm.SetTitleAlign(cview.AlignCenter)

	rootPanels.AddPanel(PANNEL_GROUP_FORM, groupForm, true, false)
}

func CreateGroupForm() {
	formMode = true
	rootPanels.ShowPanel(PANNEL_GROUP_FORM)
	rootPanels.SendToFront(PANNEL_GROUP_FORM)
	App.SetFocus(groupForm)
}

func SaveToTmpFile(bytes []byte, prefix string) (*os.File, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), prefix)
	if err != nil {
		return nil, err
	}
	if _, err = tmpFile.Write(bytes); err != nil {
		return nil, err
	}

	if err := tmpFile.Close(); err != nil {
		return nil, err
	}
	return tmpFile, nil
}

func SaveSeedToTmpFile(seedBytes []byte) (*os.File, error) {
	return SaveToTmpFile(seedBytes, "quorum-seed-")
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

func goQuorumUpdateGroupConfig() {
	res, err := api.ModifyGroupConfig(
		groupConfigParam.Action,
		groupConfigParam.GroupId,
		groupConfigParam.Name,
		groupConfigParam.Type,
		groupConfigParam.Value,
		groupConfigParam.Memo,
	)
	if err != nil {
		Error("Failed to update group config", err.Error())
	} else {
		cmdInput.SetLabel(fmt.Sprintf("Group %s: ", groupConfigParam.GroupId))
		cmdInput.SetText("Updated")
		Info(fmt.Sprintf("Config %s", groupConfigParam.Name), fmt.Sprintf("sign: %s\ntrxid: %s\n", res.Sign, res.TrxId))
	}
}
