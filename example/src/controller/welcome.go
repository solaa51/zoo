package controller

import (
	"github.com/solaa51/zoo/system/control"
)

type Welcome struct {
	control.Controller
}

func (w *Welcome) Index() {
	w.Ctx.JsonReturn(0, "", "")
}
