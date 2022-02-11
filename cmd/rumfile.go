package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	//quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"io"
	"log"
	"os"
)

var (
	ReleaseVersion string
	GitCommit      string
	ApiPrefix      string
)

const ChunkSize int = 150 * 1024

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
	fmt.Printf("download...%s from group %s with  %s\n", trxid, groupid, ApiPrefix)
	return nil
}

func Upload() error {
	return nil
}

type Segmentinfo struct {
	Id     string
	Sha256 string
}

type Fileinfo struct {
	MediaType string
	Name      string
	Title     string
}

func WriteFileinfo(segmentpath string, filehash string, fileinfo *Fileinfo, Segmentinfolist *[]Segmentinfo) {
	builder := strings.Builder{}
	fmt.Println(Segmentinfolist)
	builder.WriteString("{")
	builder.WriteString(fmt.Sprintf(`"mediaType":"%s",`, fileinfo.MediaType))
	builder.WriteString(fmt.Sprintf(`"name":"%s",`, fileinfo.Name))
	builder.WriteString(fmt.Sprintf(`"title":"%s",`, fileinfo.Title))
	builder.WriteString(fmt.Sprintf(`"sha256":"%s",`, filehash))
	builder.WriteString(`"segments":[`)
	for i, info := range *Segmentinfolist {
		builder.WriteString(fmt.Sprintf(`{"id":"%s", "sha256":"%s"}`, info.Id, info.Sha256))
		if i+1 < len(*Segmentinfolist) {
			builder.WriteString(",")
		}
	}
	builder.WriteString(`]}`)
	builder.WriteString("}")
	err := WriteToFile(segmentpath, "fileinfo", builder.String())
	fmt.Println(err)
}

