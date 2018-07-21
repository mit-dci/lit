package logs

import (
  logrus "github.com/sirupsen/logrus"
  "github.com/rifflock/lfshook"
)
var Log *logrus.Logger

func SetupLogs(logFilePath string) error {


  pathMap := lfshook.PathMap{
  	logrus.InfoLevel:  logFilePath,
  	logrus.ErrorLevel: logFilePath,
  }

  formatter := new(logrus.TextFormatter)
  formatter.TimestampFormat = "2006-01-02 15:04:05.999999"
  formatter.FullTimestamp = true

  logrus.SetFormatter(formatter) // for normal logs, wont be saevd to logfile
  Log = logrus.New()
  Log.Hooks.Add(lfshook.NewHook(
  	pathMap,
  	&logrus.TextFormatter{},
  ))
  return nil
}
