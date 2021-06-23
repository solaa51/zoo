package handler

import (
	"context"
	"errors"
	"github.com/solaa51/zoo/system/cFunc"
	"github.com/solaa51/zoo/system/config"
	"github.com/solaa51/zoo/system/control"
	"github.com/solaa51/zoo/system/mCtx"
	"github.com/solaa51/zoo/system/mLog"
	"github.com/solaa51/zoo/system/router"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"
)

var Handle = New()

type NewControl func() control.Control

func New() *MHandle {
	return &MHandle{
		compile: make(map[string]NewControl, 0),
		outTime: 10 * time.Second, //默认超时时间30秒
	}
}

// AddCompile 添加控制器的映射规则
func AddCompile(className string, nc NewControl) {
	Handle.compile[className] = nc
}

// MHandle http请求处理程序结构
// 控制器必须继承control.Controller
type MHandle struct {
	compile map[string]NewControl //控制器映射 实例化规则

	msg     string        //超时文本提示信息
	outTime time.Duration //超时时间 默认30秒
	ctx     context.Context
}

// http请求调用入口
func (m *MHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//处理静态文件请求
	if r.URL.String() == "/favicon.ico" {
		http.ServeFile(w, r, "./favicon.ico")
		return
	}

	//解析路由规则 解析出class method params
	className, methodName, params := router.ParseRoute(r)
	if className == "" || methodName == "" {
		http.NotFound(w, r)
		return
	}

	//检查IP 是否允许通过
	if !config.IpPassCheck(cFunc.ClientIP(r)) {
		mLog.Warn(cFunc.ClientIP(r) + " - " + r.RequestURI + " - " + className + "-" + methodName + " - IP被禁止")
		http.Error(w, cFunc.ClientIP(r)+"被禁止", http.StatusNotFound)
		return
	}

	//handler解析到class的实例
	controlInterface, err := m.parseCompile(className)
	if err != nil {
		mLog.Warn(cFunc.ClientIP(r) + " - " + r.RequestURI + " - " + className + "-" + methodName + " - " + err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	//handler校验method以及params
	call, args, err := m.checkMethodParams(methodName, params, controlInterface)
	if err != nil {
		mLog.Warn(cFunc.ClientIP(r) + " - " + r.RequestURI + " - " + className + "-" + methodName + " - " + err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	//记录访问日志
	mLog.Info(cFunc.ClientIP(r) + " - " + r.RequestURI + " - " + className + "-" + methodName + " - " + r.Header.Get("User-Agent"))

	/*//设置handler超时ctx
	ctx := m.ctx
	if ctx == nil {
		var cancelCtx context.CancelFunc
		if m.outTime <= 0 { //没有设置超时时间
			ctx, cancelCtx = context.WithCancel(r.Context())
		} else {
			ctx, cancelCtx = context.WithTimeout(r.Context(), m.outTime)
		}
		defer cancelCtx()
	}*/

	//设置控制器 context 信息
	cc := controlInterface.(control.Control)     //转义为 control interface
	err = cc.SetCtx(w, r, className, methodName) //利用组合特效，设置controller的Ctx成员属性
	if err != nil {
		mLog.Warn(cFunc.ClientIP(r) + " - " + r.RequestURI + " - " + className + "-" + methodName + " - " + err.Error())
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer func() { //处理panic 需要在调用之前声明
		if e := recover(); e != nil {
			switch e {
			case mCtx.JSONRETURN: //正常提前返回退出请求应答
			case mCtx.TEXTRETURN: //正常提前返回退出请求应答
			default: //如果想输出错误状态码 请在writer写入数据前，写入writerHeader为500等状态码  默认情况均为200状态码
				//记录日志：
				var buf [4096]byte
				n := runtime.Stack(buf[:], false)
				mLog.Error("PANIC:", string(buf[:n]))
			}
		}
	}()
	//调用方法
	call.Call(args)

	/*//调用方式二:
	//处理异常panic
	panicChan := make(chan interface{}, 1)
	done := make(chan struct{}) //标记处理完成

	// 处理请求
	go func() {
		defer func() {
			if p := recover(); p != nil {
				switch p {
				case mCtx.JSONRETURN: //正常退出
					panicChan <- 1
				default: //如果想输出错误状态码 请在writer写入数据前，写入writerHeader为500等状态码  默认情况均为200状态码
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					mLog.Error("PANIC:", string(buf[:n]))
					panicChan <- 2
				}
			}
		}()

		call.Call(args)
		close(done)
	}()

	select {
	case p := <-panicChan: //panic信号
		if p == 1 { //正常的panic返回 退出信号
			responseWriter.Mu.Lock()
			defer responseWriter.Mu.Unlock()
			if responseWriter.Status > 0 && responseWriter.Status != 200 {
				w.WriteHeader(responseWriter.Status)
			}
			if responseWriter.RetType == "json" {
				w.Header().Set("Content-Type", "application/json;charset=UTF-8")
			}
			_, _ = w.Write([]byte(responseWriter.Body))
		} else { //异常
			w.WriteHeader(500)
		}
	case <-done: //正常返回 或 指定的panic返回
		responseWriter.Mu.Lock()
		defer responseWriter.Mu.Unlock()
		if responseWriter.Status >= 0 {
			w.WriteHeader(responseWriter.Status)
		}
		if responseWriter.RetType == "json" {
			w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		}
		_, _ = w.Write([]byte("我是自己写的输出 done信号"))
		_, _ = w.Write([]byte(responseWriter.Body))
	case <-ctx.Done(): //处理超时
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = io.WriteString(w, m.timeoutBody())
	}*/
}

//处理超时返回文本信息
func (m *MHandle) timeoutBody() string {
	return "Timeout"
}

// 解析控制器映射关系，返回映射到的控制器
func (m *MHandle) parseCompile(className string) (control.Control, error) {
	if _, OK := m.compile[className]; !OK {
		return nil, errors.New("404 class not found")
	}

	return m.compile[className](), nil
}

// 映射校验方法以及参数
func (m *MHandle) checkMethodParams(methodName string, params []string, control control.Control) (reflect.Value, []reflect.Value, error) {
	getType := reflect.TypeOf(control)
	_, bol := getType.MethodByName(methodName) //判断是否存在调用的方法
	if !bol {
		return reflect.Value{}, nil, errors.New("404 method not found")
	}

	getValue := reflect.ValueOf(control)

	/*//给control赋值ctx的另一种方式
	elem := getValue.Elem()
	elem.FieldByName("R").Set(reflect.ValueOf(r))
	ctxStruct := elem.FieldByName("Ctx")
	c, err := mCtx.New(r, w, className, methodName)
	if err != nil {
		return reflect.Value{}, nil, err
	}
	ctxStruct.Set(reflect.ValueOf(c))*/

	method := getValue.MethodByName(methodName)
	//method.Type().NumIn() //获取到参数个数
	//method.Type().In(index) //获取指定位置的参数类型
	args := make([]reflect.Value, method.Type().NumIn())
	if method.Type().NumIn() > 0 {
		if method.Type().NumIn() != len(params) {
			//方法参数不匹配
			return reflect.Value{}, nil, errors.New("参数不匹配")
		}
		for i := 0; i < method.Type().NumIn(); i++ {
			switch method.Type().In(i).Name() {
			case "string":
				args[i] = reflect.ValueOf(params[i])
			case "int64":
				tmpInt64, _ := strconv.ParseInt(params[i], 10, 64)
				args[i] = reflect.ValueOf(tmpInt64)
			default:
				return reflect.Value{}, nil, errors.New("参数类型不匹配")
			}
		}
	}

	return method, args, nil
}
