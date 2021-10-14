package mLog

import (
	"bytes"
	"fmt"
	rotateLogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"github.com/solaa51/zoo/system/cFunc"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var env string
var ll *log.Logger

// SetEvn 设置当前开发环境
func SetEvn(e string) {
	env = e
	if env != "test" {
		SetOutput(&bytes.Buffer{})
	} else {
		SetOutput(os.Stdout)
	}
}

func Debug(args ...interface{}) {
	ll.Debug(args...)
}

func Warn(args ...interface{}) {
	ll.Warn(args)
}

func Info(args ...interface{}) {
	ll.Info(args)
}

func Error(args ...interface{}) {
	ll.Error(args)
}

func Fatal(args ...interface{}) {
	ll.Fatal(args...)
}

func Panic(args ...interface{}) {
	ll.Panic(args...)
}

func SetOutput(output io.Writer) {
	ll.SetOutput(output)
}

func init() {
	ll = log.New()
	ll.SetLevel(log.TraceLevel)

	//日志拆分配置 最长保留7天 24小时更新一次日志
	logFullPath := cFunc.GetAppDir() + "logs/log"
	//fileSuffix := time.Now().Format("2006-01-02") + ".log"
	writer, err := rotateLogs.New(
		//logFullPath+"-"+fileSuffix,
		logFullPath+".%Y-%m-%d",
		rotateLogs.WithLinkName(logFullPath),      // 生成软链，指向最新日志文
		rotateLogs.WithRotationCount(5),           // 文件最大保存份数
		rotateLogs.WithRotationTime(24*time.Hour), // 日志切割时间间隔
	)

	if err != nil {
		log.Fatal("配置本地日志存储出错:", err)
	}

	ll.AddHook(lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: writer, // 为不同级别设置不同的输出目的
		log.InfoLevel:  writer,
		log.WarnLevel:  writer,
		log.ErrorLevel: writer,
		log.FatalLevel: writer,
		log.PanicLevel: writer,
	}, &myLogFormat{}))

	SetOutput(os.Stdout) //默认在标准输出打印信息
}

//自定义日志输出格式
type myLogFormat struct{}

func (_ *myLogFormat) Format(entry *log.Entry) ([]byte, error) {
	var file string
	var funcName string
	var line int
	if env == "test" || entry.Level == log.ErrorLevel || entry.Level == log.WarnLevel || entry.Level == log.DebugLevel {
		//调用内置的caller不能跳过mLog包  这部分需要自己实现 10是该项目下适合的调用位置索引
		var pc uintptr
		var ok bool
		pc, file, line, ok = runtime.Caller(10)
		if ok {
			file = filepath.Base(file)
			funcName = runtime.FuncForPC(pc).Name()
		}
		/* 可通过以下方法 确定具体的索引的位置
		var pc uintptr
		for i:=1; i<25; i++ {
			pc, file, line, ok = runtime.Caller(i)
			if ok {
				file = filepath.Base(file)
				funcName := runtime.FuncForPC(pc).Name()
				fmt.Println(i, "-", file, "--", funcName, "---", line)
			}else{
				break
			}
		}*/
	}

	//[时间] [level] [file:funcName:line] [msg] //--[data]
	msg := fmt.Sprintf("[%s] [%s] [%s:%s:%d] %s\n", time.Now().Local().Format("2006-01-02 15:04:05"), strings.ToUpper(entry.Level.String()), file, funcName, line, entry.Message)
	return []byte(msg), nil
}
