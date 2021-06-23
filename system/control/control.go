package control

import (
	"github.com/solaa51/zoo/system/mCtx"
	"net/http"
)

// Control http服务 class接口
type Control interface {
	SetCtx(w http.ResponseWriter, r *http.Request, className string, methodName string) error //设置请求上下文处理
}

// Controller 基础控制器 http服务上的其他控制器必须继承Controller才能正常使用
type Controller struct {
	Ctx *mCtx.Con
}

// SetCtx 设置请求上下文处理 利用组合 设置控制器的Ctx成员
// 也可以利用反射设置，参考handler内的代码
func (c *Controller) SetCtx(w http.ResponseWriter, r *http.Request, className string, methodName string) error {
	ctx, err := mCtx.New(w, r, className, methodName)
	if err != nil {
		return err
	}

	c.Ctx = ctx

	return nil
}
