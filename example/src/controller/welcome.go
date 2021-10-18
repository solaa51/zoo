package controller

import (
	"fmt"
	"github.com/solaa51/zoo/system/control"
)

type Welcome struct {
	control.Controller
}

// 优先调用
func (w *Welcome) PreInit() {
	fmt.Println("我是优先调用的")
}

func (w *Welcome) Index() {
	w.Ctx.JsonReturn(0, "", "")
}
