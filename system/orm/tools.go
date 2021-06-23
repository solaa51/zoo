package orm

import (
	"fmt"
	"github.com/solaa51/zoo/system/mLog"
	"strconv"
	"strings"
)

// TableToStruct 将数据库表 转换为struct结构输出
func TableToStruct(dbIndex int, tableName string) {
	var d *dbConfig
	if dbIndex >= 0 && len(dbs) > dbIndex {
		d = dbs[dbIndex]
	} else {
		mLog.Fatal("调用错误的数据库连接")
	}

	ShowSql(d.db)

	//表信息
	type TableInfo struct {
		TableName    string `gorm:"column:TABLE_NAME;size:64"`
		TableComment string `gorm:"column:TABLE_COMMENT;size:255"`
	}
	var tInfo []TableInfo
	d.db.Raw("SELECT * FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", d.dbName, tableName).Scan(&tInfo)

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
	d.db.Raw("SELECT * FROM information_schema.columns a WHERE table_schema = ? AND table_name = ?", d.dbName, tableName).Scan(&result)

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
