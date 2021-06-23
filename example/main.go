package main

import (
	"github.com/solaa51/zoo/system/gHttp"
	"github.com/solaa51/zoo/system/mLog"
)

func main() {
	//启动http服务
	err := gHttp.Start()
	if err != nil {
		mLog.Fatal(err)
	}
}
