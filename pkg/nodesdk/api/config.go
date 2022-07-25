package nodesdkapi

import "fmt"

//const name
const GROUP_NAME string = "group_name"
const GROUP_ID string = "group_id"
const GROUP_OWNER_PUBKEY string = "owner_pubkey"
const GROUP_ITEMS string = "group_items"
const GROUP_ITEM string = "group_item"
const GENESIS_BLOCK string = "genesis_block"
const NODE_VERSION string = "node_version"
const NODE_PUBKEY string = "node_publickey"
const NODE_STATUS string = "node_status"
const NODE_ID string = "node_id"
const SIGNATURE string = "signature"
const TRX_ID string = "trx_id"
const PEERS string = "peers"
const NODETYPE string = "node_type"

const (
	Add      = "Add"
	Like     = "Like"
	Dislike  = "Dislike"
	Update   = "Update"
	Remove   = "Remove"
	Group    = "Group"
	User     = "User"
	Auth     = "Auth"
	Note     = "Note"
	Page     = "Page"
	File     = "File"
	Producer = "Producer"
	Announce = "Announce"
	App      = "App"
)

//config
const GROUP_NAME_MIN_LENGTH int = 5

//error
const ERROR_INFO string = "error"

const BLACK_LIST_OP_PREFIX string = "blklistop_"

const GROUP_INFO string = "group_info"

const AUTH_TYPE string = "auth_type"
const AUTH_ALLOWLIST string = "auth_allowlist"
const AUTH_DENYLIST string = "auth_denylist"

const APPCONFIG_KEYLIST string = "appconfig_listlist"
const APPCONFIG_ITEM_BYKEY string = "appconfig_item_bykey"

const ANNOUNCED_PRODUCER string = "announced_producer"
const ANNOUNCED_USER string = "announced_user"
const GROUP_PRODUCER string = "group_producer"

const POST_TRX_URI string = "/api/v1/node/trx"
const GET_CTN_URI string = "/api/v1/node/groupctn"
const GET_CHAIN_DATA_URI string = "/api/v1/node/getchaindata"

func GetPostTrxURI(groupId string) string {
	return fmt.Sprintf("%s/%s", POST_TRX_URI, groupId)
}

func GetGroupCtnURI(groupId string) string {
	return fmt.Sprintf("%s/%s", GET_CTN_URI, groupId)
}

func GetChainDataURI(groupId string) string {
	return fmt.Sprintf("%s/%s", GET_CHAIN_DATA_URI, groupId)
}

type NodeSDKSendTrxItem struct {
	TrxItem []byte
}

type NodeSDKTrxItem struct {
	TrxBytes []byte
}

type NodeSDKGetChainDataItem struct {
	ReqType string
	Req     []byte
}

type AppConfigKeyListItem struct {
	GroupId string
}

type AppConfigItem struct {
	GroupId string
	Key     string
}

type AnnGrpProducer struct {
	GroupId string
}

type GrpProducer struct {
	GroupId string
}

type AnnGrpUser struct {
	GroupId    string
	SignPubkey string
}

type ProducerListItem struct {
	ProducerPubkey string
	OwnerPubkey    string
	OwnerSign      string
	TimeStamp      int64
	BlockProduced  int64
}

type AuthTypeItem struct {
	GroupId string
	TrxType string
}

type AuthAllowListItem struct {
	GroupId string
}

type AuthDenyListItem struct {
	GroupId string
}

type GrpInfo struct {
	GroupId string
}

type GrpInfoNodeSDK struct {
	GroupId        string
	Owner          string
	HighestBlockId string
	HighestHeight  int64
	LatestUpdate   int64
	Provider       string
	Singature      string
}
