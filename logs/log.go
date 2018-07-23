package logs

import (
	"fmt"
	"github.com/rifflock/lfshook"
	logrus "github.com/sirupsen/logrus"
	"os"
)

var Log *logrus.Logger

func SetupTestLogs() {

	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	// magic date, please don't change.
	formatter.FullTimestamp = true

	Log = &logrus.Logger{
		Out:       os.Stderr,
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		// set Level below
	}
	Log.Info("COOL!")
	logrus.SetFormatter(formatter) // for any "normal" log messages
	Log.SetLevel(logrus.DebugLevel)
}

func SetupFormatter(logFile *os.File, logLevel int) {
	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	// magic date, don't change.
	formatter.FullTimestamp = true
	logrus.SetFormatter(formatter) // for any "normal" log messages

	Log = &logrus.Logger{
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		// set Level below
	}
	Log.SetLevel(logrus.DebugLevel)
	if logLevel == 0 {
		Log.Out = logFile
	} else {
		Log.Out = os.Stderr
	}
}

func SetupLogs(logFile *os.File, logFilePath string, logLevel int) error {

	SetupFormatter(logFile, logLevel)
	if logLevel == 0 {
		return nil
	}
	pathMap := lfshook.PathMap{
		logrus.InfoLevel:  logFilePath,
		logrus.ErrorLevel: logFilePath,
		logrus.WarnLevel:  logFilePath,
		logrus.DebugLevel: logFilePath,
		logrus.PanicLevel: logFilePath,
		logrus.FatalLevel: logFilePath,
	}
	// setup a local filesystem hook to write to the logfile
	// logs will be of the format:
	// time="2018-07-23 10:47:03.617692" level=warning msg="You should start over and use a good passphrase!\n"
	Log.Hooks.Add(lfshook.NewHook(
		pathMap,
		&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.999999",
			FullTimestamp:   true,
		},
	))
	Log.SetLevel(logrus.DebugLevel)
	switch logLevel {
	case 6:
		Log.SetLevel(logrus.DebugLevel)
	case 5:
		Log.SetLevel(logrus.InfoLevel)
	case 4:
		Log.SetLevel(logrus.WarnLevel)
	case 3:
		Log.SetLevel(logrus.ErrorLevel)
	case 2:
		Log.SetLevel(logrus.FatalLevel)
	case 1:
		Log.SetLevel(logrus.PanicLevel)
	default:
		return fmt.Errorf("Invalid logging param passed, proceeding with defaults")
	}

	return nil
}
