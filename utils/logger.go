package utils

import "fmt"

type MyLogger struct {
	logging bool
}

func NewMyLogger(logging bool) MyLogger {
	return MyLogger{logging: logging}
}

func (l *MyLogger) Log(message string) {
	if l.logging {
		fmt.Println(message)
	}
}
