package errors

import (
	"encoding/json"
	"errors"
	"log"
	"runtime"
)

// Уровни важности ошибки
const (
	Critical = iota
	Notice
	Warning
	Ingnored
)

const DefaultSeverity = Notice

// Интерфейс для работы с ошибками enriched с доп. инфой
type ErrorInfo interface {
	PushStack() ErrorInfo
	WithSeverity(severity int) ErrorInfo
	Severity() int
	WithData(data map[string]any) ErrorInfo
	IsNil() bool
	Unwrap() error
	Error() string
	JSON() string
	Fatal()
}

// Основная реализация интерфейса ошибок
type ErrorInfoImpl struct {
	Message       string         `json:"message"`
	Data          map[string]any `json:"data"`
	Err           error          `json:"-"`
	Stack         []CodeLocation `json:"stack"`
	BirthLocation *CodeLocation  `json:"birth_location"`
	SeverityLevel int            `json:"severity"`
}

// Место в коде для стека
type CodeLocation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

// Создать ErrorInfoImpl из err и msg
func FromError(err error, msg string) ErrorInfo {
	return &ErrorInfoImpl{
		Message:       msg,
		Data:          make(map[string]any),
		Err:           err,
		Stack:         make([]CodeLocation, 0),
		BirthLocation: getCodeLocation(),
		SeverityLevel: DefaultSeverity,
	}
}

// Создать ErrorInfoImpl из строки ошибки и описания
func NewError(err string, msg string) ErrorInfo {
	return FromError(errors.New(err), msg)
}

// "Пустая" ошибка (обычно для успешного кейса)
func Nil() ErrorInfo {
	return &ErrorInfoImpl{
		Message:       "nil",
		Data:          make(map[string]any),
		Err:           nil,
		Stack:         make([]CodeLocation, 0),
		BirthLocation: getCodeLocation(),
		SeverityLevel: Ingnored,
	}
}

// Получить информацию о месте вызова (для stack trace)
func getCodeLocation() *CodeLocation {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}
	return &CodeLocation{
		File:     file,
		Line:     line,
		Function: runtime.FuncForPC(pc).Name(),
	}
}

// Добавить новый фрейм стека
func (e *ErrorInfoImpl) PushStack() ErrorInfo {
	e.Stack = append(e.Stack, *getCodeLocation())
	return e
}

// Установить уровень важности ошибки
func (e *ErrorInfoImpl) WithSeverity(severity int) ErrorInfo {
	e.SeverityLevel = severity
	return e
}

// Получить уровень важности ошибки
func (e *ErrorInfoImpl) Severity() int {
	return e.SeverityLevel
}

// Установить произвольные данные к ошибке
func (e *ErrorInfoImpl) WithData(data map[string]any) ErrorInfo {
	e.Data = data
	return e
}

// Проверить на "пустую" ошибку
func (e *ErrorInfoImpl) IsNil() bool {
	return e == nil || e.Err == nil
}

// Вернуть оригинальную ошибку
func (e *ErrorInfoImpl) Unwrap() error {
	return e.Err
}

// Текстовое представление — сериализация в JSON
func (e *ErrorInfoImpl) Error() string {
	return e.JSON()
}

// Кастомная сериализация для ErrorInfo
func (e *ErrorInfoImpl) MarshalJSON() ([]byte, error) {
	type Alias ErrorInfoImpl

	var errMsg string
	if e.Err != nil {
		errMsg = e.Err.Error()
	}

	return json.Marshal(&struct {
		*Alias
		Err string `json:"err"`
	}{
		Alias: (*Alias)(e),
		Err:   errMsg,
	})
}

// Кастомная десериализация (чтобы err/string в error)
func (e *ErrorInfoImpl) UnmarshalJSON(data []byte) error {
	type Alias ErrorInfoImpl
	aux := &struct {
		*Alias
		Err any `json:"err"`
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.Err.(type) {
	case nil:
		e.Err = nil
	case string:
		if v == "" {
			e.Err = nil
		} else {
			e.Err = errors.New(v)
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			e.Err = errors.New("unknown error payload")
			return nil
		}
		e.Err = errors.New(string(b))
	}

	return nil
}

// Вернуть JSON-представление ошибки для вывода
func (e *ErrorInfoImpl) JSON() string {
	jsonBytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

// Фатальная ошибка — завершить выполнение приложения
func (e *ErrorInfoImpl) Fatal() {
	log.Fatal(e.Error())
}

// Проверка на nil/пустую ошибку для любых типов
func IsNil(e any) bool {
	switch v := e.(type) {
	case *ErrorInfoImpl:
		return v.IsNil()
	case error:
		return v == nil
	default:
		return e == nil
	}
}

// Инверсия IsNil для удобства (возвращает true если ошибка есть)
func IsNonNil(e any) bool {
	return !IsNil(e)
}
