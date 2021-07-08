// +build !windows

package testnode

import (
	"bytes"
	"log"
	"os/exec"
	"syscall"
)

func Fork(pidch chan int, cmdName string, cmdArgs ...string) {
	go func() {
		var stderr bytes.Buffer
		command := exec.Command(cmdName, cmdArgs...)
		log.Printf("run command: %s", command)
		command.Stderr = &stderr
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		err := command.Start()
		if err != nil {
			log.Println(err, stderr.String())
		}
		pidch <- command.Process.Pid
	}()
}
