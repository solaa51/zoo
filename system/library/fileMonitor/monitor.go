package fileMonitor

import (
	"bufio"
	"os"
	"sync"
	"time"
	"zoo/system/mLog"
)

//结构
type confModify struct {
	modTime int64  //文件修改时间
	content []byte //文件内容
	path    string
	m       sync.Mutex
}

func New(fullFileName string, pf func(interface{})) {
	_, err := os.Stat(fullFileName)
	if err != nil {
		mLog.Error("文件获取失败，无法监听")
	}

	conf := &confModify{
		path: fullFileName,
	}

	go func() {
		for {
			conf.listenModify(pf)
			time.Sleep(1 * time.Second)
		}
	}()
}

//监控文件状态 变化时 执行预设函数
func (c *confModify) listenModify(pf func(interface{})) {
	c.m.Lock()
	file, err := os.Open(c.path)
	if err != nil {
		mLog.Fatal("获取文件出错")
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		mLog.Fatal("获取文件基本信息出错")
	}

	if fileInfo.ModTime().Unix() != c.modTime {
		c.modTime = fileInfo.ModTime().Unix()

		//调用参数 看情况 传递 本处为返回文件内容
		//如果文件内容大 则最好 在自定义函数中自己处理
		fr := bufio.NewReader(file)
		b2 := make([]byte, fileInfo.Size())
		_, _ = fr.Read(b2)
		c.content = b2
		pf(string(b2)) //调用函数
	}

	c.m.Unlock()
}
