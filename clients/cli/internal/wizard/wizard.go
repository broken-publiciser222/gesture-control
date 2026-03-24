package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"cli/internal/config"
)

// Run запускает интерактивный wizard и записывает результат в переданный cfg.
// Если пользователь прерывает wizard (Ctrl+C), cfg остаётся без изменений
// и возвращается ErrAborted.
func Run(cfg *config.Config) error {
	prefill := resultFromConfig(cfg)
	m := initialModel(prefill)

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard: %w", err)
	}

	wm, ok := finalModel.(model)
	if !ok {
		return fmt.Errorf("wizard: unexpected model type")
	}
	if wm.aborted {
		return ErrAborted
	}

	applyResult(cfg, wm.finalResult)
	return nil
}

// ErrAborted возвращается из Run, если пользователь прервал wizard через Ctrl+C.
var ErrAborted = fmt.Errorf("wizard aborted by user")

// resultFromConfig конвертирует config.Config в result для предзаполнения полей.
func resultFromConfig(cfg *config.Config) result {
	if cfg == nil {
		return result{}
	}
	return result{
		APIKey:           cfg.APIKey,
		FFmpegPath:       cfg.FFmpeg.Path,
		ReconnectRetries: cfg.Stream.ReconnectRetries,
		ReconnectDelayMS: cfg.Stream.ReconnectDelayMS,
		Resolution:       cfg.FFmpeg.Resolution,
		FPS:              cfg.FFmpeg.FPS,
		Grayscale:        cfg.FFmpeg.Grayscale,
	}
}

// applyResult записывает собранные wizard'ом значения обратно в config.Config.
func applyResult(cfg *config.Config, r result) {
	cfg.APIKey = r.APIKey
	cfg.FFmpeg.Path = r.FFmpegPath
	cfg.FFmpeg.Resolution = r.Resolution
	cfg.FFmpeg.FPS = r.FPS
	cfg.FFmpeg.Grayscale = r.Grayscale
	cfg.Stream.ReconnectRetries = r.ReconnectRetries
	cfg.Stream.ReconnectDelayMS = r.ReconnectDelayMS
}

// model - основная структура состояния wizard'а.
type model struct {
	steps       []step
	currentStep int
	// focusedInput - индекс активного поля внутри текущего шага.
	focusedInput int
	// validationErr - ошибка валидации текущего шага, показывается под полями.
	validationErr string
	// aborted - true если пользователь нажал Ctrl+C.
	aborted bool
	// done - true если wizard завершён успешно.
	done bool
	// finalResult - заполненные значения, доступны после done=true.
	finalResult result
}

func initialModel(prefill result) model {
	return model{
		steps: buildSteps(prefill),
	}
}

// currentInputs возвращает inputs текущего шага (nil для экрана подтверждения).
func (m *model) currentInputs() []textinput.Model {
	return m.steps[m.currentStep].inputs
}

// focusInput переключает фокус на указанный input внутри текущего шага.
func (m *model) focusInput(idx int) {
	inputs := m.currentInputs()
	for i := range inputs {
		if i == idx {
			inputs[i].Focus()
		} else {
			inputs[i].Blur()
		}
	}
	m.steps[m.currentStep].inputs = inputs
	m.focusedInput = idx
}

// validateCurrent запускает валидацию текущего шага.
// Возвращает true если шаг прошёл валидацию.
func (m *model) validateCurrent() bool {
	s := m.steps[m.currentStep]
	if s.validate == nil {
		return true
	}
	msg := s.validate(s.inputs)
	m.validationErr = msg
	return msg == ""
}

