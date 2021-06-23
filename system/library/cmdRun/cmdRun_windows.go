package run

import (
	"os/exec"
)

//用来执行本地 命令
func CmdRun(cmd string) (string, error) {
	var cmd2 *exec.Cmd
	cmd2 = exec.Command("cmd", "/C", cmd)

	msg, err := cmd2.Output()
	return string(msg), err
}
