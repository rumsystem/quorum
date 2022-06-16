package main

import (
	"os"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/urfave/cli/v2"
)

var (
	logger = logging.Logger("jwt_cmd")
)

func init() {
	logLevel, err := logging.LevelFromString("debug")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(logLevel)
}

func main() {
	app := &cli.App{
		Name:                 "jwt",
		Usage:                "JWT tool, create or parse jwt token",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name: "create",
				// Usage: "Create a jwt token",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "configdir",
						Aliases: []string{"c"},
						Usage:   "config and keys dir",
						Value:   "./config/",
					},
					&cli.StringFlag{
						Name:    "peername",
						Aliases: []string{"p"},
						Usage:   "peer name",
						Value:   "peer",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:   "node",
						Usage:  "Create a jwt token for node api",
						Action: createNodeTokenCmd,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "name for jwt token",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "key",
								Usage:    "jwt key",
								Required: true,
							},
							&cli.StringSliceFlag{
								Name:     "allowgroup",
								Aliases:  []string{"a"},
								Usage:    "Allow group id or '*' for all group ids",
								Required: true,
							},
							&cli.DurationFlag{
								Name:    "duration",
								Aliases: []string{"d"},
								Usage:   "Expiration duration",
								Value:   time.Hour * 24 * 365,
							},
						},
					},
					{
						Name:   "chain",
						Usage:  "Create a jwt token for chain api",
						Action: createChainTokenCmd,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Usage:    "name for jwt token",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "key",
								Usage:    "jwt key",
								Required: true,
							},
							&cli.DurationFlag{
								Name:    "duration",
								Aliases: []string{"d"},
								Usage:   "Expiration duration",
								Value:   time.Hour * 24 * 365,
							},
						},
					},
				},
			},
			{
				Name:   "parse",
				Usage:  "parse jwt token",
				Action: parseTokenCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "token",
						Aliases:  []string{"t"},
						Usage:    "jwt token string",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "key",
						Usage:    "jwt key",
						Required: true,
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
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

func createNodeTokenCmd(c *cli.Context) error {
	name := c.String("name")
	_tokenStr := newToken(name, "node", c.StringSlice("allowgroup"), c.String("key"), time.Now().Add(c.Duration("duration")))
	logger.Infof("new token: %s", _tokenStr)
	saveToken(name, _tokenStr, c.String("configdir"), c.String("peername"))
	return nil
}

func createChainTokenCmd(c *cli.Context) error {
	name := c.String("name")
	_tokenStr := newToken(name, "chain", []string{}, c.String("key"), time.Now().Add(c.Duration("duration")))
	logger.Infof("new token: %s", _tokenStr)
	saveToken(name, _tokenStr, c.String("configdir"), c.String("peername"))
	return nil
}

func parseTokenCmd(c *cli.Context) error {
	claims, err := utils.ParseJWTToken(c.String("token"), c.String("key"))
	if err != nil {
		logger.Fatalf("parse token failed: %s", err)
	}
	logger.Infof("parse token: %+v\n", *claims)
	return nil
}
