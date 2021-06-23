package run

import (
	"bytes"
	"errors"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"time"
)

//用来执行 linux 命令
//需要完善

//采用ssh密钥 登录服务器
type sshRun struct {
	privateKey string
	host       string

	client *ssh.Client
}

func NewCmdRun() (*sshRun, error) {
	var c sshRun

	//从配置文件获取 私钥信息
	key, err := ioutil.ReadFile("conf/sshPrivateKey.txt")
	if err != nil {
		return nil, errors.New("从配置文件读取私钥出错:" + err.Error())
	}

	c.host = "120.26.208.95:22"

	//构建 ssh 登录验证方式 密钥校验
	auth := make([]ssh.AuthMethod, 0)
	signer, _ := ssh.ParsePrivateKey(key)
	auth = append(auth, ssh.PublicKeys(signer))

	//设置连接配置信息
	cConfig := &ssh.ClientConfig{
		User:    "root", //ssh登录用户名
		Auth:    auth,   //ssh验证方式 公钥私钥 / 密码验证
		Timeout: 10 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", c.host, cConfig)
	if err != nil {
		return nil, errors.New("连接到服务器出错：" + err.Error())
	}

	c.client = client

	return &c, nil
}

//执行
func (r *sshRun) Run(command string) error {

	s, err := r.client.NewSession()
	if err != nil {
		return errors.New("生成客户端会话出错：" + err.Error())
	}

	defer s.Close()

	var stdOut, stdErr bytes.Buffer

	s.Stdout = &stdOut
	s.Stderr = &stdErr

	err = s.Run(command)

	//fmt.Println("命令输出内容：" + stdOut.String())
	//fmt.Println("命令错误内容：" + stdErr.String())

	return err
}
