package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cli/internal/ffmpeg"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Запустить управление с помощью жестов",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := ffmpeg.Find(); err != nil {
			return err
		}

		fmt.Println("Утилита запущена...")
		return nil
	},
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(startCmd)
}
