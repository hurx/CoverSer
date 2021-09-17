package log

import (
	"CoverSer/util/conf"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

const (
	HTTPSERVERLOGFILE = "CoverSer.log"
)

var Logger *logrus.Logger
var TaskLogger *logrus.Logger

func NewTaskLogger(logFileName_ string) *logrus.Logger {
	var logDir string
	if !filepath.IsAbs(conf.Conf.Log.LogDir) {
		absPath, err := filepath.Abs(conf.Conf.Log.LogDir)
		if err != nil {
			panic(fmt.Sprintf("init logger error:%s", err.Error()))
		}
		logDir = absPath
	} else {
		logDir = conf.Conf.Log.LogDir
	}
	logPath := filepath.Join(logDir, logFileName_)
	fmt.Printf("task log on :%s", logPath)
	writer, err := rotatelogs.New(
		logPath+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(logPath),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		panic(fmt.Sprintf("init task logger error: %s", err.Error()))
		//logrus.Error("init log error:", err)
	}

	Log := logrus.New()
	Log.Level = logrus.DebugLevel

	Log.SetReportCaller(true)
	Log.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.InfoLevel:  writer,
			logrus.ErrorLevel: writer,
			logrus.DebugLevel: writer,
		},
		&easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%][%field%]: %time% - %msg% \n",
		},
	))
	TaskLogger = Log
	return Log
}

func Jtos(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func Init() {
	curDir, _ := os.Getwd()

	var loggerDir = filepath.Join(curDir, "log")
	if conf.Conf.Log.LogDir != "" {
		if !filepath.IsAbs(conf.Conf.Log.LogDir) {
			absPath, err := filepath.Abs(conf.Conf.Log.LogDir)
			if err != nil {
				panic(fmt.Sprintf("init logger error:%s", err.Error()))
			}
			loggerDir = absPath
		} else {
			loggerDir = conf.Conf.Log.LogDir
		}
	}
	if _, err := os.Stat(loggerDir); os.IsNotExist(err) {
		os.MkdirAll(loggerDir, os.ModePerm)
	}
	logPath := filepath.Join(loggerDir, HTTPSERVERLOGFILE)
	writer, err := rotatelogs.New(
		logPath+".%Y-%m-%d",
		rotatelogs.WithLinkName(logPath),
		rotatelogs.WithMaxAge(7*24*time.Hour),     // 保留7天
		rotatelogs.WithRotationTime(24*time.Hour), // 每天一份日志
	)
	if err != nil {
		logrus.Error("init log error:", err)
	}

	Logger = logrus.New()
	Logger.Level = logrus.DebugLevel

	Logger.SetReportCaller(true)
	Logger.AddHook(lfshook.NewHook(
		lfshook.WriterMap{
			logrus.InfoLevel:  writer,
			logrus.ErrorLevel: writer,
			logrus.DebugLevel: writer,
		},
		&easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%][%field%]: %time% - %msg% \n",
		},
	))
}

func LoggerHandler() gin.HandlerFunc {

	return func(c *gin.Context) {
		t := time.Now()

		// before request
		c.Next()
		// after request
		latency := time.Since(t)
		// request method
		reqMethod := c.Request.Method
		// 请求路由
		reqUri := c.Request.RequestURI
		// 状态码
		statusCode := c.Writer.Status()
		// 请求IP
		clientIP := c.ClientIP()
		if statusCode/100 <= 2 {
			Logger.WithField("field", "[Gin Info]").Infof(` %v %v responseTime: %v status: %v remoteIp: %s`,
				reqUri, reqMethod, latency, statusCode, clientIP)
		} else {
			Logger.WithField("field", "[Gin Error]").Errorf(`%v %v responseTime: %v status: %v remoteIp: %s`,
				reqUri, reqMethod, latency, statusCode, clientIP)
		}

	}
}
