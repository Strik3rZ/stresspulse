package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	debugLogger   *log.Logger
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
)

func Init(logLevel string) {
	debugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	warningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	switch strings.ToLower(logLevel) {
	case "debug":
		debugLogger.SetOutput(os.Stdout)
	case "info":
		debugLogger.SetOutput(os.Stdout)
		infoLogger.SetOutput(os.Stdout)
	case "warn":
		debugLogger.SetOutput(os.Stdout)
		infoLogger.SetOutput(os.Stdout)
		warningLogger.SetOutput(os.Stdout)
	case "error":
		debugLogger.SetOutput(os.Stdout)
		infoLogger.SetOutput(os.Stdout)
		warningLogger.SetOutput(os.Stdout)
		errorLogger.SetOutput(os.Stderr)
	default:
		debugLogger.SetOutput(os.Stdout)
		infoLogger.SetOutput(os.Stdout)
	}
}

func Debug(format string, v ...interface{}) {
	debugLogger.Output(2, fmt.Sprintf(format, v...))
}

func Info(format string, v ...interface{}) {
	infoLogger.Output(2, fmt.Sprintf(format, v...))
}

func Warning(format string, v ...interface{}) {
	warningLogger.Output(2, fmt.Sprintf(format, v...))
}

func Error(format string, v ...interface{}) {
	errorLogger.Output(2, fmt.Sprintf(format, v...))
}

// GetLogger возвращает базовый логгер для использования в других модулях
func GetLogger() *log.Logger {
	return log.New(os.Stdout, "", log.Ldate|log.Ltime)
} 