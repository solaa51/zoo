package orm

import (
	"errors"
	"github.com/BurntSushi/toml"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"zoo/system/cFunc"
	"zoo/system/mLog"
)

/*
数据库连接与初始化配置
*/

// dbs 数据库的连接
type dbConfig struct {
	db     *gorm.DB
	dbName string //数据库名
}

var dbs []*dbConfig

// Get 获取数据库连接实例
func Get(dbIndex int) (*gorm.DB, error) {
	if dbIndex >= 0 && len(dbs) > dbIndex {
		return dbs[dbIndex].db, nil
	} else {
		return nil, errors.New("未配置的数据库信息")
	}
}

// ShowSql 为数据库连接实例开启sql日志
func ShowSql(db *gorm.DB) {
	db.Logger = logger.Default.LogMode(logger.Info)
}

func ShowSqlIndex(dbIndex int) {
	if dbIndex >= 0 && len(dbs) > dbIndex {
		dbs[dbIndex].db.Logger = logger.Default.LogMode(logger.Info)
	}
}

func init() {
	dbFileName := "database.toml"
	dbConfigPath, err := cFunc.FindConfigPath(dbFileName)
	if err != nil {
		mLog.Fatal("没有找到数据库配置文件信息", dbFileName)
	}

	// Db 数据库配置格式
	type Db struct {
		Mark string `toml:"mark"` //备注
		Host string `toml:"host"`
		User string `toml:"user"`
		Pass string `toml:"pass"`
		Port string `toml:"port"`
		Name string `toml:"name"`
	}
	type DbConfigParse struct {
		Dbs []Db `toml:"dbConfig"`
	}

	dbConfigParse := &DbConfigParse{}
	_, err = toml.DecodeFile(dbConfigPath+dbFileName, dbConfigParse)
	if err != nil {
		mLog.Fatal("读取数据库配置文件出错", err)
	}

	for _, v := range dbConfigParse.Dbs {
		dsn := v.User + ":" + v.Pass + "@tcp(" + v.Host + ":" + v.Port + ")/" + v.Name + "?charset=utf8"

		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true, //使用单表名
			},
		})
		if err != nil {
			mLog.Fatal("连接数据库-"+v.Mark+"-失败：", err.Error())
		}

		//db.AutoMigrate()

		dbs = append(dbs, &dbConfig{
			db:     db,
			dbName: v.Name,
		})
	}
}
