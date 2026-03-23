package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	appconfig "cli/internal/config"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Остановить управление с помощью жестов",
	RunE: func(cmd *cobra.Command, args []string) error {
		lockPath, err := appconfig.LockFilePath()
		if err != nil {
			return wrapCLIError(exitConfig, "lock path: %v", err)
		}

		data, err := os.ReadFile(lockPath)
		if err != nil {
			fmt.Println("gesture-control не запущен")
			return nil
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			_ = os.Remove(lockPath)

			fmt.Println("gesture-control не запущен")
			return nil
		}

		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Signal(syscall.SIGTERM)

			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if err := proc.Signal(syscall.Signal(0)); err != nil {
					break
				}
				time.Sleep(250 * time.Millisecond)
			}

			_ = proc.Signal(syscall.SIGKILL)
		}

		_ = os.Remove(lockPath)

		fmt.Println("✓ gesture-control остановлен")
		return nil
	},
}

func init() { rootCmd.AddCommand(stopCmd) }