func FileToSegments(filename string, fileinfo *Fileinfo, tmpdir string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	segfileinfolist := []Segmentinfo{}

	segmentpath := filepath.Join(tmpdir, fmt.Sprintf("%s-segs", path.Base(filename)))
	log.Println("Splitting %s ...", filename)

	log.Println("Create a temp dir %s ...", segmentpath)
	_, err = os.Stat(segmentpath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(segmentpath, 0770)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	filehash := sha256.New()
	r := bufio.NewReader(f)
	nBytes, nChunks := int64(0), int64(0)
	buf := make([]byte, 0, ChunkSize)
	for {
		n, err := r.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		nChunks++
		nBytes += int64(len(buf))
		// process buf
		filehash.Write(buf)
		bufhash := sha256.Sum256(buf)
		bufhashhex := hex.EncodeToString(bufhash[:])
		data := base64.StdEncoding.EncodeToString(buf)

		segname := fmt.Sprintf("seg-%d", nChunks)
		log.Printf("save base64 file segment size %d (base64ed size %d) to %s", len(buf), len(data), segname)
		err = WriteToFile(segmentpath, segname, data)
		if err == nil {
			seginfo := &Segmentinfo{Id: segname, Sha256: bufhashhex}
			segfileinfolist = append(segfileinfolist, *seginfo)
		} else {
			log.Fatal(err)
		}
	}
	log.Println("Bytes:", nBytes, "Chunks:", nChunks)

	filesha256 := filehash.Sum(nil)
	filesha256hex := hex.EncodeToString(filesha256[:])
	WriteFileinfo(segmentpath, filesha256hex, fileinfo, &segfileinfolist)
	return nil
}
func WriteToFile(tmpdir string, filename string, data string) error {
	f, err := os.Create(filepath.Join(tmpdir, path.Base(filename)))
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(data)
	return nil
}

func Split() error {
	var filename, tmpdir string
	fs := flag.NewFlagSet("split", flag.ContinueOnError)
	fs.StringVar(&filename, "file", "", "a file name")
	fs.StringVar(&tmpdir, "tmpdir", "/tmp/", "a dir name")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	if len(filename) == 0 {
		fmt.Println("Split a file for the Rum SeedNetwork uploader")
		fmt.Println()
		fmt.Println("Usage:...")
		fs.PrintDefaults()
		return nil
	}

	fileinfo, err := VerifyFileFormat(filename)
	if err != nil {
		return err
	}

	err = FileToSegments(filename, fileinfo, tmpdir)
	if err != nil {
		log.Fatalf("error: %s", err)
		return err
	}
	return nil
}

//prepare files , build the fileinfo ,  post to api
//{"type":"Add","object":{"type":"File","name":"xab","file":{"id":"xab", "compression":"none", "mediaType":"application/octet-stream", "content":""}},"target":{"id":"f1858fba-093b-4ba6-88ed-4dd30576a011","type":"Group"}}' | curl --insecure -X POST -H 'Content-Type: application/json' -d @- https://127.0.0.1:8004/api/v1/group/content
//obj := &quorumpb.Object{Type: "File"}
//post := &quorumpb.Activity{Type: "Add", Object: obj}

func OpenFileInZip(zipfile *zip.ReadCloser, name string) (io.ReadCloser, error) {
	for _, f := range zipfile.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("not exist:%s", name)
}

func ReadFileInZip(zipfile *zip.ReadCloser, filename string) (string, error) {
	fileinzip, err := OpenFileInZip(zipfile, filename)
	if err != nil {
		return "", err
	}
	defer fileinzip.Close()

	content, err := ioutil.ReadAll(fileinzip)
	return string(content), err
}

type Container struct {
	Rootfile Rootfile `xml:"rootfiles>rootfile"`
}

type Rootfile struct {
	Path string `xml:"full-path,attr"`
}

type Contentfile struct {
	Metadata Metadata `xml:"metadata"`
}

type Metadata struct {
	Title string `xml:"title"`
}

func OpenEpub(filename string) (*Fileinfo, error) {
	fd, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	fileinfo := Fileinfo{Name: path.Base(filename)}

	mimetypecontent, err := ReadFileInZip(fd, "mimetype")
	if err != nil {
		return nil, err
	}
	fileinfo.MediaType = string(mimetypecontent)

	xmlcontent, err := ReadFileInZip(fd, "META-INF/container.xml")
	if err != nil {
		return nil, err
	}

	container := &Container{}
	xml.Unmarshal([]byte(xmlcontent), container)

	xmlcontent, err = ReadFileInZip(fd, container.Rootfile.Path)

	if err != nil {
		return nil, err
	}

	contentfile := &Contentfile{}
	xml.Unmarshal([]byte(xmlcontent), contentfile)

	fileinfo.Title = contentfile.Metadata.Title
	return &fileinfo, nil
}

func VerifyFileFormat(filename string) (*Fileinfo, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".epub":
		return OpenEpub(filename)
	default:
		return nil, fmt.Errorf("unsupported file: ", ext)
	}
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
	flag.StringVar(&ApiPrefix, "api", "https://localhost:8000", "api prefix of the rumservice")

	flag.Parse()

	if len(os.Args) < 2 {
		log.Fatalf("error: wrong number of arguments")
	}

	var err error
	if os.Args[1][0] != '-' {
		switch os.Args[1] {
		case "split":
			err = Split()
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

//func HttpPostToGroup(Url string, jsondata []byte) (string, error) {
//	//bearer := "Bearer " + httptask.Taskapijwt
//	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(jsondata))
//	if err != nil {
//		log.Printf("new http request  err: %s", err)
//	}
//	//req.Header.Add("Authorization", bearer)
//	req.Header.Set("Content-Type", "application/json")
//	resp, err := http.DefaultClient.Do(req)
//	if err != nil {
//		log.Printf("post add tasks err: %s", err)
//		return "", false
//	}
//	if resp.StatusCode != 200 {
//		log.Printf("post add tasks err: %s", err)
//		return "", false
//	} else {
//		return "trxid", true
//	}
//}
