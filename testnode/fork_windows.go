// +build windows

package testnode

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

func Fork(pidch chan int, keystorepassword string, cmdName string, cmdArgs ...string) {
	go func() {
		command := exec.Command(cmdName, cmdArgs...)

		var stdout, stderr []byte
		var errStdout, errStderr error
		stdoutIn, _ := command.StdoutPipe()
		stderrIn, _ := command.StderrPipe()

		command.Env = append(os.Environ(),
			"RUM_KSPASSWD="+keystorepassword,
		)

		log.Printf("run command: %s", command)
		err := command.Start()
		if err != nil {
			log.Println(err, string(stderr))
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
			wg.Done()
		}()

		stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
		wg.Wait()

		if errStdout != nil || errStderr != nil {
			log.Fatal("failed to capture stdout or stderr\n")
		}
		outStr, errStr := string(stdout), string(stderr)
		fmt.Printf("\nout:\n%s\nerr:\n%s\n", outStr, errStr)

		pidch <- command.Process.Pid
	}()
}

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
			if err != nil {
				return out, err
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
}
