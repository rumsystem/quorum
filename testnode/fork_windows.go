// +build windows

package testnode

import (
	"bytes"
	"log"
	"os/exec"
)

func Fork(pidch chan int, cmdName string, cmdArgs ...string) {
	go func() {
		var stderr bytes.Buffer
		command := exec.Command(cmdName, cmdArgs...)
		log.Printf("run command: %s", command)
		command.Stderr = &stderr
		err := command.Start()
		if err != nil {
			log.Println(err, stderr.String())
		}
		pidch <- command.Process.Pid
	}()
}
