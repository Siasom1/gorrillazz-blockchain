package log

import "log"

type Logger struct {
	Level string
}

func NewLogger(level string) *Logger {
	return &Logger{Level: level}
}

func (l *Logger) Info(msg string) {
	if l.Level == "info" || l.Level == "debug" {
		log.Println("[INFO]", msg)
	}
}

func (l *Logger) Debug(msg string) {
	if l.Level == "debug" {
		log.Println("[DEBUG]", msg)
	}
}

func (l *Logger) Error(msg string) {
	log.Println("[ERROR]", msg)
}
