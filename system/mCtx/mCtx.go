package mCtx

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/solaa51/zoo/system/cFunc"
	"github.com/solaa51/zoo/system/config"
	"github.com/solaa51/zoo/system/mLog"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// JSONRETURN 提前退出http请求使用
	JSONRETURN = errors.New("")
	TEXTRETURN = errors.New("")
)

type Con struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter

	Header http.Header

	ClassName  string
	MethodName string

	Post     url.Values //单纯的form-data请求数据 或者x-www-form-urlencoded请求数据
	GetPost  url.Values //get参数与 form-data或者x-www-form-urlencoded合集
	BodyData []byte     //body内包含的数据

	CommonParam CommonParam //公共参数 验证签名的请求使用
	YewuParam   YewuParam   //业务参数 验证签名的请求使用
}

func New(w http.ResponseWriter, r *http.Request, className string, methodName string) (*Con, error) {
	ctx := &Con{
		Request:        r,
		ClassName:      className,
		MethodName:     methodName,
		ResponseWriter: w,
	}

	//解析请求参数 以及body数据
	ctx.parseData()

	if !config.IgnoreSign(ctx.ClassName) {
		//查看是否包含param参数 固定格式
		if ctx.Post["param"] == nil {
			return nil, errors.New("缺少param参数,无法验证签名")
		}
	}

	if ctx.Post["param"] != nil {
		//解析参数
		secret, err := ctx.parseParam(ctx.Post["param"][0])
		if err != nil {
			return nil, err
		}

		err = ctx.signCheck(secret)
		if err != nil {
			return nil, err
		}
	}

	return ctx, nil
}

// 解析签名参数 验证签名
func (c *Con) parseParam(paramData string) (string, error) {
	data := CommonParam{}
	err := jsoniter.Unmarshal([]byte(paramData), &data)
	if err != nil {
		return "", err
	}

	if data.AppKey == "" {
		return "", errors.New("app_key不能为空")
	}

	if data.Control == "" {
		return "", errors.New("control不能为空")
	}

	if data.Method == "" {
		return "", errors.New("method不能为空")
	}

	if data.Ip == "" {
		return "", errors.New("IP不能为空")
	}

	if data.Sign == "" {
		return "", errors.New("签名不能为空")
	}

	secret := ""
	hasKey := false
	for _, v := range config.Info().Encrypt.Keys {
		if v.Key == data.AppKey {
			secret = v.Value
			hasKey = true
			break
		}
	}

	if !hasKey {
		return "", errors.New("无效的key:" + data.AppKey)
	}

	c.CommonParam = data

	yData := YewuParam{}
	err = jsoniter.Unmarshal([]byte(data.Param), &yData)
	if err != nil {
		return "", err
	}
	c.YewuParam = yData

	return secret, nil
}

// sign 生成签名
func (c *Con) signCheck(secret string) error {
	//是否验证签名
	if !config.IgnoreSign(c.ClassName) {
		str := "app_key=" + c.CommonParam.AppKey + "&" +
			"control=" + c.CommonParam.Control + "&" +
			"ip=" + c.CommonParam.Ip + "&" +
			"method=" + c.CommonParam.Method + "&" +
			"param=" + url.QueryEscape(c.CommonParam.Param) + secret

		if c.CommonParam.Sign != cFunc.Md5(str) {
			mLog.Warn(c.CommonParam.Ip + " - " + c.ClassName + "-" + c.MethodName + " 签名错误:" + str)
			return errors.New("签名错误")
		}
	}

	return nil
}

// parseData 解析请求参数 以及body数据
func (c *Con) parseData() {
	_ = c.Request.ParseForm()                  //解析get参数
	_ = c.Request.ParseMultipartForm(32 << 20) //解析post参数

	c.GetPost = c.Request.Form
	c.Post = c.Request.PostForm

	bodyData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		mLog.Error("获取body数据失败", err)
	}
	defer c.Request.Body.Close()

	c.BodyData = bodyData
}

// CommonParam 公共参数
type CommonParam struct {
	AppKey  string `json:"app_key"`
	Control string `json:"control"`
	Method  string `json:"method"`
	Ip      string `json:"ip"`
	Sign    string `json:"sign"`
	Param   string `json:"param"` //业务参数部分
}

// YewuParam 业务参数
type YewuParam = map[string]interface{}

// CheckParamString 检测外部访问来的参数
// GET POST参数检测并转换为string
// dec 参数描述
// request 是否必填
// min 0不判断最小长度
// max 0 强制max = 65535 判断最大长度
func (c *Con) CheckParamString(param string, dec string, request bool, min int64, max int64) (string, error) {
	var tmp string
	if request {
		if c.GetPost[param] == nil {
			return "", errors.New(dec + "不能为空")
		}
		tmp = strings.TrimSpace(c.GetPost[param][0])
		if tmp == "" {
			return "", errors.New(dec + "不能为空格等空字符")
		}
	} else {
		if c.GetPost[param] == nil {
			tmp = ""
		} else {
			tmp = strings.TrimSpace(c.GetPost[param][0])
		}
	}

	//获得字符长度
	num := int64(utf8.RuneCountInString(tmp))

	//判断长度
	if num > 0 && min > 0 {
		if num < min {
			return "", errors.New(dec + "最少" + strconv.FormatInt(min, 10) + "个字")
		}
	}

	if max == 0 { //限定下 数据库存储 普通情况 最大65535
		max = 65535
	}

	if num > max {
		return "", errors.New(dec + "最多" + strconv.FormatInt(max, 10) + "个字")
	}

	return tmp, nil
}

