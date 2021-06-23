// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package run

import (
	"os"
	"os/exec"
)

func CmdRun(cmd string) (string, error) {
	var cmd2 *exec.Cmd
	cmd2 = exec.Command(os.Getenv("SHELL"), "-c", cmd)

	msg, err := cmd2.Output()
	return string(msg), err
}
