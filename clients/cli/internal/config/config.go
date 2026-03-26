package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// FFmpegConfig хранит настройки FFmpeg: путь к бинарнику и параметры захвата видео.
type FFmpegConfig struct {
	Path       string `mapstructure:"path" yaml:"path"`
	Resolution string `mapstructure:"resolution" yaml:"resolution"`
	FPS        int    `mapstructure:"fps" yaml:"fps"`
	Grayscale  bool   `mapstructure:"grayscale" yaml:"grayscale"`
}

// StreamConfig хранит параметры переподключения к бэкенду при обрыве соединения.
type StreamConfig struct {
	ReconnectRetries int `mapstructure:"reconnect_retries" yaml:"reconnect_retries"`
	ReconnectDelayMS int `mapstructure:"reconnect_delay_ms" yaml:"reconnect_delay_ms"`
}

// Config - корневая конфигурация приложения.
type Config struct {
	APIKey     string       `mapstructure:"api_key" yaml:"api_key"`
	GatewayURL string       `mapstructure:"gateway_url" yaml:"gateway_url"`
	FFmpeg     FFmpegConfig `mapstructure:"ffmpeg" yaml:"ffmpeg"`
	Stream     StreamConfig `mapstructure:"stream" yaml:"stream"`
	LogLevel   string       `mapstructure:"log_level" yaml:"log_level"`
}

// setDefaults регистрирует значения по умолчанию в viper.
// Вызывается при инициализации и при сбросе конфига.
func setDefaults() {
	viper.SetDefault("gateway_url", "wss://api.example.com/ws")
	viper.SetDefault("ffmpeg.resolution", "640x480")
	viper.SetDefault("ffmpeg.fps", 15)
	viper.SetDefault("ffmpeg.grayscale", false)
	viper.SetDefault("stream.reconnect_retries", 3)
	viper.SetDefault("stream.reconnect_delay_ms", 1000)
	viper.SetDefault("log_level", "info")
}

// ConfigDirPath возвращает платформо-зависимый путь к директории конфига:
//   - macOS:   ~/Library/Application Support/gesture-control
//   - Windows: %APPDATA%/gesture-control
//   - Linux:   $XDG_CONFIG_HOME/gesture-control (по умолчанию ~/.config/gesture-control)
func ConfigDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "gesture-control"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "gesture-control"), nil
	default:
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(home, ".config")
		}
		return filepath.Join(base, "gesture-control"), nil
	}
}

// ConfigFilePath возвращает полный путь к файлу конфигурации config.yaml.
func ConfigFilePath() (string, error) {
	appPath, err := ConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(appPath, "config.yaml"), nil
}

// LockFilePath возвращает полный путь к PID-файлу,
// который используется для предотвращения запуска нескольких экземпляров приложения.
func LockFilePath() (string, error) {
	appPath, err := ConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(appPath, "gesture-control.pid"), nil
}

// ensureConfigDir создаёт директорию конфига, если она не существует,
// и возвращает её путь. Вызывается перед любой операцией записи.
func ensureConfigDir() (string, error) {
	dirPath, err := ConfigDirPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dirPath, nil
}

// writeConfig сохраняет текущее состояние viper в файл конфига.
// Если файл не существует - создаёт его через WriteConfigAs,
// иначе обновляет существующий через WriteConfig.
func writeConfig() error {
	if _, err := ensureConfigDir(); err != nil {
		return err
	}
	filePath, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return viper.WriteConfigAs(filePath)
	}
	return viper.WriteConfig()
}

// Init инициализирует viper: регистрирует путь к конфигу, переменные окружения
// и значения по умолчанию, создаёт директорию конфига при необходимости.
// Если файл конфига отсутствует (первый запуск) - ошибкой не считается.
// Переменные окружения имеют приоритет над файлом; префикс - GESTURE_,
// точки в ключах заменяются на подчёркивания (например, GESTURE_FFMPEG_FPS).
func Init() error {
	filePath, err := ConfigFilePath()
	if err != nil {
		return err
	}

	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("gesture")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	if _, err := ensureConfigDir(); err != nil {
		return err
	}
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	return nil
}

// Load десериализует текущее состояние viper в структуру Config.
func Load() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}

// Save записывает переданную конфигурацию в viper и сохраняет её на диск.
func Save(cfg Config) error {
	viper.Set("api_key", cfg.APIKey)
	viper.Set("gateway_url", cfg.GatewayURL)
	viper.Set("ffmpeg.path", cfg.FFmpeg.Path)
	viper.Set("ffmpeg.resolution", cfg.FFmpeg.Resolution)
	viper.Set("ffmpeg.fps", cfg.FFmpeg.FPS)
	viper.Set("ffmpeg.grayscale", cfg.FFmpeg.Grayscale)
	viper.Set("stream.reconnect_retries", cfg.Stream.ReconnectRetries)
	viper.Set("stream.reconnect_delay_ms", cfg.Stream.ReconnectDelayMS)
	viper.Set("log_level", cfg.LogLevel)

	return writeConfig()
}

// SaveCurrent сохраняет текущее состояние viper на диск без изменений.
// Используется когда значения уже установлены через viper.Set напрямую.
func SaveCurrent() error {
	return writeConfig()
}

// Reset сбрасывает конфиг до значений по умолчанию,
// очищая чувствительные поля (api_key, ffmpeg.path), и сохраняет результат на диск.
func Reset() error {
	setDefaults()

	viper.Set("api_key", "")
	viper.Set("ffmpeg.path", "")

	return SaveCurrent()
}

// IsReady возвращает true, если конфиг не содержит API-ключа -
// то есть приложение запускается впервые или ключ был сброшен.
func IsReady(cfg Config) bool {
	return strings.TrimSpace(cfg.APIKey) != ""
}
