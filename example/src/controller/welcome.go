package controller

import (
	"fmt"
	"github.com/solaa51/zoo/system/control"
)

type Welcome struct {
	control.Controller
}

// PreInit web调用时会优先调用 可用来做一些初始化判断操作
func (w *Welcome) PreInit() {
	fmt.Println("我是优先调用的")
}

func (w *Welcome) Index() {
	w.Ctx.JsonReturn(0, "", "")
}
