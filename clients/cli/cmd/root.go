package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "gesture-control",
	Short: "Управляйте своим компьютером с помощью жестов",
	Run:   printHello,
}

func printHello(cmd *cobra.Command, args []string) {
	fmt.Println("Hello, 世界!")
}
