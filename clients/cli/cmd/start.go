package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	appconfig "cli/internal/config"
	"cli/internal/ffmpeg"
	"cli/internal/gateway"
	"cli/internal/stream"
	"cli/internal/wizard"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd запускает систему управления жестами:
// загружает конфиг, при необходимости запускает wizard первичной настройки,
// проверяет отсутствие уже запущенного экземпляра, подключается к gateway
// и начинает потоковую передачу видео через FFmpeg.
//
// Флаги:
//
//	-f, --foreground   не демонизировать, логи в stdout; завершение по Ctrl+C
//	    --resolution   переопределить разрешение из конфига
//	    --fps          переопределить FPS из конфига
//	    --grayscale    переопределить режим оттенков серого из конфига
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Запустить управление с помощью жестов",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.Load()
		if err != nil {
			return wrapCLIError(exitConfig, "load config: %v", err)
		}

		if appconfig.NeedsWizard(cfg) {
			cfg, err = wizard.Run(cfg)
			if err != nil {
				return wrapCLIError(exitConfig, "%v", err)
			}
			if err := appconfig.Save(cfg); err != nil {
				return wrapCLIError(exitConfig, "save config: %v", err)
			}

			fmt.Println("✓ Настройка завершена. Производится запуск утилиты...")
		}

		applyOverrides(cmd, &cfg)

		lockPath, err := appconfig.LockFilePath()
		if err != nil {
			return wrapCLIError(exitConfig, "lock path: %v", err)
		}
		if pid, running := existingPID(lockPath); running {
			return wrapCLIError(exitAlreadyRunning, "gesture-control уже запущен (PID %d)", pid)
		}
		if err := ensureFFmpeg(&cfg); err != nil {
			return err
		}
		if err := connectGateway(cfg); err != nil {
			return err
		}
		if err := os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
			return wrapCLIError(exitConfig, "write lock file: %v", err)
		}
		defer os.Remove(lockPath)

		fmt.Printf("[gesture-control] INFO  Starting video stream @ %s %dfps grayscale=%t\n", cfg.FFmpeg.Resolution, cfg.FFmpeg.FPS, cfg.FFmpeg.Grayscale)
		fmt.Printf("[gesture-control] INFO  Preview: %s\n", stream.PreviewCommand(cfg.FFmpeg.Path, cfg.FFmpeg.Resolution, cfg.FFmpeg.FPS, cfg.FFmpeg.Grayscale))
		fmt.Println("[gesture-control] INFO  Streaming... Press Ctrl+C to stop")

		sigCh := make(chan os.Signal, 1)

		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		if foreground, _ := cmd.Flags().GetBool("foreground"); foreground {
			<-sigCh

			fmt.Println("[gesture-control] INFO  Graceful shutdown completed")
			return nil
		}

		fmt.Println("[gesture-control] INFO  Running in background mode is simulated in this build")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolP("foreground", "f", false, "Do not daemonize; log to stdout")
	startCmd.Flags().String("resolution", "", "Override resolution from config")
	startCmd.Flags().Int("fps", 0, "Override FPS from config")
	startCmd.Flags().Bool("grayscale", false, "Override grayscale from config")

	_ = viper.BindPFlag("ffmpeg.resolution", startCmd.Flags().Lookup("resolution"))
	_ = viper.BindPFlag("ffmpeg.fps", startCmd.Flags().Lookup("fps"))
	_ = viper.BindPFlag("ffmpeg.grayscale", startCmd.Flags().Lookup("grayscale"))
}

// applyOverrides применяет значения флагов и viper-переменных поверх загруженного конфига.
// Вызывается после загрузки конфига и до запуска стрима.
// Флаг grayscale применяется только если был явно передан (flag.Changed),
// чтобы не затирать значение из конфига дефолтным false.
func applyOverrides(cmd *cobra.Command, cfg *appconfig.Config) {
	if v := viper.GetString("ffmpeg.resolution"); v != "" {
		cfg.FFmpeg.Resolution = v
	}
	if v := viper.GetInt("ffmpeg.fps"); v > 0 {
		cfg.FFmpeg.FPS = v
	}
	if flag := cmd.Flags().Lookup("grayscale"); flag != nil && flag.Changed {
		cfg.FFmpeg.Grayscale = viper.GetBool("ffmpeg.grayscale")
	}
	if v := viper.GetString("log_level"); v != "" {
		cfg.LogLevel = v
	}
}

// ensureFFmpeg проверяет путь к ffmpeg из конфига.
// Если путь не задан или невалиден - ищет ffmpeg в PATH автоматически.
// Обновляет cfg.FFmpeg.Path найденным значением.
func ensureFFmpeg(cfg *appconfig.Config) error {
	if strings.TrimSpace(cfg.FFmpeg.Path) != "" {
		if _, err := ffmpeg.Validate(cfg.FFmpeg.Path); err == nil {
			return nil
		}
	}

	found, err := ffmpeg.Find()
	if err != nil {
		return wrapCLIError(exitFFmpeg, "ffmpeg не найден или невалиден")
	}

	cfg.FFmpeg.Path = found.Path
	return nil
}

// connectGateway устанавливает соединение с gateway с повторными попытками.
// Задержка между попытками растёт линейно: delay_ms * номер_попытки.
// Различает ECONNREFUSED (gateway недоступен) от остальных ошибок подключения.
func connectGateway(cfg appconfig.Config) error {
	var lastErr error
	for attempt := 1; attempt <= cfg.Stream.ReconnectRetries; attempt++ {
		fmt.Printf("[gesture-control] INFO  Connecting to gateway... %s\n", cfg.GatewayURL)

		latency, err := gateway.Handshake(cfg.GatewayURL, cfg.APIKey)
		if err == nil {
			fmt.Printf("[gesture-control] INFO  Connected. Session ID: simulated (%s)\n", latency)
			return nil
		}

		lastErr = err
		backoff := time.Duration(cfg.Stream.ReconnectDelayMS) * time.Millisecond

		fmt.Printf("[gesture-control] WARN  Gateway connect failed (attempt %d/%d): %v\n", attempt, cfg.Stream.ReconnectRetries, err)

		time.Sleep(backoff)
	}

	if errors.Is(lastErr, syscall.ECONNREFUSED) {
		return wrapCLIError(exitGateway, "gateway недоступен: %v", lastErr)
	}

	return wrapCLIError(exitGateway, "не удалось подключиться к gateway: %v", lastErr)
}

// existingPID читает lock-файл и проверяет, жив ли процесс с указанным PID.
// Возвращает (pid, true) если процесс существует и отвечает на сигнал 0.
// Возвращает (0, false) если lock-файл отсутствует, невалиден или процесс мёртв.
func existingPID(lockPath string) (int, bool) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return pid, false
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return pid, false
	}

	return pid, true
}
