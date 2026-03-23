package cmd

import (
	"fmt"
	"os"

	appconfig "cli/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd - корневая команда CLI.
// Все подкоманды (config, start, и др.) регистрируются через AddCommand в своих init().
var rootCmd = &cobra.Command{
	Use:          "gesture-control",
	Short:        "CLI для управления жестами и потоковой передачи видео",
	SilenceUsage: true,
}

// Execute - точка входа CLI, вызывается из main.
// При ошибке завершает процесс с соответствующим exit-кодом через exitCode.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCode(err))
	}
}

func init() {
	cobra.OnInitialize(func() {
		if err := appconfig.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "config init warning: %v\n", err)
		}
	})

	rootCmd.PersistentFlags().String("log-level", "", "Override log level (debug, info, warn, error)")

	_ = viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
}
