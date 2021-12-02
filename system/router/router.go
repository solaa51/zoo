package router

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

/*
	可配置自定义路由规则
	解析出class method params
	"a/b/(\d+)/(/d+)" = "welcome/index/$1/$2" //welcome-Index($1, $2)
	routerCheck = router.NewRouter()
	routerCheck.SetDefaultClassMethod("welcome", "index")
	routerCheck.AddCompile(`welcome/(\w+)/(\w+)`, "welcome/index/$1/$2")
	routerCheck.AddCompile(`welcome/index`, "welcome/index")
*/

// 默认路由
var router = New()

type regRule struct {
	key string
	val string
}

// Router 路由规则设置
type Router struct {
	compile           []*regRule //路由正则规则
	defaultClassName  string     //默认控制器
	defaultMethodName string     //默认方法
}

func New() *Router {
	return &Router{
		compile: make([]*regRule, 0),
	}
}

// AddCompile 设置路由规则
func AddCompile(key, value string) {
	router.compile = append(router.compile, &regRule{
		key: key,
		val: value,
	})
}

/*// Compiles 返回路由规则
func Compiles() map[string]string {
	return router.compile
}*/

// SetDefaultClassMethod 设置 默认控制器和默认方法
func SetDefaultClassMethod(className, methodName string) {
	router.defaultClassName = className
	router.defaultMethodName = methodName
}

// ParseRoute 根据路由规则返回 控制器 方法 参数
func ParseRoute(request *http.Request) (string, string, []string) {
	urlPath := ""
	if strings.HasSuffix(request.URL.Path, "/") {
		urlPath = request.URL.Path[0 : len(request.URL.Path)-1]
	} else {
		urlPath = request.URL.Path
	}

	if strings.HasPrefix(urlPath, "/") {
		urlPath = urlPath[1:]
	}

	//初始化返回值
	className := ""
	methodName := ""
	args := make([]string, 0)

	//可以用来处理正则匹配路由
	for _, v := range router.compile {
		reg := regexp.MustCompile(`^` + v.key + `$`)
		matchs := reg.FindStringSubmatch(urlPath)
		if len(matchs) > 0 { //匹配到了
			urlPath = v.val
			p := regexp.MustCompile(`\$\d`)
			b := p.FindAllString(v.val, -1)
			for _, i := range b {
				ij, _ := strconv.Atoi(i[1:])
				urlPath = strings.ReplaceAll(urlPath, i, matchs[ij])
			}
			break
		}
	}

	splitUri := strings.Split(urlPath, "/")

	switch len(splitUri) {
	case 0:
	case 1:
		className = splitUri[0]
	case 2:
		className = splitUri[0]
		methodName = strings.ToUpper(splitUri[1][:1]) + splitUri[1][1:]
	default:
		className = splitUri[0]
		methodName = strings.ToUpper(splitUri[1][:1]) + splitUri[1][1:]
		for k, v := range splitUri {
			if k > 1 {
				args = append(args, v)
			}
		}
	}

	if className == "" {
		className = router.defaultClassName
	}

	if methodName == "" {
		methodName = router.defaultMethodName
	}

	return className, methodName, args
}
