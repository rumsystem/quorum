package cmd

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup rum data",
	Run: func(cmd *cobra.Command, args []string) {
		params := handlers.BackupParam{
			Peername:     peerName,
			Password:     keystorePassword,
			ConfigDir:    configDir,
			KeystoreDir:  keystoreDir,
			KeystoreName: keystoreName,
			DataDir:      dataDir,
			SeedDir:      seedDir,
			BackupFile:   backupFile,
		}
		if isWasm {
			handlers.BackupForWasm(params)
		} else {
			handlers.Backup(params)
		}
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	flags := backupCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&peerName, "peername", "peer", "peer name")
	flags.StringVar(&configDir, "configdir", "config", "config dir")
	flags.StringVar(&keystoreDir, "keystoredir", "keystore", "keystore dir")
	flags.StringVar(&keystoreName, "keystorename", "defaultkeystore", "keystore name")
	flags.StringVar(&keystorePassword, "keystorepass", "", "keystore password")

	flags.StringVar(&dataDir, "datadir", "datadir", "data dir")
	flags.StringVar(&seedDir, "seeddir", "seeddir", "seed dir")
	flags.StringVar(&backupFile, "file", "", "backup filename")

	flags.BoolVar(&isWasm, "wasm", false, "is wasm")

	backupCmd.MarkFlagRequired("file")
}
