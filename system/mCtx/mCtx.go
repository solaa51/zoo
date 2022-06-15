package mCtx

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/solaa51/zoo/system/cFunc"
	"github.com/solaa51/zoo/system/config"
	"github.com/solaa51/zoo/system/library/snowflake"
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
	RequestId      string //分布式唯一ID

	Header http.Header

	ClassName  string
	MethodName string

	Post     url.Values //单纯的form-data请求数据 或者x-www-form-urlencoded请求数据
	GetPost  url.Values //get参数与 form-data或者x-www-form-urlencoded合集
	BodyData []byte     //body内包含的数据

	CommonParam CommonParam //公共参数 验证签名的请求使用
	YewuParam   YewuParam   //业务参数 验证签名的请求使用
	Node        *snowflake.Node
}

func New(w http.ResponseWriter, r *http.Request, className string, methodName string) (*Con, error) {
	ctx := &Con{
		Request:        r,
		ClassName:      className,
		MethodName:     methodName,
		ResponseWriter: w,
		RequestId:      config.Info().ServerNode.NextIdStr(),
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

	mLog.Info("访问记录：[" + ctx.RequestId + "] start -- " + ctx.ClassName + "/" + ctx.MethodName)

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
// def 默认值
func (c *Con) CheckParamString(param string, dec string, request bool, min int64, max int64, def string) (string, error) {
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
			tmp = def
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
func (c *Con) CheckParamInt(param string, dec string, request bool, min int64, max int64, def string) (int64, error) {
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
			tmp = def
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
func (c *Con) YewuParamInt(param string, dec string, request bool, min int64, max int64, def string) (int64, error) {
	if config.Info().Env == "test" && c.YewuParam == nil { //本地环境时切换处理函数
		return c.CheckParamInt(param, dec, request, min, max, def)
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
			paramInt, _ = strconv.ParseInt(def, 10, 64)
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
func (c *Con) YewuParamString(param string, dec string, request bool, min int64, max int64, def string) (string, error) {
	if config.Info().Env == "test" && c.YewuParam == nil { //本地环境时切换处理函数
		return c.CheckParamString(param, dec, request, min, max, def)
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
			ps = def
		}
	} else {
		ps = def
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

	header := c.ResponseWriter.Header()
	header.Set("Content-Type", "application/json;charset=UTF-8")

	switch data.(type) {
	case string:
		if data.(string) == "" { //空字符串 替换为空struct
			data = struct{}{}
		}
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

	if code != 0 {
		b, err := jsoniter.Marshal(st)
		if err != nil {
			mLog.Error("访问记录：["+c.RequestId+"] end -- "+c.ClassName+"/"+c.MethodName, err.Error())
			panic("json序列化报错")
		}
		_, err = c.ResponseWriter.Write(b)
		if err != nil {
			mLog.Error("访问记录：["+c.RequestId+"] end -- "+c.ClassName+"/"+c.MethodName, err.Error())
			panic("写入response报错")
		}

		mLog.Info("访问记录：["+c.RequestId+"] end -- "+c.ClassName+"/"+c.MethodName, string(b))
	}

	panic(JSONRETURN)
}

//用于批量检查参数合法性
type checkType int

const (
	CHECK_INT    checkType = 0
	CHECK_STRING checkType = 1
)

// CheckField 用于检查数据的结构信息
// checkType 当前仅支持int64 和 string
// max = 0 表示不限制长度/大小
// reg数据校验，多个用|分割
// 待支持：mobile email ...
type CheckField struct {
	Name    string
	Dec     string
	Tpe     checkType
	Request bool   //是否必填
	Def     string //默认值
	Min     int64  //最小值或最小长度
	Max     int64  //最大值或最大长度
	Yewu    bool   //是否为业务参数
	Reg     string //正则规则校验
}

// CheckField 批量检查参数是否合法
func (c *Con) CheckField(fields []*CheckField) (map[string]interface{}, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	ret := make(map[string]interface{}, 0)
	var tmp interface{}
	var err error
	for _, v := range fields {
		if v.Yewu {
			switch v.Tpe {
			case CHECK_INT:
				tmp, err = c.YewuParamInt(v.Name, v.Dec, v.Request, v.Min, v.Max, v.Def)
			case CHECK_STRING:
				tmp, err = c.YewuParamString(v.Name, v.Dec, v.Request, v.Min, v.Max, v.Def)
			default:
				return nil, errors.New("暂不支持的数据类型检测")
			}
		} else {
			switch v.Tpe {
			case CHECK_INT:
				tmp, err = c.CheckParamInt(v.Name, v.Dec, v.Request, v.Min, v.Max, v.Def)
			case CHECK_STRING:
				tmp, err = c.CheckParamString(v.Name, v.Dec, v.Request, v.Min, v.Max, v.Def)
			default:
				return nil, errors.New("暂不支持的数据类型检测")
			}
		}

		if err != nil {
			return nil, err
		}

		ret[v.Name] = tmp
	}

	return ret, nil
}
