package node

import (
	"log"
)

type Logger struct {
	level string
}

func NewLogger(level string) *Logger {
	return &Logger{level: level}
}

func (l *Logger) Info(msg string) {
	if l.level == "info" || l.level == "debug" {
		log.Println("[INFO]", msg)
	}
}

func (l *Logger) Debug(msg string) {
	if l.level == "debug" {
		log.Println("[DEBUG]", msg)
	}
}

func (l *Logger) Error(msg string) {
	log.Println("[ERROR]", msg)
}
