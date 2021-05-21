package testnode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"syscall"
	"time"
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

func RequestAPI(apiurl string, endpoint string, method string, data string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", apiurl, endpoint)
	switch method {
	case "GET":
		log.Printf("%s %s", method, url)

		req, err := http.NewRequest("GET", url, bytes.NewBufferString(data))
		if err != nil {
			return []byte(""), err
		}
		req.Header.Add("Content-Type", "application/json")
		client := &http.Client{}

		//resp, err := http.Get(url)
		resp, err := client.Do(req)
		if err != nil {
			return []byte(""), err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte(""), err
		}
		return body, nil
	case "POST":
		log.Printf("%s %s", method, url)
		resp, err := http.Post(url, "application/json", bytes.NewBufferString(data))

		if err != nil {
			return []byte(""), err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte(""), err
		}
		return body, nil
	}
	return []byte(""), nil
}

func CheckNodeRunning(ctx context.Context, url string) bool {
	apiurl := fmt.Sprintf("%s/api/v1", url)
	fmt.Printf("checkNodeRunning: %s\n", apiurl)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return false
		case <-ticker.C:
			resp, err := RequestAPI(apiurl, "/node", "GET", "")
			if err == nil {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					fmt.Println(err)
				} else {
					if objmap["node_publickey"] != "" {
						ticker.Stop()
						return true
					}
				}
			}
		}
	}
}
