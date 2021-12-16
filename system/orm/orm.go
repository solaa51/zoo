package orm

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/solaa51/zoo/system/mLog"
	"github.com/solaa51/zoo/system/path"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//自动管理 数据库的连接与更新

func GetDb(dbUName string) (*gorm.DB, error) {
	if _, ok := dbInstances[dbUName]; ok {
		return dbInstances[dbUName].dbIns, nil
	}

	return nil, errors.New("没找到对应数据库示例")
}

// ShowSql 为数据库连接实例开启sql日志
func ShowSql(db *gorm.DB) {
	db.Logger = logger.Default.LogMode(logger.Info)
}

// dbInstances 当前已连接到的数据库实例
var dbInstances map[string]*dbInstance

// dbInstance 单个数据库连接实例
type dbInstance struct {
	mux    sync.Mutex
	dbConf DbConf
	dbIns  *gorm.DB
}

// update 更新实例
func (d *dbInstance) update(conf DbConf, db *gorm.DB) {
	d.mux.Lock()
	defer d.mux.Unlock()
	d.dbConf = conf
	d.dbIns = db
}

// DbConf 数据库配置格式
type DbConf struct {
	UName string `toml:"uName"` //唯一名称标识
	Mark  string `toml:"mark"`  //备注
	Host  string `toml:"host"`
	User  string `toml:"user"`
	Pass  string `toml:"pass"`
	Port  string `toml:"port"`
	Name  string `toml:"name"` //数据库名称
}
type DbConfigParse struct {
	Dbs []DbConf `toml:"dbConfig"`
}

func init() {
	configDir, err := path.ConfigsDir("")
	if err != nil {
		mLog.Error(err)
		return
	}

	dbFileName = configDir + "database.toml"
	if _, err := os.Stat(dbFileName); err != nil {
		mLog.Error(err)
		return
	}

	//初始化 数据库连接池
	dbInstances = make(map[string]*dbInstance, 0)

	initDbs()

	//开启数据库配置文件监控
	go autoUpdateDbInstance()
}

// 数据库配置文件名称
var dbFileName string

func initDbs() {
	dbConfigParse := &DbConfigParse{}
	_, err := toml.DecodeFile(dbFileName, dbConfigParse)
	if err != nil {
		mLog.Error("读取数据库配置文件出错", err)
		return
	}

	for _, v := range dbConfigParse.Dbs {
		if dd, ok := dbInstances[v.UName]; ok { //已存在连接
			//判断是否有变化
			if dd.dbConf.Host != v.Host || dd.dbConf.Pass != v.Pass || dd.dbConf.Port != v.Port || dd.dbConf.User != v.User || dd.dbConf.Name != v.Name {
				db, err := linkDb(v)
				if err != nil {
					mLog.Error("连接数据库-"+v.Mark+"-失败：", err.Error())
					continue
				}
				dd.update(v, db)
			}
		} else {
			db, err := linkDb(v)
			if err != nil {
				mLog.Error("连接数据库-"+v.Mark+"-失败：", err.Error())
				continue
			}

			dbInstances[v.UName] = &dbInstance{
				dbConf: v,
				dbIns:  db,
			}
		}
	}
}

// 连接数据库
func linkDb(conf DbConf) (*gorm.DB, error) {
	dsn := conf.User + ":" + conf.Pass + "@tcp(" + conf.Host + ":" + conf.Port + ")/" + conf.Name + "?charset=utf8"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, //使用单表名
		},
		Logger: logger.New(&slowSqlWrite{}, logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			Colorful:      true,
			LogLevel:      logger.Warn,
		}),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// slowSqlWrite 自定义日志输出 存放到日志文件中 根据日志存储等级以及慢日志规则接收日志信息
type slowSqlWrite struct{}

func (l *slowSqlWrite) Printf(format string, v ...interface{}) {
	logStr := fmt.Sprintf(format, v...)
	mLog.Warn(logStr)
}

// autoUpdateDbInstance 自动更新数据库实例连接
// 1. 定时检查数据库配置文件是否存在
// 2. 当文件出现变更时触发
// 3. 按照db配置中的uName交叉比对 数据库的修改、上线、移除
// autoUpdateDbInstance
func autoUpdateDbInstance() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		mLog.Error("pkg.orm 数据库配置变化监控失败：", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(dbFileName)
	if err != nil {
		mLog.Error("pkg.orm 数据库配置变化监控失败：", err)
		return
	}

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
				mLog.Warn("pkg.orm 数据库配置文件变化，更新数据库连接信息")
				initDbs()
			}
		}
	}
}

/***便捷方法***/

