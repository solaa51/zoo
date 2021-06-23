处理路由匹配
    
    限流
        规划支持ip限流配置

    可配置自定义路由规则
	"a/b/(\d+)/(/d+)" = "welcome/index/$1/$2" //welcome-Index($1, $2)
	routerCheck = router.NewRouter()
	routerCheck.SetDefaultClassMethod("welcome", "index")
	routerCheck.AddCompile(`welcome/(\w+)/(\w+)`, "welcome/index/$1/$2")
	routerCheck.AddCompile(`welcome/index`, "welcome/index")