package storage

import (
    //"encoding/json"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "os"
	"context"
    "fmt"
    "path/filepath"
    "io/ioutil"
)


func filePathWalkDir(root string) ([]string, error) {

    var files []string
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return files, err
	}
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            files = append(files, path)
        }
        return nil
    })
    return files, err
}

func JsonSyncData(ctx context.Context, dir string, topic *pubsub.Topic) {
    files, err := filePathWalkDir(dir)
    if err != nil {
        fmt.Println(err)
        return
    }
    for _, f := range files{
        fmt.Println(f)
        data, err := ioutil.ReadFile(f)
        if err != nil {
          fmt.Print(err)
        } else {
            textmsg , err1 :=NewTextMessage(data)
            if err1 == nil {
                if textmsg.Message.Status != "OK" { //not published
                        fmt.Println("ok connected")
                        //run publish
                        textmsg.Message.Status = "OK"
                        jsonData, _:= textmsg.ToJson()
                        err = topic.Publish(ctx, jsonData)
                        if err != nil {
                            fmt.Println("publish err")
                            fmt.Println(err)
	                    } else {
                            fmt.Println("publish message success, update the local file status")
                            err := ioutil.WriteFile(f, jsonData, 0644)
                            fmt.Println(err)
                        }
                }
            }
        }
    }

}