// TableToStruct 将数据库表 转换为struct结构输出
func TableToStruct(dbUName string, tableName string) {
	var d *dbInstance
	if _, ok := dbInstances[dbUName]; ok {
		d = dbInstances[dbUName]
	}

	//表信息
	type TableInfo struct {
		TableName    string `gorm:"column:TABLE_NAME;size:64"`
		TableComment string `gorm:"column:TABLE_COMMENT;size:255"`
	}
	var tInfo []TableInfo
	d.dbIns.Raw("SELECT * FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", d.dbConf.Name, tableName).Scan(&tInfo)

	if len(tInfo) != 1 {
		mLog.Fatal("没有找到对应的表")
	}

	//表字段结构
	type Result struct {
		TableName     string `gorm:"column:TABLE_NAME;size:64"`
		ColumnName    string `gorm:"column:COLUMN_NAME;size:64"`      //列名
		ColumnDefault string `gorm:"column:COLUMN_DEFAULT;size:1024"` //默认值
		IsNullable    string `gorm:"column:IS_NULLABLE;size:3"`       //是否允许为空 yes no
		DataType      string `gorm:"column:DATA_TYPE;size:64"`        //数据精确类型 int tinyint smallint decimal varchar
		CharMaxLen    int64  `gorm:"column:CHARACTER_MAXIMUM_LENGTH"` //字符串允许的最大长度
		NumPre        int64  `gorm:"column:NUMERIC_PRECISION"`        //数字类型最大长度
		NumScale      int64  `gorm:"column:NUMERIC_SCALE"`            //decimal类型 后面的精度
		ColumnType    string `gorm:"column:COLUMN_TYPE;size:50"`      //列类型
		ColumnKey     string `gorm:"column:COLUMN_KEY;size:3"`        //主键 唯一等 pri uni mul
		Extra         string `gorm:"column:EXTRA;size:30"`            //自增 auto_increment
		ColumnComment string `gorm:"column:COLUMN_COMMENT;size:255"`  //备注
	}

	var result []Result
	d.dbIns.Raw("SELECT * FROM information_schema.columns a WHERE table_schema = ? AND table_name = ?", d.dbConf.Name, tableName).Scan(&result)

	fmt.Println("// " + convert(tInfo[0].TableName) + " " + tInfo[0].TableComment)
	fmt.Println("type " + convert(tInfo[0].TableName) + " struct {")

	for _, v := range result {
		str := ""
		str += "    " + convert(v.ColumnName) //字段名

		//gorm属性
		var g []string
		g = append(g, "column:"+v.ColumnName)

		comment := "\t\t//"

		switch v.DataType {
		case "int", "tinyint", "bigint", "smallint":
			str += "\tint64\t`json:\"" + v.ColumnName + "\" gorm:\""
			if v.NumPre > 0 {
				g = append(g, "size:"+strconv.FormatInt(v.NumPre, 10))
			}

			//检查default
			if v.ColumnDefault != "" {
				g = append(g, "default:"+v.ColumnDefault)
			}
		case "float", "double", "decimal": //精度问题
			str += "\tfloat64\t`json:\"" + v.ColumnName + "\" gorm:\""
			g = append(g, "type:"+v.ColumnType)
			//检查default
			if v.ColumnDefault != "" {
				g = append(g, "default:"+v.ColumnDefault)
			}
		case "varchar", "char":
			str += "\tstring\t`json:\"" + v.ColumnName + "\" gorm:\""
			g = append(g, "size:"+strconv.FormatInt(v.CharMaxLen, 10))
			//检查default
			if v.ColumnDefault != "" {
				g = append(g, "default:"+v.ColumnDefault)
			}
		case "text", "longtext":
			str += "\tstring\t`json:\"" + v.ColumnName + "\" gorm:\""
			g = append(g, "size:"+strconv.FormatInt(v.CharMaxLen, 10))
		case "datetime":
			str += "\tstring\t`json:\"" + v.ColumnName + "\" gorm:\""
			g = append(g, "size:20")
		default:
			mLog.Fatal("暂未支持的类型-快快修改工具源码：", v.DataType)
		}

		//主键
		if v.ColumnKey == "pri" {
			g = append(g, "primaryKey")
		}

		//自增
		if v.Extra == "auto_increment" {
			g = append(g, "autoIncrement")
		}

		comment += "" + v.ColumnType + " " + v.ColumnComment

		//构建一行列数据
		gs := strings.Join(g, ";")

		fmt.Println(str + gs + "\"`" + comment)
	}

	fmt.Println("}")

	fmt.Println("func (" + convert(tInfo[0].TableName) + ") TableName() string {")
	fmt.Println("    return \"" + tInfo[0].TableName + "\"")
	fmt.Println("}")
}

//将数据库中的表字段 转换为go中使用的名称
func convert(col string) string {
	var s string
	//s = strings.ToUpper(col[:1]) + col[1:]
	flag := 0
	for k, v := range col {
		if k == 0 {
			s += strings.ToUpper(string(v))
			flag = 0
		} else {
			if v == '_' {
				flag = 1
			} else {
				if flag == 1 {
					s += strings.ToUpper(string(v))
					flag = 0
				} else {
					s += string(v)
					flag = 0
				}
			}
		}
	}

	return s
}
