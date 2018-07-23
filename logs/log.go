package logs

import (
	"fmt"
	"github.com/rifflock/lfshook"
	logrus "github.com/sirupsen/logrus"
	"os"
)

var Log *logrus.Logger

func SetupLogs(logFilePath string, logLevel int) error {

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
	logrus.SetFormatter(formatter) // for any "normal" log messages

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
	case 5:
		Log.SetLevel(logrus.DebugLevel)
	case 4:
		Log.SetLevel(logrus.InfoLevel)
	case 3:
		Log.SetLevel(logrus.WarnLevel)
	case 2:
		Log.SetLevel(logrus.ErrorLevel)
	case 1:
		Log.SetLevel(logrus.FatalLevel)
	case 0:
		Log.SetLevel(logrus.PanicLevel)
	default:
		return fmt.Errorf("Invalid logging param passed, proceeding with defaults")
	}

	return nil
}
