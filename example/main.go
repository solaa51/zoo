package main

import (
	"github.com/solaa51/zoo/example/router"
	"github.com/solaa51/zoo/system/gHttp"
	"github.com/solaa51/zoo/system/mLog"
)

func main() {
	//加载路由配置
	_ = router.CustomRouter{}

	//启动http服务
	err := gHttp.Start()
	if err != nil {
		mLog.Fatal(err)
	}
}
