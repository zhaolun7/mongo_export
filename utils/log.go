package utils

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// use this Logger for all log
var Logger *logrus.Logger = nil

type myFormatter struct{}

func (s *myFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("2006/01/02 15:04:05")
	msg := fmt.Sprintf("%s [%s] %s\n", timestamp, strings.ToUpper(entry.Level.String()), entry.Message)
	return []byte(msg), nil
}

/*
获取程序运行路径
*/
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic("err")
	}
	return strings.Replace(dir, "\\", "/", -1)
}
func init() {
	Logger = logrus.New()
	Logger.SetFormatter(new(myFormatter))
	Logger.SetLevel(logrus.InfoLevel)
	Logger.SetReportCaller(true)
	path := getCurrentDirectory() + "/logs/server.log"
	logger := &lumberjack.Logger{
		LocalTime:  true,
		Filename:   path,
		MaxSize:    200, // megabytes
		MaxBackups: 5,
		MaxAge:     30,    //days
		Compress:   false, // disabled by default
	}
	writers := []io.Writer{
		logger,
		//os.Stdout,
	}
	fileAndStdoutWriter := io.MultiWriter(writers...)
	Logger.SetOutput(fileAndStdoutWriter)
	Logger.Info("Logger init complete.")
	/*
		logrus.SetFormatter(new(myFormatter))
		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetReportCaller(true)

		path := getCurrentDirectory() + "/logs/server.log"
		//fmt.Println(path)
		logger:=&lumberjack.Logger{
			LocalTime:  true,
			Filename:   path,
			MaxSize:    200, // megabytes
			MaxBackups: 5,
			MaxAge:     30,    //days
			Compress:   false, // disabled by default
		}
		writers := []io.Writer{
			logger,
			//os.Stdout,
		}
		fileAndStdoutWriter := io.MultiWriter(writers...)
		logrus.SetOutput(fileAndStdoutWriter)
		logrus.Info("logrus init complete.")
	*/
}