/** CheckParamInt
param 要检查的字段名
dec 字段描述
request 是否必填
min 最小值
max 最大允许值
*/

// CheckParamInt 检测外部访问来的参数 转换为int返回
// GET POST参数检测并转换为string
// dec 字段描述
// request 是否必填
// min 最小值
// max 最大允许值
func (c *Con) CheckParamInt(param string, dec string, request bool, min int64, max int64) (int64, error) {
	var tmp string
	if request { //判断必填
		if c.GetPost[param] == nil {
			return 0, errors.New(dec + "不能为空")
		}
		tmp = strings.TrimSpace(c.GetPost[param][0])
		if tmp == "" {
			return 0, errors.New(dec + "不能为空格等空字符")
		}
	} else {
		if c.GetPost[param] == nil {
			tmp = ""
		} else {
			tmp = strings.TrimSpace(c.GetPost[param][0])
		}
	}

	paramInt, _ := strconv.ParseInt(tmp, 10, 64)

	if paramInt < min {
		return 0, errors.New(dec + "不能小于" + strconv.FormatInt(min, 10))
	}

	if max == 0 { //限定下 数据库int 普通情况 11位 够用即可
		max = 99999999999
	}

	if paramInt > max {
		return 0, errors.New(dec + "不能大于" + strconv.FormatInt(max, 10))
	}

	return paramInt, nil
}

// YewuParamInt 按字段名 获取业务参数
func (c *Con) YewuParamInt(param string, dec string, request bool, min int64, max int64) (int64, error) {
	if config.Info().Env == "test" && c.YewuParam == nil { //本地环境时切换处理函数
		return c.CheckParamInt(param, dec, request, min, max)
	}

	if request {
		if c.YewuParam[param] == nil {
			return 0, errors.New(dec + "不能为空")
		}
	}

	var paramInt int64
	if c.YewuParam[param] != nil {
		switch c.YewuParam[param].(type) {
		case int:
			paramInt = int64(c.YewuParam[param].(int))
		case int64:
			paramInt = c.YewuParam[param].(int64)
		case string:
			paramInt, _ = strconv.ParseInt(strings.TrimSpace(c.YewuParam[param].(string)), 10, 64)
		case float64:
			paramInt = int64(c.YewuParam[param].(float64))
		default:
			paramInt = 0
		}
	} else {
		paramInt = 0
	}

	if paramInt < min {
		return 0, errors.New(dec + "不能小于" + strconv.FormatInt(min, 10))
	}

	if max == 0 {
		max = 9999999999
	}
	if paramInt > max {
		return 0, errors.New(dec + "不能大于" + strconv.FormatInt(max, 10))
	}

	return paramInt, nil
}

// YewuParamString 业务参数检测并转换为string
func (c *Con) YewuParamString(param string, dec string, request bool, min int64, max int64) (string, error) {
	if config.Info().Env == "test" && c.YewuParam == nil { //本地环境时切换处理函数
		return c.CheckParamString(param, dec, request, min, max)
	}

	if request {
		if c.YewuParam[param] == nil {
			return "", errors.New(dec + "不能为空")
		}
	}

	var ps string
	if c.YewuParam[param] != nil {
		switch c.YewuParam[param].(type) {
		case int:
			ps = strconv.Itoa(c.YewuParam[param].(int))
		case int64:
			ps = strconv.FormatInt(c.YewuParam[param].(int64), 10)
		case string:
			ps = strings.TrimSpace(c.YewuParam[param].(string))
		case float64:
			ps = strconv.FormatFloat(c.YewuParam[param].(float64), 'f', -1, 64)
		default:
			ps = ""
		}
	} else {
		ps = ""
	}

	//获得字符长度
	num := int64(utf8.RuneCountInString(ps))

	//判断长度
	if num > 0 && min > 0 {
		if num < min {
			return "", errors.New(dec + "最少" + strconv.FormatInt(min, 10) + "个字")
		}
	}

	if max == 0 {
		max = 65535
	}

	if num > max {
		return "", errors.New(dec + "最多" + strconv.FormatInt(max, 10) + "个字")
	}

	return ps, nil
}

// WebSocket 升级请求为websocket
func (c *Con) WebSocket() (*websocket.Conn, error) {
	upgrade := websocket.Upgrader{}
	upgrade.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := upgrade.Upgrade(c.ResponseWriter, c.Request, nil)
	return conn, err
}

func (c *Con) JsonReturn(code int, data interface{}, format string, a ...interface{}) {
	msg := ""
	if strings.Contains(format, "%s") || strings.Contains(format, "%d") || strings.Contains(format, "%v") || strings.Contains(format, "%t") {
		msg = fmt.Sprintf(format, a)
	} else {
		msg = format
	}

	//json返回数据
	type JsonErr struct {
		Msg  string      `json:"msg"`
		Ret  int         `json:"ret"`
		Data interface{} `json:"data"`
	}

	st := &JsonErr{
		Msg:  msg,
		Ret:  code,
		Data: data,
	}

	b, _ := jsoniter.Marshal(st)
	n, err := c.ResponseWriter.Write(b)
	if err != nil {
		fmt.Println("ERROR:", c.Request.URL, n, err)
		return
	}
	panic(JSONRETURN)
}
