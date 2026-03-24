package wizard

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// stepID идентифицирует конкретный шаг wizard'а.
type stepID int

const (
	stepAPIKey stepID = iota
	stepFFmpeg
	stepStream
	stepVideo
	stepConfirm
)

// step описывает один шаг wizard'а.
type step struct {
	id          stepID
	title       string
	description string
	inputs      []textinput.Model
	// validate вызывается перед переходом на следующий шаг.
	// Возвращает текст ошибки или пустую строку.
	validate func(inputs []textinput.Model) string
}

// result хранит итоговые значения, собранные wizard'ом.
// Используется для записи в config.Config после завершения.
type result struct {
	APIKey           string
	FFmpegPath       string
	ReconnectRetries int
	ReconnectDelayMS int
	Resolution       string
	FPS              int
	Grayscale        bool
}

// inputIndex - именованные индексы полей внутри шагов с несколькими inputs.
const (
	idxReconnectRetries = 0
	idxReconnectDelay   = 1

	idxResolution = 0
	idxFPS        = 1
	idxGrayscale  = 2
)

// newInput создаёт textinput с общими настройками.
func newInput(placeholder, defaultVal string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(defaultVal)
	ti.Width = width
	return ti
}

// buildSteps собирает все шаги wizard'а с начальными значениями из prefill.
// prefill позволяет предзаполнить поля при повторном запуске wizard'а.
func buildSteps(prefill result) []step {
	// --- Шаг 1: API ключ ---
	apiInput := newInput("sk-...", prefill.APIKey, 60)
	apiInput.EchoMode = textinput.EchoPassword
	apiInput.EchoCharacter = '•'
	apiInput.Focus()

	// --- Шаг 2: FFmpeg ---
	ffmpegInput := newInput("/usr/bin/ffmpeg", prefill.FFmpegPath, 60)

	// --- Шаг 3: Настройки стрима ---
	retriesDefault := "3"
	if prefill.ReconnectRetries > 0 {
		retriesDefault = strconv.Itoa(prefill.ReconnectRetries)
	}
	delayDefault := "1000"
	if prefill.ReconnectDelayMS > 0 {
		delayDefault = strconv.Itoa(prefill.ReconnectDelayMS)
	}
	retriesInput := newInput("3", retriesDefault, 10)
	delayInput := newInput("1000", delayDefault, 10)

	// --- Шаг 4: Настройки видео ---
	resDefault := "640x480"
	if prefill.Resolution != "" {
		resDefault = prefill.Resolution
	}
	fpsDefault := "15"
	if prefill.FPS > 0 {
		fpsDefault = strconv.Itoa(prefill.FPS)
	}
	grayscaleDefault := "false"
	if prefill.Grayscale {
		grayscaleDefault = "true"
	}
	resInput := newInput("640x480", resDefault, 20)
	fpsInput := newInput("15", fpsDefault, 10)
	grayscaleInput := newInput("true/false", grayscaleDefault, 10)

	return []step{
		{
			id:          stepAPIKey,
			title:       "API Key",
			description: "Enter your API key. It will be stored locally in the config file.",
			inputs:      []textinput.Model{apiInput},
			validate: func(inputs []textinput.Model) string {
				val := strings.TrimSpace(inputs[0].Value())
				if val == "" {
					return "API key cannot be empty"
				}
				if len(val) < 8 {
					return "API key looks too short"
				}
				return ""
			},
		},
		{
			id:          stepFFmpeg,
			title:       "FFmpeg Path",
			description: "Path to the ffmpeg binary. Leave as-is if correct, or enter a custom path.",
			inputs:      []textinput.Model{ffmpegInput},
			validate: func(inputs []textinput.Model) string {
				val := strings.TrimSpace(inputs[0].Value())
				if val == "" {
					return "FFmpeg path cannot be empty. Run 'gesture-control setup ffmpeg' to auto-detect."
				}
				return ""
			},
		},
		{
			id:          stepStream,
			title:       "Stream Settings",
			description: "Configure reconnect behaviour when the stream connection is lost.",
			inputs:      []textinput.Model{retriesInput, delayInput},
			validate: func(inputs []textinput.Model) string {
				retries, err := strconv.Atoi(strings.TrimSpace(inputs[idxReconnectRetries].Value()))
				if err != nil || retries < 0 {
					return "Reconnect retries must be a non-negative integer"
				}
				delay, err := strconv.Atoi(strings.TrimSpace(inputs[idxReconnectDelay].Value()))
				if err != nil || delay < 0 {
					return "Reconnect delay must be a non-negative integer (milliseconds)"
				}
				return ""
			},
		},
		{
			id:          stepVideo,
			title:       "Video Quality",
			description: "Set capture resolution, frame rate, and whether to use grayscale.",
			inputs:      []textinput.Model{resInput, fpsInput, grayscaleInput},
			validate: func(inputs []textinput.Model) string {
				res := strings.TrimSpace(inputs[idxResolution].Value())
				parts := strings.SplitN(res, "x", 2)
				if len(parts) != 2 {
					return "Resolution must be in WxH format, e.g. 640x480"
				}
				for _, p := range parts {
					if n, err := strconv.Atoi(p); err != nil || n <= 0 {
						return "Resolution dimensions must be positive integers"
					}
				}

				fps, err := strconv.Atoi(strings.TrimSpace(inputs[idxFPS].Value()))
				if err != nil || fps <= 0 {
					return "FPS must be a positive integer"
				}

				gs := strings.TrimSpace(strings.ToLower(inputs[idxGrayscale].Value()))
				if gs != "true" && gs != "false" {
					return "Grayscale must be 'true' or 'false'"
				}

				return ""
			},
		},
		{
			id:          stepConfirm,
			title:       "Review & Confirm",
			description: "Review your settings below. Press Enter to save, Ctrl+C to cancel without saving.",
		},
	}
}

// extractResult читает введённые значения из шагов и возвращает заполненный result.
func extractResult(steps []step) (result, error) {
	retries, err := strconv.Atoi(strings.TrimSpace(steps[stepStream].inputs[idxReconnectRetries].Value()))
	if err != nil {
		return result{}, fmt.Errorf("invalid retries value")
	}
	delay, err := strconv.Atoi(strings.TrimSpace(steps[stepStream].inputs[idxReconnectDelay].Value()))
	if err != nil {
		return result{}, fmt.Errorf("invalid delay value")
	}
	fps, err := strconv.Atoi(strings.TrimSpace(steps[stepVideo].inputs[idxFPS].Value()))
	if err != nil {
		return result{}, fmt.Errorf("invalid fps value")
	}
	grayscale := strings.ToLower(strings.TrimSpace(steps[stepVideo].inputs[idxGrayscale].Value())) == "true"

	return result{
		APIKey:           strings.TrimSpace(steps[stepAPIKey].inputs[0].Value()),
		FFmpegPath:       strings.TrimSpace(steps[stepFFmpeg].inputs[0].Value()),
		ReconnectRetries: retries,
		ReconnectDelayMS: delay,
		Resolution:       strings.TrimSpace(steps[stepVideo].inputs[idxResolution].Value()),
		FPS:              fps,
		Grayscale:        grayscale,
	}, nil
}
