package config

import (
	"errors"
	"github.com/solaa51/zoo/system/cFunc"
	"github.com/solaa51/zoo/system/library/fileMonitor"
	"github.com/solaa51/zoo/system/mLog"
	"os"
	"strings"
)

var config = New("app.toml")

// Http http服务配置
type Http struct {
	HTTP     bool   `toml:"http"`  //是否开启http服务
	PORT     string `toml:"PORT"`  //http 监听端口
	HTTPS    bool   `toml:"https"` //是否开启https服务
	HTTPSKEY string `toml:"httpsKey"`
	HTTPSPEM string `toml:"httpsPem"`
}

// StaticConfig 静态文件匹配配置
type StaticConfig struct {
	Prefix    string `toml:"prefix"`    //html js等引入文件的前缀路径
	LocalPath string `toml:"localPath"` //对应的本地路径 为空则表示不替换
	Index     string `toml:"index"`     //默认查找的文件 必须填写
}

type Config struct {
	AppName    string `toml:"appName"`    //程序名称
	AppVersion string `toml:"appVersion"` //程序版本号
	AppVerMark string `toml:"appVerMark"` //版本说明

	//http配置
	Http Http `toml:"http"`

	//pprof配置
	Pprof Http `toml:"pprof"`

	/******以下为自动判断 生成配置******/
	configPath string //程序配置文件所在目录

	//**********以下为可实时更新项***********//
	Encrypt          Encrypt         `toml:"encrypt"`         //http请求加密处理 秘钥支持实时更新 当前仅支持md5,加密方式不支持其他
	Env              string          `toml:"env"`             //发布dev   测试test
	IgnoreSignCheck  string          `toml:"ignoreSignCheck"` //忽略签名检查的类
	ignoresSignClass map[string]bool //map存储忽略签名检查的类 方便查询
	IpCheck          bool            `toml:"ipCheck"` //是否开启IP检查
	IpPass           string          `toml:"ipPass"`  //指定允许通过的IP
	ipPass           map[string]bool //map存储 指定允许通过的IP列表 方便查询
	IgnoreIpCheck    string          `toml:"ignoreIpCheck"` //忽略IP检查的类
	ignoreIpClass    map[string]bool //map存储忽略IP检查的类 方便查询
	StaticFiles      []StaticConfig  `toml:"staticFiles"`
	//**********允许实时更新项***********//
}

func Info() *Config {
	return config
}

// IgnoreSign 是否忽略签名检查
func IgnoreSign(className string) bool {
	//如果为测试环境 直接通过
	if config.Env == "test" {
		return true
	}

	if _, ok := config.ignoresSignClass[className]; ok {
		return true
	}

	return false
}

// IpPassCheck 检查IP是否允许通过
// 如果为测试环境 则内网IP 直接通过
func IpPassCheck(ip string, className string) bool {
	//如果为测试环境 或内网IP 则直接通过
	if config.Env == "test" || cFunc.InnerIP(ip) {
		return true
	}

	if config.IpCheck {
		//可通过列表
		if _, ok := config.ipPass[ip]; ok {
			return true
		}

		//忽略列表
		if config.ignoreIpCheckClass(className) {
			return true
		}

		return false
	}

	return true
}

// IgnoreIp 是否忽略ip检查
func (c *Config) ignoreIpCheckClass(className string) bool {
	if _, ok := c.ignoreIpClass[className]; ok {
		return true
	}

	return false
}

// New 新建配置信息
func New(configFileName string) *Config {
	cc := &Config{
		ipPass:           make(map[string]bool, 0),
		ignoresSignClass: make(map[string]bool, 0),
		ignoreIpClass:    make(map[string]bool, 0),
	}

	var err error
	cc.configPath, err = cFunc.LoadConfig(configFileName, cc)
	if err != nil {
		mLog.Fatal("未能加载配置文件", err)
	}

	if err = cc.checkParam(); err != nil {
		mLog.Fatal(err)
	}

	//包含一次冗余调用
	fileMonitor.New(cc.configPath+configFileName, func(i interface{}) {
		resetConfig(cc, configFileName)
	})

	return cc
}

// 检查设置的参数 是否正确
func (c *Config) checkParam() error {
	//检查http参数
	if c.Http.HTTP {
		if c.Http.PORT == "" {
			return errors.New("http port不能为空")
		}
		if c.Http.HTTPS {
			if c.Http.HTTPSPEM == "" || c.Http.HTTPSKEY == "" {
				return errors.New("https证书不能为空")
			}

			c.Http.HTTPSKEY = c.configPath + c.Http.HTTPSKEY
			c.Http.HTTPSPEM = c.configPath + c.Http.HTTPSPEM

			if _, err := os.Stat(c.Http.HTTPSKEY); err != nil {
				return errors.New("没找到配置的证书文件" + err.Error())
			}

			if _, err := os.Stat(c.Http.HTTPSPEM); err != nil {
				return errors.New("没找到配置的证书文件" + err.Error())
			}
		} else {
			c.Http.HTTPSKEY = ""
			c.Http.HTTPSPEM = ""
		}
	}

	//检查pprof参数
	if c.Pprof.HTTP {
		if c.Pprof.PORT == "" {
			return errors.New("http port不能为空")
		}

		if c.Pprof.HTTPS {
			if c.Pprof.HTTPSPEM == "" || c.Pprof.HTTPSKEY == "" {
				return errors.New("pprof https证书不能为空")
			}

			c.Pprof.HTTPSKEY = c.configPath + c.Pprof.HTTPSKEY
			c.Pprof.HTTPSPEM = c.configPath + c.Pprof.HTTPSPEM

			if _, err := os.Stat(c.Pprof.HTTPSKEY); err != nil {
				return errors.New("没找到配置的证书文件" + err.Error())
			}

			if _, err := os.Stat(c.Pprof.HTTPSPEM); err != nil {
				return errors.New("没找到配置的证书文件" + err.Error())
			}
		} else {
			c.Pprof.HTTPSPEM = ""
			c.Pprof.HTTPSKEY = ""
		}
	}

	return nil
}

func resetConfig(con *Config, configFileName string) {
	var err error
	cc := &Config{}
	cc.configPath, err = cFunc.LoadConfig(configFileName, cc)
	if err != nil {
		mLog.Error("未能加载配置文件", err)
	}

	if err = cc.checkParam(); err != nil {
		mLog.Error(err)
	}

	//修改可更改的配置项
	con.AppName = cc.AppName
	con.AppVersion = cc.AppVersion
	con.AppVerMark = cc.AppVerMark

	con.Env = cc.Env

	con.Encrypt = cc.Encrypt
	con.IgnoreSignCheck = cc.IgnoreSignCheck

	con.IpCheck = cc.IpCheck
	con.IpPass = cc.IpPass
	con.IgnoreIpCheck = cc.IgnoreIpCheck

	con.StaticFiles = cc.StaticFiles

	//忽略签名检查的类
	iSc := strings.Split(con.IgnoreSignCheck, ",")
	for _, v := range iSc {
		s := strings.TrimSpace(v)
		if s != "" {
			con.ignoresSignClass[s] = true
		}
	}

	//ip白名单
	iBip := strings.Split(con.IpPass, ",")
	for _, v := range iBip {
		s := strings.TrimSpace(v)
		if s != "" {
			con.ipPass[s] = true
		}
	}

	//忽略IP检查的类
	iSip := strings.Split(con.IgnoreIpCheck, ",")
	for _, v := range iSip {
		s := strings.TrimSpace(v)
		if s != "" {
			con.ignoreIpClass[s] = true
		}
	}

	mLog.SetEvn(cc.Env)
}
