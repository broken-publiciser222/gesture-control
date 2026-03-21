package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// knownPaths - места где ffmpeg часто лежит вне PATH
var knownPaths = map[string][]string{
	"linux": {
		"/usr/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/snap/bin/ffmpeg",
	},
	"darwin": {
		"/opt/homebrew/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/opt/local/bin/ffmpeg",
	},
	"windows": {
		`C:\ffmpeg\bin\ffmpeg.exe`,
		`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `ffmpeg\bin\ffmpeg.exe`),
	},
}

type FindResult struct {
	Path    string
	Version string
}

// Find ищет ffmpeg: сначала PATH, потом known paths
func Find() (*FindResult, error) {
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		if ver, err := Validate(path); err == nil {
			return &FindResult{Path: path, Version: ver}, nil
		}
	}

	for _, path := range knownPaths[runtime.GOOS] {
		if _, err := os.Stat(path); err == nil {
			if ver, err := Validate(path); err == nil {
				return &FindResult{Path: path, Version: ver}, nil
			}
		}
	}

	return nil, fmt.Errorf("ffmpeg не найден")
}

// Validate запускает ffmpeg -version и парсит номер версии
func Validate(path string) (string, error) {
	out, err := exec.Command(path, "-version").Output()
	if err != nil {
		return "", err
	}

	var version string
	fmt.Sscanf(string(out), "версия ffmpeg - %s", &version)

	return version, nil
}
