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

type FFmpegConfig struct {
	Path       string `mapstructure:"path" yaml:"path"`
	Resolution string `mapstructure:"resolution" yaml:"resolution"`
	FPS        int    `mapstructure:"fps" yaml:"fps"`
	Grayscale  bool   `mapstructure:"grayscale" yaml:"grayscale"`
}

type StreamConfig struct {
	ReconnectRetries int `mapstructure:"reconnect_retries" yaml:"reconnect_retries"`
	ReconnectDelayMS int `mapstructure:"reconnect_delay_ms" yaml:"reconnect_delay_ms"`
}

type Config struct {
	APIKey     string       `mapstructure:"api_key" yaml:"api_key"`
	GatewayURL string       `mapstructure:"gateway_url" yaml:"gateway_url"`
	FFmpeg     FFmpegConfig `mapstructure:"ffmpeg" yaml:"ffmpeg"`
	Stream     StreamConfig `mapstructure:"stream" yaml:"stream"`
	LogLevel   string       `mapstructure:"log_level" yaml:"log_level"`
}

func setDefaults() {
	viper.SetDefault("gateway_url", "wss://api.example.com/ws")
	viper.SetDefault("ffmpeg.resolution", "640x480")
	viper.SetDefault("ffmpeg.fps", 15)
	viper.SetDefault("ffmpeg.grayscale", false)
	viper.SetDefault("stream.reconnect_retries", 3)
	viper.SetDefault("stream.reconnect_delay_ms", 1000)
	viper.SetDefault("log_level", "info")
}

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

func ConfigFilePath() (string, error) {
	appPath, err := ConfigDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(appPath, "config.yaml"), nil
}

func LockFilePath() (string, error) {
	appPath, err := ConfigDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(appPath, "gesture-control.pid"), nil
}

func Init() error {
	cfgFilePath, err := ConfigFilePath()
	if err != nil {
		return err
	}

	viper.SetConfigFile(cfgFilePath)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("gesture")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	cfgDirPath, err := ConfigDirPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfgDirPath, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if _, err := os.Stat(cfgFilePath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}

func Load() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func Save(cfg Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	viper.Set("api_key", cfg.APIKey)
	viper.Set("gateway_url", cfg.GatewayURL)
	viper.Set("ffmpeg.path", cfg.FFmpeg.Path)
	viper.Set("ffmpeg.resolution", cfg.FFmpeg.Resolution)
	viper.Set("ffmpeg.fps", cfg.FFmpeg.FPS)
	viper.Set("ffmpeg.grayscale", cfg.FFmpeg.Grayscale)
	viper.Set("stream.reconnect_retries", cfg.Stream.ReconnectRetries)
	viper.Set("stream.reconnect_delay_ms", cfg.Stream.ReconnectDelayMS)
	viper.Set("log_level", cfg.LogLevel)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return viper.WriteConfigAs(path)
}

func SaveCurrent() error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return viper.WriteConfigAs(path)
	}

	return viper.WriteConfig()
}

func Reset() error {
	setDefaults()

	viper.Set("api_key", "")
	viper.Set("ffmpeg.path", "")

	return SaveCurrent()
}

func NeedsWizard(cfg Config) bool {
	return strings.TrimSpace(cfg.APIKey) == ""
}
