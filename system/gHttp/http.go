package gHttp

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"zoo/system/config"
	"zoo/system/handler"
	"zoo/system/mLog"
)

/**
用于启动http服务和支持热重启
*/
type gracefulHttp struct {
	server   *http.Server //http服务server配置
	listener net.Listener

	httpsPem string //https ssl配置
	httpsKey string //https ssl配置
}

// 启动服务
func (g *gracefulHttp) start(config *config.Config) {
	//pprof 启动
	if config.Pprof.HTTP {
		mLog.Info("pprof启动：", config.Pprof.PORT+"/debug/pprof 访问")
		if config.Pprof.HTTPS {
			go func() {
				err := http.ListenAndServeTLS(config.Pprof.PORT, config.Pprof.HTTPSPEM, config.Pprof.HTTPSKEY, nil)
				if err != nil {
					mLog.Error("pprof启动报错：", config.Pprof.PORT)
				}
			}()
		} else {
			go func() {
				err := http.ListenAndServe(config.Pprof.PORT, nil)
				if err != nil {
					mLog.Error("pprof启动报错：", config.Pprof.PORT)
					return
				}
			}()
		}
	}

	//http 服务放于goroutine中
	go func() {
		var err error
		if g.httpsPem != "" && g.httpsKey != "" {
			err = g.server.ServeTLS(g.listener, g.httpsPem, g.httpsKey)
		} else {
			err = g.server.Serve(g.listener)
		}

		if err != nil {
			mLog.Fatal("http服务启动失败：" + err.Error())
		}
	}()
}

// 监听信号 用于热更新
// 监听信号必须位于主goroutine中 才能生效
func (g *gracefulHttp) singleListen() {
	//创建一个无阻塞信号 channel
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		sig := <-ch
		ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			mLog.Info("收到kill信号，关闭服务")
			signal.Stop(ch)
			_ = g.server.Shutdown(ctx) //平滑关闭已有连接
			return
		case syscall.SIGHUP:
			mLog.Info("收到sigHup信号:重启服务")
			err := g.restart()
			if err != nil {
				mLog.Error("热重启服务失败:", err)
			}

			_ = g.server.Shutdown(ctx) //平滑关闭已有连接
			mLog.Info("热重启完成")
			return
		}
	}
}

// 相对调用时 不能用 每隔5秒监控自己是否有更新，有更新时则发送sighup信号
func (gracefulHttp) updateSelf() {
	a, _ := filepath.Abs(os.Args[0])
	af, err := os.Stat(a)
	if err != nil {
		//fmt.Println("临时性的，不需要监控自身")
		return
	}
	aLT := af.ModTime().Unix()
	go func() {
		t := time.NewTicker(time.Second * 5)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				an, err := os.Stat(a)
				if err != nil {
					//此时可能文件正在更新，需要跳过，因为此时的文件，可能是不完整的
					continue
				}
				if an.ModTime().Unix() > aLT { //存在更新
					p, err := os.FindProcess(os.Getpid())
					if err != nil {
						mLog.Error("监听APP可执行文件，获取pid失败:", err)
						continue
					}

					//发送信号
					mLog.Info("检测到app文件更新:发送升级信号sigHup")
					_ = p.Signal(syscall.SIGHUP)
				}
			}
		}
	}()
}

// 重启服务
func (g *gracefulHttp) restart() error {
	mLog.Info("重启服务中...")
	ln, ok := g.listener.(*net.TCPListener)
	if !ok {
		return errors.New("转换tcp listener失败")
	}

	ff, err := ln.File()
	if err != nil {
		return errors.New("获取socket文件描述符失败")
	}

	cmd := exec.Command(os.Args[0], []string{"-g"}...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{ff} //重用原有的socket文件描述符

	err = cmd.Start()
	if err != nil {
		return errors.New("启动新进程报错了：" + err.Error())
	}

	return nil
}

// Start 开启一个http默认服务
func Start() error {
	cc := config.Info()
	/*if customPort != "" { //本意想允许覆盖配置文件中的端口  没什么价值，放弃掉
		cc.Http.PORT = customPort
	}*/

	return run(cc, handler.Handle)
}

// StartCustom 自定义配置和handler 启动http服务 以后再来补充
func StartCustom(config *config.Config, handler http.Handler) error {
	return newGracefulHttp(config, handler, false)
}

func run(config *config.Config, handler http.Handler) error {

	//包含-D参数则 进入后台进程
	d := flag.Bool("d", false, "启动后提权进入后台进程")
	//平滑重启 检测到升级信号时 自动赋值调用
	g := flag.Bool("g", false, "平滑重启-g，不需要手动调用") //系统自动调用

	flag.Parse()

	//fmt.Println("d参数的值", *d, *g)

	daemon(*d)

	return newGracefulHttp(config, handler, *g)
}

//进入守护进程
func daemon(d bool) {
	if d && os.Getppid() != 1 { //判断父进程  父进程为1则表示已被系统接管
		filePath, _ := filepath.Abs(os.Args[0]) //将启动命令 转换为 绝对地址命令
		cmd := exec.Command(filePath, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Start()

		os.Exit(0)
	}
}

//检查所需配置 构建gracefulHttp
func newGracefulHttp(config *config.Config, handler http.Handler, gracefulReload bool) error {
	var err error

	var ln net.Listener
	if gracefulReload { //启动命令中包含参数 热重启时，从socket文件描述符 重新启动一个监听
		//当存在监听socket时 socket的文件描述符就是3 所以从本进程的3号文件描述符 恢复socket监听
		f := os.NewFile(3, "")
		ln, err = net.FileListener(f)
		if err != nil {
			return err
		}

		mLog.Info("升级重启-", os.Args, ln.Addr().String())
	} else {
		ln, err = net.Listen("tcp", config.Http.PORT)
		if err != nil {
			return err
		}
	}

	//路由管理器
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:         config.Http.PORT,
		Handler:      mux,
		TLSConfig:    nil,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}

	gf := &gracefulHttp{
		server:   server,
		listener: ln,
		httpsPem: config.Http.HTTPSPEM,
		httpsKey: config.Http.HTTPSKEY,
	}

	//goroutine 启动http服务
	gf.start(config)

	mLog.Info("服务启动完成-进程pid:", os.Getpid(), " http端口为:"+config.Http.PORT)

	//监控该APP可执行文件是否更新
	gf.updateSelf()

	//主进程监控信号
	gf.singleListen()

	return nil
}
