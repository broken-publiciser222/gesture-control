package cmd

import (
	"errors"
	"fmt"
)

// Exit-коды процесса. Используются в os.Exit() для сигнализации
// причины завершения родительскому процессу или скриптам.
//
//	0 - успех
//	1 - общая ошибка
//	2 - ошибка конфигурации / wizard прерван
//	3 - ошибка подключения к API Gateway
//	4 - ffmpeg не найден или упал
//	5 - приложение уже запущено
const (
	exitOK             = 0
	exitGeneral        = 1
	exitConfig         = 2
	exitGateway        = 3
	exitFFmpeg         = 4
	exitAlreadyRunning = 5
)

// cliError - ошибка с привязанным exit-кодом.
// Используется для передачи кода завершения из глубины call stack
// до точки вызова os.Exit в main.
type cliError struct {
	Code int
	Err  error
}

// Error реализует интерфейс error.
func (e *cliError) Error() string {
	return e.Err.Error()
}

// Unwrap реализует интерфейс errors.Unwrap,
// позволяя errors.As/errors.Is проходить сквозь обёртку.
func (e *cliError) Unwrap() error { return e.Err }

// wrapCLIError создаёт cliError с заданным exit-кодом и отформатированным сообщением.
// Используется аналогично fmt.Errorf:
//
// return wrapCLIError(exitConfig, "не удалось загрузить конфиг: %v", err)
func wrapCLIError(code int, format string, args ...any) error {
	return &cliError{Code: code, Err: fmt.Errorf(format, args...)}
}

// exitCode извлекает exit-код из ошибки.
// Если err == nil - возвращает exitOK.
// Если err содержит *cliError - возвращает его Code.
// В остальных случаях возвращает exitGeneral.
func exitCode(err error) int {
	if err == nil {
		return exitOK
	}

	var cliErr *cliError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}

	return exitGeneral
}
