package cmd

import (
	"time"

	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	// create
	jwtType        string // node or chain
	jwtName        string
	jwtAllowGroups []string
	jwtDuration    time.Duration

	// parse
	jwtToken string
)

// jwtCmd represents the jwt command
var jwtCmd = &cobra.Command{
	Use:   "jwt",
	Short: "A jwt tool, create or parse jwt",
}

var jwtCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create jwt and save to config file",
}

var jwtCreateNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Create jwt for node sdk and save to config file",
	Run: func(cmd *cobra.Command, args []string) {
		createNodeToken(configDir, peerName, jwtName, jwtDuration, jwtAllowGroups)
	},
}

var jwtCreateChainCmd = &cobra.Command{
	Use:   "chain",
	Short: "Create jwt for chain sdk and save to config file",
	Run: func(cmd *cobra.Command, args []string) {
		createChainToken(configDir, peerName, jwtName, jwtDuration)
	},
}

var jwtParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse jwt",
	Run: func(cmd *cobra.Command, args []string) {
		parseToken(configDir, peerName, jwtToken)
	},
}

func init() {
	jwtCreateCmd.AddCommand(jwtCreateNodeCmd)
	jwtCreateCmd.AddCommand(jwtCreateChainCmd)

	jwtCmd.AddCommand(jwtCreateCmd)
	jwtCmd.AddCommand(jwtParseCmd)

	rootCmd.AddCommand(jwtCmd)

	// create node jwt
	createNodeFlags := jwtCreateNodeCmd.Flags()
	createNodeFlags.SortFlags = false
	createNodeFlags.StringVarP(&configDir, "configdir", "c", "config", "config directory")
	createNodeFlags.StringVarP(&peerName, "peername", "p", "peer", "peer name")
	createNodeFlags.StringVarP(&jwtName, "name", "n", "", "name of the node jwt")
	createNodeFlags.DurationVarP(&jwtDuration, "duration", "d", time.Hour*24*365, "duration of node jwt")
	createNodeFlags.StringArrayVarP(&jwtAllowGroups, "allow_group", "a", []string{}, "allow groups for node jwt")

	jwtCreateNodeCmd.MarkFlagRequired("name")
	jwtCreateNodeCmd.MarkFlagRequired("allow_group")

	// create chain jwt
	createChainFlags := jwtCreateChainCmd.Flags()
	createChainFlags.SortFlags = false
	createChainFlags.StringVarP(&configDir, "configdir", "c", "config", "config directory")
	createChainFlags.StringVarP(&peerName, "peername", "p", "peer", "peer name")
	createChainFlags.StringVarP(&jwtName, "name", "n", "", "name of the node jwt")
	createChainFlags.DurationVarP(&jwtDuration, "duration", "d", time.Hour*24*365, "duration of node jwt")

	jwtCreateChainCmd.MarkFlagRequired("name")

	// parse jwt
	parseFlags := jwtParseCmd.Flags()
	parseFlags.SortFlags = false
	parseFlags.StringVarP(&configDir, "configdir", "c", "config", "config directory")
	parseFlags.StringVarP(&peerName, "peername", "p", "peer", "peer name")
	parseFlags.StringVarP(&jwtToken, "token", "t", "", "jwt token")

	jwtParseCmd.MarkFlagRequired("token")
}

// getJWTKey get jwt key or fatal
func getJWTKey(configDir string, peerName string) string {
	opt, err := options.InitNodeOptions(configDir, peerName)
	if err != nil {
		logger.Fatalf("get jwt key failed: %s", err)
	}
	return opt.JWTKey
}

func newToken(name string, role string, groups []string, key string, exp time.Time) string {
	_tokenStr, err := utils.NewJWTToken(name, role, groups, key, exp)
	if err != nil {
		logger.Fatalf("create token failed: %s", err)
	}
	return _tokenStr
}

func saveToken(name, token, configdir, peername string) {
	nodeoptions, err := options.InitNodeOptions(configdir, peername)
	if err != nil {
		logger.Fatalf("init node option failed: %s", err)
	}
	nodeoptions.SetJWTTokenMap(name, token)
}

func createNodeToken(configDir string, peerName string, name string, duration time.Duration, allowgroups []string) {
	key := getJWTKey(configDir, peerName)
	_tokenStr := newToken(name, "node", allowgroups, key, time.Now().Add(duration))
	logger.Infof("new nodesdk token: %s", _tokenStr)
	saveToken(name, _tokenStr, configDir, peerName)
}

func createChainToken(configDir string, peerName string, name string, duration time.Duration) {
	key := getJWTKey(configDir, peerName)
	_tokenStr := newToken(name, "chain", []string{}, key, time.Now().Add(duration))
	logger.Infof("new chainsdk token: %s", _tokenStr)
	saveToken(name, _tokenStr, configDir, peerName)
}

func parseToken(configDir string, peerName string, token string) {
	key := getJWTKey(configDir, peerName)
	claims, err := utils.ParseJWTToken(token, key)
	if err != nil {
		logger.Fatalf("parse token failed: %s", err)
	}
	logger.Infof("parse token: %+v\n", *claims)
}
