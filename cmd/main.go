package main

import (
	"fmt"
	"flag"
	"path/filepath"
	"github.com/spf13/viper"
	"github.com/golang/glog"
)

var (
	rootDir string
)

func loadconf() {
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.AddConfigPath(filepath.Dir("."))
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.ReadInConfig()
	rootDir = viper.GetString("ROOT_DIR")
}

func main() {
	flag.Parse()
	glog.V(2).Infof("Start...")
	loadconf()
    fmt.Println(rootDir)
}
