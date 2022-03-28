# zoo
go语言文本精简框架

tag:

v1.0.0----待支持

    整体增加对jsonRpc的支持，http服务中融合rpc，外部通过json数据格式调用

    路由增加rpc服务的注册

    rpc服务header信息允许对调用鉴权验证
    

启动流程：
    1. 加载日志mLog
        日志按日期拆分 最长保留7天

    2. 加载自定义路由 和 配置的handler
        router.CustomRouter 这名字与目录不是重点，可自由更改。主要是为了加载init方法

    3. 自定义路由router.AddCompile增加规则
            初始化路由router并加载自定义匹配规则
            允许重定向
            支持正则匹配参数

    4. 增加路由与控制器的handler映射 控制器必须继承control.Controller 否则无法自动调用
            初始化handler并配置映射关系
            handler.AddCompile

    5. 调用gHttp.Start 此处可自定义端口 可用于覆盖配置文件中的端口

    6. 加载config配置信息 绝大部分信息修改后会触发系统内自更新-及时生效
        app基本信息
        http服务配置信息
        pprof监控配置信息
        Encrypt加密验签配置
        ip检查
        ip白名单
        忽略ip检查的class
        葫芦签名检查的class

    7.  判断执行命令是否包含参数-d 是否进入后台运行

    8.  判断执行命令是否包含参数-g 是否热重启 -g参数由系统自动判断

    9.  goroutine中启动http服务
        goroutine中启动pprof服务

    10. goroutine中启动APP可执行文件监听 如有更新则 -g 热重启

    11. 主线程启动信号监听SIGHUP
    
    12. 增加对静态文件的访问支持
