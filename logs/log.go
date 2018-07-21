package logs

import (
  "fmt"
  logrus "github.com/sirupsen/logrus"
  "github.com/rifflock/lfshook"
)
var Log *logrus.Logger

func SetupLogs(logFilePath string, logLevel int) error {

  switch logLevel {
  case 5:
    logrus.SetLevel(logrus.DebugLevel)
  case 4:
    logrus.SetLevel(logrus.InfoLevel)
  case 3:
    logrus.SetLevel(logrus.WarnLevel)
  case 2:
    logrus.SetLevel(logrus.ErrorLevel)
  case 1:
    logrus.SetLevel(logrus.FatalLevel)
  case 0:
    logrus.SetLevel(logrus.PanicLevel)
  default:
    return fmt.Errorf("Invalid logging param passed, proceeding with defaults")
  }

  pathMap := lfshook.PathMap{
  	logrus.InfoLevel:  logFilePath,
  	logrus.ErrorLevel: logFilePath,
  }

  formatter := new(logrus.TextFormatter)
  formatter.TimestampFormat = "2006-01-02 15:04:05.999999"
  formatter.FullTimestamp = true

  logrus.SetFormatter(formatter) // for normal logs, wont be saevd to logfile
  logrus.Info("THIS IS BS")
  Log = logrus.New()
  Log.Hooks.Add(lfshook.NewHook(
  	pathMap,
  	&logrus.JSONFormatter{},
  ))
  Log.Info("THIS IS SHIT ")
  logrus.Error("WTF IS THIS")
  return nil
}
