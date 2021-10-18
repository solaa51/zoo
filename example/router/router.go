package router

import (
	"github.com/solaa51/zoo/example/src/controller"
	"github.com/solaa51/zoo/system/control"
	"github.com/solaa51/zoo/system/handler"
	"github.com/solaa51/zoo/system/router"
)

// CustomRouter 路由配置 该struct仅用来包装 会被系统加载
type CustomRouter struct{}

//自定义 url路由匹配规则
func init() {
	router.AddCompile(`welcome/index`, "welcome/index")
	router.AddCompile(`welcome/version/(\w+)/(\w+)`, "welcome/version/$1/$2")
}

// appCheck工具可自动配置 路由与控制器的映射关系 该部分可由bin下的appCheck工具自动生成
func init() {
	//仅提供对内的访问
	handler.AddCompile("welcome", func() control.Control { return &controller.Welcome{} })

}