// --- tea.Model interface ---

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {

		// Ctrl+C - прервать без сохранения
		case tea.KeyCtrlC:
			m.aborted = true
			return m, tea.Quit

		// Tab / Shift+Tab - переключение между полями внутри шага
		case tea.KeyTab:
			inputs := m.currentInputs()
			if len(inputs) > 1 {
				next := (m.focusedInput + 1) % len(inputs)
				m.focusInput(next)
			}
			return m, textinput.Blink

		case tea.KeyShiftTab:
			inputs := m.currentInputs()
			if len(inputs) > 1 {
				prev := (m.focusedInput - 1 + len(inputs)) % len(inputs)
				m.focusInput(prev)
			}
			return m, textinput.Blink

		// Enter - подтвердить шаг и перейти к следующему
		case tea.KeyEnter:
			// Экран подтверждения - финальный шаг
			if m.steps[m.currentStep].id == stepConfirm {
				res, err := extractResult(m.steps)
				if err != nil {
					m.validationErr = err.Error()
					return m, nil
				}
				m.finalResult = res
				m.done = true
				return m, tea.Quit
			}

			if !m.validateCurrent() {
				return m, nil
			}

			m.validationErr = ""
			m.currentStep++
			m.focusedInput = 0

			// Фокус на первый input нового шага
			if len(m.steps[m.currentStep].inputs) > 0 {
				m.focusInput(0)
			}
			return m, textinput.Blink

		// Escape - вернуться на шаг назад
		case tea.KeyEsc:
			if m.currentStep > 0 {
				m.currentStep--
				m.validationErr = ""
				m.focusedInput = 0
				if len(m.steps[m.currentStep].inputs) > 0 {
					m.focusInput(0)
				}
			}
			return m, textinput.Blink
		}
	}

	// Передаём нажатия клавиш в активный input текущего шага
	inputs := m.currentInputs()
	if len(inputs) > 0 && m.focusedInput < len(inputs) {
		var cmd tea.Cmd
		inputs[m.focusedInput], cmd = inputs[m.focusedInput].Update(msg)
		m.steps[m.currentStep].inputs = inputs
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.done || m.aborted {
		return ""
	}

	var b strings.Builder

	// Заголовок
	b.WriteString(titleStyle.Render("⚡ gesture-control setup"))
	b.WriteString("\n")

	// Прогресс: "Step 2 of 4" (не считаем шаг подтверждения)
	totalInputSteps := len(m.steps) - 1
	if m.steps[m.currentStep].id != stepConfirm {
		b.WriteString(progressStyle.Render(
			fmt.Sprintf("Step %d of %d", m.currentStep+1, totalInputSteps),
		))
		b.WriteString("\n")
	}

	s := m.steps[m.currentStep]

	// Заголовок и описание шага
	b.WriteString(stepTitleStyle.Render(s.title))
	b.WriteString("\n")
	b.WriteString(descriptionStyle.Render(s.description))
	b.WriteString("\n")

	// Контент шага
	switch s.id {
	case stepAPIKey:
		b.WriteString(renderSingleInput(s.inputs[0]))

	case stepFFmpeg:
		b.WriteString(renderSingleInput(s.inputs[0]))

	case stepStream:
		b.WriteString(renderLabeledInput("Reconnect retries", s.inputs[idxReconnectRetries], m.focusedInput == idxReconnectRetries))
		b.WriteString("\n")
		b.WriteString(renderLabeledInput("Reconnect delay (ms)", s.inputs[idxReconnectDelay], m.focusedInput == idxReconnectDelay))

	case stepVideo:
		b.WriteString(renderLabeledInput("Resolution (WxH)", s.inputs[idxResolution], m.focusedInput == idxResolution))
		b.WriteString("\n")
		b.WriteString(renderLabeledInput("FPS", s.inputs[idxFPS], m.focusedInput == idxFPS))
		b.WriteString("\n")
		b.WriteString(renderLabeledInput("Grayscale", s.inputs[idxGrayscale], m.focusedInput == idxGrayscale))

	case stepConfirm:
		b.WriteString(renderSummary(m.steps))
	}

	// Ошибка валидации
	if m.validationErr != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("✗ " + m.validationErr))
	}

	// Подсказки
	b.WriteString("\n")
	if s.id == stepConfirm {
		b.WriteString(helpStyle.Render("enter: save • esc: back • ctrl+c: cancel"))
	} else if m.currentStep == 0 {
		b.WriteString(helpStyle.Render("enter: next • tab: switch field • ctrl+c: cancel"))
	} else {
		b.WriteString(helpStyle.Render("enter: next • tab: switch field • esc: back • ctrl+c: cancel"))
	}

	return b.String()
}

// --- Вспомогательные функции рендеринга ---

func renderSingleInput(input textinput.Model) string {
	if input.Focused() {
		return activeInputStyle.Render(input.View())
	}
	return inactiveInputStyle.Render(input.View())
}

func renderLabeledInput(label string, input textinput.Model, focused bool) string {
	style := inactiveInputStyle
	if focused {
		style = activeInputStyle
	}
	return fmt.Sprintf("%s\n%s", descriptionStyle.Render(label), style.Render(input.View()))
}

// renderSummary формирует итоговый экран с введёнными значениями.
func renderSummary(steps []step) string {
	apiKey := steps[stepAPIKey].inputs[0].Value()
	masked := maskAPIKey(apiKey)

	rows := []struct{ key, val string }{
		{"API Key", masked},
		{"FFmpeg Path", steps[stepFFmpeg].inputs[0].Value()},
		{"Reconnect Retries", steps[stepStream].inputs[idxReconnectRetries].Value()},
		{"Reconnect Delay (ms)", steps[stepStream].inputs[idxReconnectDelay].Value()},
		{"Resolution", steps[stepVideo].inputs[idxResolution].Value()},
		{"FPS", steps[stepVideo].inputs[idxFPS].Value()},
		{"Grayscale", steps[stepVideo].inputs[idxGrayscale].Value()},
	}

	var inner strings.Builder
	for _, r := range rows {
		inner.WriteString(
			summaryKeyStyle.Render(r.key) +
				summaryValueStyle.Render(r.val) + "\n",
		)
	}

	return summaryBoxStyle.Render(inner.String())
}

// maskAPIKey скрывает большую часть ключа, оставляя видимыми первые и последние 4 символа.
// Например: "sk-abcdef1234" → "sk-a••••••••1234"
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:4] + strings.Repeat("•", len(key)-8) + key[len(key)-4:]
}
