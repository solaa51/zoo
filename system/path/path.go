package path

import (
	"errors"
	"os"
	"strings"
)

/**
处理跟执行入口有关的路径问题
*/

// HomeDir 解析程序执行根目录 包含os.PathSeparator
func HomeDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
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
