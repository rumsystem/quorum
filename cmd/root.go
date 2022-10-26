package cmd

import (
	"os"
	"strings"

	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger = logging.Logger("cmd")

	logLevel      string
	logFile       string
	logMaxSize    int // megabytes
	logMaxBackups int
	logMaxAge     int // days
	logCompress   bool

	isDebug bool // true is lower(logLevel) == "debug" else false

	// flags
	peerName         string
	peerList         cli.AddrList
	configDir        string
	keystoreDir      string
	keystoreName     string
	keystorePassword string
	dataDir          string
	seedDir          string
	backupFile       string
	isWasm           bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "quorum",
	Short: "The internet alternatives",
	Long:  `An open source peer-to-peer application infrastructure to offer the internet alternatives in a decentralized and privacy oriented way.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&logLevel, "loglevel", "", "log level")
	rootCmd.PersistentFlags().StringVar(&logFile, "logfile", "", "log file, default output to stdout")
	rootCmd.PersistentFlags().IntVar(&logMaxSize, "log-max-size", 100, "log file max size, unit: megabytes")
	rootCmd.PersistentFlags().IntVar(&logMaxAge, "log-max-age", 7, "log file max ages, unit: day")
	rootCmd.PersistentFlags().IntVar(&logMaxBackups, "log-max-backups", 3, "log file max backups count")
	rootCmd.PersistentFlags().BoolVar(&logCompress, "log-compress", true, "is log file compress")
}

func initConfig() {
	isDebug = strings.ToLower(logLevel) == "debug"

	// set log level
	lvl, err := logging.LevelFromString(logLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logging.SetAllLoggers(lvl)

	if logFile != "" {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    logMaxSize,
			MaxBackups: logMaxBackups,
			MaxAge:     logMaxAge,
			Compress:   logCompress,
		})
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
			w,
			zap.InfoLevel,
		)
		logging.SetPrimaryCore(core)
	}
}
