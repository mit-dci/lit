package logging

import (
	"fmt"
	"io"
	"log"
	"os"
)

type LogLevel int

const (
	LogLevelError   LogLevel = 0
	LogLevelWarning LogLevel = 1
	LogLevelInfo    LogLevel = 2
	LogLevelDebug   LogLevel = 3
)

var logLevel = LogLevelError

func SetLogLevel(newLevel int) {
	logLevel = LogLevel(newLevel)
}

func SetLogFile(logFile io.Writer) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	logOutput := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logOutput)
}

func getPrefix(level string) string {
	return fmt.Sprintf("[%s]", level)
}

func Fatalln(args ...interface{}) {
	log.Fatalln(args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func Debugf(format string, args ...interface{}) {
	if logLevel >= LogLevelDebug {
		log.Printf(fmt.Sprintf("%s %s", getPrefix("DEBUG"), format), args...)
	}
}

func Infof(format string, args ...interface{}) {
	if logLevel >= LogLevelInfo {
		log.Printf(fmt.Sprintf("%s %s", getPrefix("INFO"), format), args...)
	}
}

func Warnf(format string, args ...interface{}) {
	if logLevel >= LogLevelWarning {
		log.Printf(fmt.Sprintf("%s %s", getPrefix("WARN"), format), args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if logLevel >= LogLevelError {
		log.Printf(fmt.Sprintf("%s %s", getPrefix("ERROR"), format), args...)
	}
}

func Debugln(args ...interface{}) {
	if logLevel >= LogLevelDebug {
		args = append([]interface{}{getPrefix("DEBUG")}, args...)
		log.Println(args...)
	}
}

func Infoln(args ...interface{}) {
	if logLevel >= LogLevelInfo {
		args = append([]interface{}{getPrefix("INFO")}, args...)
		log.Println(args...)
	}
}

func Warnln(args ...interface{}) {
	if logLevel >= LogLevelWarning {
		args = append([]interface{}{getPrefix("WARN")}, args...)
		log.Println(args...)
	}
}

func Errorln(args ...interface{}) {
	if logLevel >= LogLevelError {
		args = append([]interface{}{getPrefix("ERROR")}, args...)
		log.Println(args...)
	}
}

func Debug(args ...interface{}) {
	if logLevel >= LogLevelDebug {
		args = append([]interface{}{getPrefix("DEBUG")}, args...)
		log.Print(args...)
	}
}

func Info(args ...interface{}) {
	if logLevel >= LogLevelInfo {
		args = append([]interface{}{getPrefix("INFO")}, args...)
		log.Print(args...)
	}
}

func Warn(args ...interface{}) {
	if logLevel >= LogLevelWarning {
		args = append([]interface{}{getPrefix("WARN")}, args...)
		log.Print(args...)
	}
}

func Error(args ...interface{}) {
	if logLevel >= LogLevelError {
		args = append([]interface{}{getPrefix("ERROR")}, args...)
		log.Print(args...)
	}
}

func SetupTestLogs() {
	logLevel = LogLevelDebug
}
