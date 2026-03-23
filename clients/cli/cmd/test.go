package cmd

import (
	"fmt"

	appconfig "cli/internal/config"
	"cli/internal/gateway"
	"cli/internal/stream"

	"github.com/spf13/cobra"
)

// testCmd - корневая команда `test`.
// Последовательно запускает все три проверки: соединение, стрим и задержку.
// Если хотя бы одна из них завершается с ошибкой, выполнение прерывается.
// Пример: cli test
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Проверить соединение, стрим и задержку",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runConnectionTest(); err != nil {
			return err
		}
		if err := runStreamTest(); err != nil {
			return err
		}
		if err := runLatencyTest(); err != nil {
			return err
		}

		fmt.Println("─────────────────────────────────")
		fmt.Println("Overall:     READY")
		return nil
	},
}

// testConnectionCmd - подкоманда `test connection`.
// Запускает только проверку соединения и аутентификации.
// Пример: cli test connection
var testConnectionCmd = &cobra.Command{
	Use:  "connection",
	RunE: func(cmd *cobra.Command, args []string) error { return runConnectionTest() },
}

// testStreamCmd - подкоманда `test stream`.
// Запускает только проверку стрима (FPS и битрейт).
// Пример: cli test stream
var testStreamCmd = &cobra.Command{
	Use:  "stream",
	RunE: func(cmd *cobra.Command, args []string) error { return runStreamTest() },
}

// testLatencyCmd - подкоманда `test latency`.
// Запускает только замер задержки WSS-соединения.
// Пример: cli test latency
var testLatencyCmd = &cobra.Command{
	Use:  "latency",
	RunE: func(cmd *cobra.Command, args []string) error { return runLatencyTest() },
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(testConnectionCmd, testStreamCmd, testLatencyCmd)
}

// runConnectionTest проверяет HTTP-доступность шлюза и WSS-аутентификацию.
//
// Шаги:
//  1. Загружает конфигурацию приложения.
//  2. Выполняет HTTP-пробу шлюза (latency, версия API).
//  3. Выполняет WSS-хендшейк с проверкой API-ключа.
//
// Возвращает ошибку с кодом exitGateway, если любой из шагов завершился неудачно.
func runConnectionTest() error {
	cfg, _ := appconfig.Load()

	probe, err := gateway.ProbeHTTP(cfg.GatewayURL)
	if err != nil {
		return wrapCLIError(exitGateway, "connection probe failed: %v", err)
	}

	handshake, err := gateway.Handshake(cfg.GatewayURL, cfg.APIKey)
	if err != nil {
		return wrapCLIError(exitGateway, "wss handshake failed: %v", err)
	}

	fmt.Printf("Connection   ✓  %s\n", probe.Latency.Round(1e6))
	fmt.Printf("Auth         ✓  API key valid (%s)\n", handshake)
	fmt.Printf("API          ✓  %s\n", probe.APIVersion)
	return nil
}

// runStreamTest проверяет параметры стрима: FPS и битрейт.
// Сравнивает реальный FPS с ожидаемым из конфига (cfg.FFmpeg.FPS).
// Если реальный FPS ниже 80% от целевого - выводит предупреждение "!" вместо "✓".
// Ошибку не возвращает: деградация стрима считается некритичной.
func runStreamTest() error {
	cfg, _ := appconfig.Load()

	metrics, status := stream.TestMetrics(cfg.FFmpeg.FPS), "✓"
	if metrics.FPS < float64(cfg.FFmpeg.FPS)*0.8 {
		status = "!"
	}

	fmt.Printf("Stream       %s  %.1ffps / %.1f Mbps\n", status, metrics.FPS, metrics.BitrateMbps)
	return nil
}

// runLatencyTest замеряет статистику задержки WSS-соединения.
// Выводит в stdout: минимальную, среднюю и максимальную задержку,
// джиттер и процент потерь пакетов.
// Всегда возвращает nil: ошибки замера не являются фатальными.
func runLatencyTest() error {
	min, avg, max, jitter, loss := gateway.LatencyStats()

	fmt.Printf("WSS Latency  ✓  min %s / avg %s / max %s, jitter %s, loss %.1f%%\n", min, avg, max, jitter, loss)
	return nil
}
