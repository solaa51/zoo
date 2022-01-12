package path

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

/**
处理跟执行入口有关的路径问题
*/

// HomeDir 解析程序执行根目录 包含os.PathSeparator
func HomeDir() (string, error) {
	var dir string
	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	if err != nil {
		return "", err
	}

	//如果路径中 含有 /T/go-build 字符 则可认为是 go run 下执行的临时程序
	switch runtime.GOOS {
	case "darwin":
		if strings.Contains(dir, "/T/go-build") {
			dir, _ = os.Getwd()
		}

		return dir + string(os.PathSeparator), nil
	case "windows":
		if strings.Contains(dir, "\\Temp\\go-build") {
			dir, _ = os.Getwd()
		}
		return dir + string(os.PathSeparator), nil
	default:
	}

	return dir + string(os.PathSeparator), nil
}

// ConfigsDir 解析配置文件所在的目录 名称默认为configs
//
// 包含os.PathSeparator
//
// 从下往上依次查找
//
// ConfigsDir windows不能直接放在分区的一级目录
func ConfigsDir(configsDirName string) (string, error) {
	dir, err := HomeDir()
	if err != nil {
		return "", err
	}

	dirSplit := strings.Split(dir, string(os.PathSeparator))

	//配置目录名称
	if configsDirName == "" {
		configsDirName = "configs"
	}

	l := len(dirSplit)
	for i := l - 1; i > 1; i-- { //不能直接放在分区的一级目录
		tmpDir := strings.Join(dirSplit[:i], string(os.PathSeparator))
		if _, err = os.Stat(tmpDir + string(os.PathSeparator) + configsDirName); err == nil {
			return tmpDir + string(os.PathSeparator) + configsDirName + string(os.PathSeparator), nil
		}
	}

	return "", errors.New("没有找到配置目录:" + configsDirName)
}
