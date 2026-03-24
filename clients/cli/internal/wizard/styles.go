package wizard

import "github.com/charmbracelet/lipgloss"

var (
	// Цвета
	colorPrimary = lipgloss.Color("#7C3AED")
	colorMuted   = lipgloss.Color("#6B7280")
	colorSuccess = lipgloss.Color("#10B981")
	colorError   = lipgloss.Color("#EF4444")
	colorWarning = lipgloss.Color("#F59E0B")

	// Заголовок wizard'а
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	// Подсказка текущего шага (например "Step 2 of 4")
	progressStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginBottom(1)

	// Заголовок текущего шага
	stepTitleStyle = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(1)

	// Описание под заголовком шага
	descriptionStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				MarginBottom(1)

	// Активный input (в фокусе)
	activeInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Неактивный input
	inactiveInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorMuted).
				Padding(0, 1)

	// Сообщение об ошибке валидации
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			MarginTop(1)

	// Сообщение об успехе (например, "FFmpeg found")
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// Предупреждение (например, "FFmpeg not found")
	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	// Подсказки по клавишам внизу экрана
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(2)

	// Итоговый экран подтверждения
	summaryKeyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(24)

	summaryValueStyle = lipgloss.NewStyle().
				Bold(true)

	summaryBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)
)
