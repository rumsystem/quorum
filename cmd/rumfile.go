package main

import (
	"flag"
	"fmt"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"log"
	"os"
)

var (
	ReleaseVersion string
	GitCommit      string
)

func Download() error {
	var groupid, trxid string
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.StringVar(&groupid, "groupid", "", "group_id of the SeedNetwork")
	fs.StringVar(&trxid, "trxid", "", "trx_id of the fileinfo")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	if len(groupid) == 0 || len(trxid) == 0 {
		fmt.Println("Download a file from the Rum SeedNetwork")
		fmt.Println()
		fmt.Println("Usage:...")
		fs.PrintDefaults()
	}
	fmt.Println("download...")
	return nil
}
func Upload() error {
	var filename string
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.StringVar(&filename, "file", "", "a file name")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	if len(filename) == 0 {
		fmt.Println("Upload a file to the Rum SeedNetwork")
		fmt.Println()
		fmt.Println("Usage:...")
		fs.PrintDefaults()
	}
	fmt.Printf("upload...%s\n", filename)
	return nil
}

func main() {

	if ReleaseVersion == "" {
		ReleaseVersion = "v1.0.0"
	}
	if GitCommit == "" {
		GitCommit = "devel"
	}
	utils.SetGitCommit(GitCommit)
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")

	flag.Parse()

	if len(os.Args) < 2 {
		log.Fatalf("error: wrong number of arguments")
	}

	var err error
	if os.Args[1][0] != '-' {
		switch os.Args[1] {
		case "upload":
			err = Upload()
		case "download":
			err = Download()
		default:
			err = fmt.Errorf("error: unknown command - %s", os.Args[1])
		}
		if err != nil {
			log.Fatalf("error: %s", err)
		}
	}

	if *help {
		fmt.Println("Output a help ")
		fmt.Println()
		fmt.Println("Usage:...")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Printf("%s - %s\n", ReleaseVersion, GitCommit)
		return
	}
}
