package mLog

import (
	"fmt"
	"github.com/solaa51/zoo/system/cFunc"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

// NLog 用于处理少量日志
type NLog struct {
	sync.Mutex
	prefix string
	env    string
}

// NewLog 开发环境  文件名前缀
func NewLog(e string, logFilePrefix string) *NLog {
	l := &NLog{
		prefix: logFilePrefix,
	}
	env = e

	path := cFunc.GetAppDir() + "logs/"
	_, err := os.Stat(path)
	if err != nil {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatal("无法创建日志文件夹：", err)
		}
	}

	return l
}

func (l *NLog) Info(s string) {
	l.echo("INFO", s)
}

func (l *NLog) Warn(s string) {
	l.echo("WARN", s)
}

func (l *NLog) Error(s string) {
	l.echo("ERROR", s)
}

func (l *NLog) Trace(s string) {
	l.echo("TRACE", s)
}

func (l *NLog) Debug(s string) {
	l.echo("DEBUG", s)
}

func (l *NLog) echo(flag, s string) {
	l.Lock()
	defer l.Unlock()

	//| log.Lshortfile 输出的是 当前出错输出内容的行 没什么意义
	var pc uintptr
	var fileName string
	var line int
	var funcName string

	if env == "test" { //该函数对性能的消耗比较大，仅test时输出
		pc, fileName, line, _ = runtime.Caller(2)
		funcName = runtime.FuncForPC(pc).Name()
		/*if strings.HasSuffix(funcName, "mCtx.New") {
			pc, fileName, line, _ = runtime.Caller(3)
			funcName = runtime.FuncForPC(pc).Name()
		}*/
		fileName = filepath.Base(fileName)
	}

	msg := "[" + flag + "] [" + cFunc.Date("Y-m-d H:i:s", 0) + "] [" + fileName + ":" + funcName + ":" + strconv.Itoa(line) + "] " + s + "\n"

	//获取文件
	logDateName := l.prefix + cFunc.Date("Y-m-d", 0)
	logFileName := cFunc.GetAppDir() + "logs/" + logDateName + ".log"

	if _, err := os.Stat(logFileName); err != nil {
		// 文件不存在,创建
		_, err = os.Create(logFileName)
		if err != nil {
			panic(err)
		}
	}

	logFile, err := os.OpenFile(logFileName, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic("打开日志文件时发生错误")
	}

	//延迟关闭文件
	defer logFile.Close()

	//写入文件
	n, _ := logFile.Seek(0, io.SeekEnd)
	_, _ = logFile.WriteAt([]byte(msg), n)

	if env == "test" {
		fmt.Println(s)
	}
}
