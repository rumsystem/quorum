package testnode

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"
)

func Fork(pidch chan int, cmdName string, cmdArgs ...string) {
	go func() {
		command := exec.Command(cmdName, cmdArgs...)
		err := command.Start()
		//output, err := command.Output()
		if err != nil {
			OnError(err)
		}
		pidch <- command.Process.Pid
	}()
}

func OnError(err error) {
	log.Println("Error: %s", err)
}

func RequestAPI(apiurl string, endpoint string, method string, data string) ([]byte, error) {
	switch method {
	case "GET":
		url := fmt.Sprintf("%s%s", apiurl, endpoint)
		log.Printf("%s %s", method, url)
		resp, err := http.Get(url)
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
		//log.Println("post")
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
					if objmap["node_status"] == "NODE_ONLINE" {
						ticker.Stop()
						return true
					}
				}
			}
		}
	}
}
