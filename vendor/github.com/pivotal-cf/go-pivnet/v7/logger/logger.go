package logger

type Data map[string]interface{}

//go:generate counterfeiter . Logger

type Logger interface {
	Debug(action string, data ...Data)
	Info(action string, data ...Data)
}
