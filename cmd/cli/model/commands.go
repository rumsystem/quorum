package model

type Command struct {
	Cmd  string
	Help string
}

type CommandList []Command

func (e CommandList) Len() int {
	return len(e)
}

func (e CommandList) Less(i, j int) bool {
	return e[i].Cmd > e[j].Cmd
}

func (e CommandList) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

var (
	CommandConnect          = Command{Cmd: "/connect", Help: "$cmd ip:port\t Connect to your API server.\n"}
	CommandBackup           = Command{Cmd: "/backup", Help: "$cmd\t To get backup file from API server.\n"}
	CommandJoin             = Command{Cmd: "/join", Help: "$cmd @seed.json\t Join into a group from seed file.\n$cmd xxxxxxxxxx\t Join into a group from raw seed string.\n"}
	CommandTokenApply       = Command{Cmd: "/token.apply", Help: "$cmd\t Apply JWT if you don't applied before.\n"}
	CommandGroupSync        = Command{Cmd: "/group.sync", Help: "$cmd\t Trigger a sync on current group manually.\n"}
	CommandGroupSeed        = Command{Cmd: "/group.seed", Help: "$cmd\t Get current group's seed.\n"}
	CommandGroupCreate      = Command{Cmd: "/group.create", Help: "$cmd\t Open a creation form to create a group.\n"}
	CommandGroupLeave       = Command{Cmd: "/group.leave", Help: "$cmd\t Leave cuerrent group.\n"}
	CommandGroupDelete      = Command{Cmd: "/group.delete", Help: "$cmd\t Delete cuerrent group(if you are the owner).\n"}
	CommandGroupAdmin       = Command{Cmd: "/group.admin", Help: "$cmd\t Open admin page to manage current group(manage keys, producers, etc.)\n"}
	CommandGroupChainConfig = Command{Cmd: "/group.chain.config", Help: "$cmd\t Open chain config page to manage current chain(manage allow/deny list, auth mode.)\n"}
	CommandConfigReload     = Command{Cmd: "/config.reload", Help: "$cmd\t Reload config from disk.\n"}
	CommandConfigSave       = Command{Cmd: "/config.save", Help: "$cmd\t Write current config to disk.\n"}
	CommandModeQuorum       = Command{Cmd: "/mode.quorum", Help: "$cmd\t Switch to quorum mode(default).\n"}
	CommandModeBlocks       = Command{Cmd: "/mode.blocks", Help: "$cmd\t Switch to blocks mode.\n"}
	CommandModeNetwork      = Command{Cmd: "/mode.network", Help: "$cmd\t Switch to network mode.\n"}

	// mode quorum only
	CommandQuorumSend = Command{Cmd: "/send", Help: "$cmd xxxx\t Send message to current group.(quorum mode only)\n"}
	CommandQuorumNick = Command{Cmd: "/nick", Help: "$cmd nickname\t Change nickname of current group.(quorum mode only)\n"}

	// mode blocks only
	CommandBlocksJmp = Command{Cmd: "/blocks.jmp", Help: "$cmd <block_num>\t Jump to block (blocks mode only).\n"}

	// mode network only
	CommandNetworkPing = Command{Cmd: "/network.ping", Help: "$cmd \t Ping connected peers(network mode only).\n"}
)

var BaseCommands = []Command{
	CommandConnect,
	CommandBackup,
	CommandJoin,
	CommandTokenApply,
	CommandGroupSync,
	CommandGroupSeed,
	CommandGroupCreate,
	CommandGroupLeave,
	CommandGroupDelete,
	CommandGroupAdmin,
	CommandGroupChainConfig,
	CommandConfigReload,
	CommandConfigSave,
	CommandModeBlocks,
	CommandModeQuorum,
	CommandModeNetwork,
}

var QuorumCommands = []Command{
	CommandQuorumSend,
	CommandQuorumNick,
}

var BlocksCommands = []Command{
	CommandBlocksJmp,
}

var NetworkCommands = []Command{
	CommandNetworkPing,
}
