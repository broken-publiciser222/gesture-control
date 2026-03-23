package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	appconfig "cli/internal/config"
	"cli/internal/wizard"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd - корневая команда группы "config".
// Используется как точка входа для подкоманд управления конфигурацией.
//
// Доступные подкоманды:
//   - show   - отобразить текущий конфиг
//   - get    - получить значение по ключу
//   - set    - установить значение по ключу
//   - reset  - сбросить конфиг к дефолтам
//   - wizard - интерактивная настройка конфига
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Управление конфигом",
}

// configShowCmd выводит все поля текущего конфига в stdout.
// Чувствительные поля (например, api_key) маскируются через maskConfig.
var configShowCmd = &cobra.Command{
	Use: "show",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.Load()
		if err != nil {
			return err
		}

		fmt.Printf("api_key:                   %s\n", maskConfig(cfg.APIKey))
		fmt.Printf("gateway_url:               %s\n", cfg.GatewayURL)
		fmt.Printf("ffmpeg.path:               %s\n", cfg.FFmpeg.Path)
		fmt.Printf("ffmpeg.resolution:         %s\n", cfg.FFmpeg.Resolution)
		fmt.Printf("ffmpeg.fps:                %d\n", cfg.FFmpeg.FPS)
		fmt.Printf("ffmpeg.grayscale:          %t\n", cfg.FFmpeg.Grayscale)
		fmt.Printf("stream.reconnect_retries:  %d\n", cfg.Stream.ReconnectRetries)
		fmt.Printf("stream.reconnect_delay_ms: %d\n", cfg.Stream.ReconnectDelayMS)
		fmt.Printf("log_level:                 %s\n", cfg.LogLevel)
		return nil
	},
}

// configGetCmd выводит значение конкретного ключа конфига.
// Использование: config get <key>
// Пример: config get ffmpeg.fps
var configGetCmd = &cobra.Command{
	Use:  "get <key>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(viper.Get(args[0]))
	},
}

// configSetCmd устанавливает значение ключа и сохраняет конфиг на диск.
// Использование: config set <key> <value>
// Пример: config set log_level debug
var configSetCmd = &cobra.Command{
	Use:  "set <key> <value>",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.Set(args[0], args[1])
		return appconfig.SaveCurrent()
	},
}

// configResetCmd сбрасывает конфиг к дефолтным значениям.
// Перед сбросом запрашивает подтверждение у пользователя.
var configResetCmd = &cobra.Command{
	Use: "reset",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Сбросить конфиг к дефолтам? [y/N]: ")

		resp, _ := reader.ReadString('\n')

		resp = strings.TrimSpace(resp)
		if resp != "y\n" && resp != "Y\n" {
			fmt.Println("Отменено")
			return nil
		}

		return appconfig.Reset()
	},
}

// configWizardCmd запускает интерактивный мастер настройки.
// Загружает текущий конфиг, предлагает изменить поля в диалоговом режиме,
// затем сохраняет результат на диск.
var configWizardCmd = &cobra.Command{
	Use: "wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := appconfig.Load()

		cfg, err := wizard.Run(cfg)
		if err != nil {
			return wrapCLIError(exitConfig, "%v", err)
		}

		if err := appconfig.Save(cfg); err != nil {
			return wrapCLIError(exitConfig, "%v", err)
		}

		fmt.Println("✓ Настройка завершена. Запустите: gesture-control start")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd, configSetCmd, configGetCmd, configResetCmd, configWizardCmd)
}

// maskConfig маскирует строку для безопасного отображения в логах и выводе.
// Строки короче 10 символов заменяются на "****".
// Более длинные строки обрезаются: первые 5 и последние 4 символа сохраняются.
// Пример: "sk-abc123xyz9876" → "sk-ab...9876"
func maskConfig(value string) string {
	if len(value) < 10 {
		return "****"
	}

	return value[:5] + "..." + value[len(value)-4:]
}
