package api

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
const USER_ID string = "user_id"
const SIGNATURE string = "signature"
const TRX_ID string = "trx_id"

//config
const GROUP_NAME_MIN_LENGTH int = 5

//error
const ERROR_INFO string = "error"

const (
	Add    = "Add"
	Remove = "Remove"
	Group  = "Group"
	User   = "User"
	Auth   = "Auth"
	Note   = "Note"
)

const BLACK_LIST_OP_PREFIX string = "blklistop_"
