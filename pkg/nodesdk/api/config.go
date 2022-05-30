package nodesdkapi

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

const JwtToken string = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

const AUTH_TYPE string = "auth_type"
const AUTH_ALLOWLIST string = "auth_allowlist"
const AUTH_DENYLIST string = "auth_denylist"

const APPCONFIG_KEYLIST string = "appconfig_listlist"
const APPCONFIG_ITEM_BYKEY string = "appconfig_item_bykey"

const ANNOUNCED_PRODUCER string = "announced_producer"
const ANNOUNCED_USER string = "announced_user"
const GROUP_PRODUCER string = "group_producer"

const POST_TRX_URI string = "/api/v1/nodesdk/trx"
const GET_CTN_URI string = "/api/v1/nodesdk/groupctn"
const GET_CHAIN_DATA_URI string = "/api/v1/nodesdk/getchaindata"

type NodeSDKSendTrxItem struct {
	GroupId string
	TrxItem []byte
}

type NodeSDKTrxItem struct {
	TrxBytes []byte
	JwtToken string
}

type NodeSDKGetChainDataItem struct {
	GroupId string
	ReqType string
	Req     []byte
}

type AppConfigKeyListItem struct {
	GroupId  string
	JwtToken string
}

type AppConfigItem struct {
	GroupId  string
	Key      string
	JwtToken string
}

type AnnGrpProducer struct {
	GroupId  string
	JwtToken string
}

type GrpProducer struct {
	GroupId  string
	JwtToken string
}

type AnnGrpUser struct {
	GroupId    string
	SignPubkey string
	JwtToken   string
}

type ProducerListItem struct {
	ProducerPubkey string
	OwnerPubkey    string
	OwnerSign      string
	TimeStamp      int64
	BlockProduced  int64
}

type AuthTypeItem struct {
	GroupId  string
	TrxType  string
	JwtToken string
}

type AuthAllowListItem struct {
	GroupId  string
	JwtToken string
}

type AuthDenyListItem struct {
	GroupId  string
	JwtToken string
}
