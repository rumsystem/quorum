package cmd

import (
	"fmt"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	// create
	jwtName     string
	jwtGroupId  string
	jwtDuration time.Duration

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
		createNodeToken(configDir, peerName, jwtName, jwtDuration, jwtGroupId)
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
	createNodeFlags.StringVarP(&jwtGroupId, "groupid", "g", "", "allow group for node jwt")

	jwtCreateNodeCmd.MarkFlagRequired("name")
	jwtCreateNodeCmd.MarkFlagRequired("groupid")

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
	return opt.JWT.Key
}

func newToken(role string, groupid string, name string, duration time.Duration, configdir, peername string) (string, error) {
	nodeoptions, err := options.InitNodeOptions(configdir, peername)
	if err != nil {
		logger.Fatalf("init node option failed: %s", err)
	}

	if role == "node" {
		return nodeoptions.NewNodeJWT(groupid, name, time.Now().Add(duration))
	} else if role == "chain" {
		return nodeoptions.NewChainJWT(name, time.Now().Add(duration))
	} else {
		return "", fmt.Errorf("invalid token role: %s", role)
	}
}

func createNodeToken(configDir string, peerName string, name string, duration time.Duration, groupid string) {
	token, err := newToken("node", groupid, name, duration, configDir, peerName)
	if err != nil {
		logger.Fatalf("create node token failed: %s", err)
	}
	fmt.Printf("new nodesdk token: %s\n", token)
}

func createChainToken(configDir string, peerName string, name string, duration time.Duration) {
	token, err := newToken("chain", "", name, duration, configDir, peerName)
	if err != nil {
		logger.Fatalf("create chain token failed: %s", err)
	}
	fmt.Printf("new chain token: %s\n", token)
}

func parseToken(configDir string, peerName string, token string) {
	key := getJWTKey(configDir, peerName)
	claims, err := utils.ParseJWTToken(token, key)
	if err != nil {
		logger.Fatalf("parse token failed: %s", err)
	}
	fmt.Printf("parse token: %+v\n", *claims)
}
