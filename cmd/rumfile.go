package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"net/http"
	"time"
	//"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ReleaseVersion string
	GitCommit      string
	ApiPrefix      string
)

const ChunkSize int = 150 * 1024

func Download() error {
	var groupid, trxid string
	var destdir string
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.StringVar(&groupid, "groupid", "", "group_id of the SeedNetwork")
	fs.StringVar(&trxid, "trxid", "", "trx_id of the fileinfo")
	fs.StringVar(&ApiPrefix, "api", "https://localhost:8000", "api prefix of the rumservice")
	fs.StringVar(&destdir, "dir", ".", "the file segments dir.(the result of split cmd)")

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
	//first, get the fileinfo file
	fileobj, _, err := HttpGetFileFromGroup(ApiPrefix, groupid, trxid)
	if err != nil {
		return err
	}
	if strings.ToLower(fileobj.Type) != "file" || strings.ToLower(fileobj.Name) != "fileinfo" {
		return fmt.Errorf("can't find the fileinfo")
	}

	fileinfo, err := ParseFileinfo(fileobj.File)
	if err != nil {
		return err
	}
	if len(*fileinfo.Segments) == 0 {
		return fmt.Errorf("no file segments in the fileinfo")
	}
	log.Printf("file %s has %d segments, downloading...", fileinfo.Name, len(*fileinfo.Segments))

	f, err := os.Create(filepath.Join(destdir, fileinfo.Name))
	if err != nil {
		return err
	}
	defer f.Close()

	readnexttrxid := trxid
	for _, seg := range *fileinfo.Segments {
		fileobj, trxid, err := HttpGetNextFileFromGroupByTrx(ApiPrefix, groupid, readnexttrxid)
		if err != nil {
			return err
		}
		if fileobj.Name == seg.Id {
			segcontent, err := ParseFileSegment(fileobj)
			if err != nil {
				return err
			}

			//verify sha256 and write
			f.Write(segcontent)
			fmt.Println("read file seg content length:", len(segcontent))
		}
		readnexttrxid = trxid
	}
	f.Close()
	//reopen file and verify file sha256
	return nil
}

func Upload() error {
	var segmentsdir, groupid string
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.StringVar(&segmentsdir, "dir", "", "the file segments dir.(the result of split cmd)")
	fs.StringVar(&groupid, "groupid", "", "the upload target groupid")
	fs.StringVar(&ApiPrefix, "api", "https://localhost:8000", "api prefix of the rumservice")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	if len(segmentsdir) == 0 {
		fmt.Println("Upload a splitted file segments to the Rum SeedNetwork")
		fmt.Println()
		fmt.Println("Usage:...")
		fs.PrintDefaults()
		return nil
	}

	fileinfo, err := VerifySegments(segmentsdir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uploading to ..", ApiPrefix)
	f, err := os.Open(filepath.Join(segmentsdir, "fileinfo"))
	if err != nil {
		return err
	}
	defer f.Close()
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	//prepare files , build the fileinfo ,  post to api
	file := &quorumpb.File{Compression: quorumpb.File_none, MediaType: "application/json", Content: content}
	obj := &quorumpb.Object{Type: "File", Name: "fileinfo", File: file}
	target := &quorumpb.Object{Id: groupid, Type: "Group"}
	post := &quorumpb.Activity{Type: "Add", Object: obj, Target: target}
	trxid, r := PostFileToGroupApi(ApiPrefix, groupid, post)
	if r != nil {
		return fmt.Errorf("post %s failed, err %s", obj.Name, r)
	} else {
		log.Printf("post %s succeed at %s/%s", obj.Name, groupid, trxid)
	}

	for _, seg := range *fileinfo.Segments {
		f, err := os.Open(filepath.Join(segmentsdir, seg.Id))
		if err != nil {
			return err
		}
		defer f.Close()
		content, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		file := &quorumpb.File{Compression: quorumpb.File_none, MediaType: "application/octet-stream", Content: content}
		obj := &quorumpb.Object{Type: "File", Name: seg.Id, File: file}
		target := &quorumpb.Object{Id: groupid, Type: "Group"}
		post := &quorumpb.Activity{Type: "Add", Object: obj, Target: target}
		trxid, r = PostFileToGroupApi(ApiPrefix, groupid, post)
		if r != nil {
			return fmt.Errorf("post %s failed, err: %s", obj.Name, r)
		} else {
			log.Printf("post %s succeed at %s/%s", obj.Name, groupid, trxid)
		}
	}
	return nil
}

func PostFileToGroupApi(apiPrefix, groupid string, post *quorumpb.Activity) (string, error) {
	data, err := json.Marshal(post)
	trxresult, err := HttpPostToGroup(apiPrefix, data)
	if err != nil {
		return "", err
	}
	for {
		result, _ := HttpCheckTrxId(ApiPrefix, groupid, trxresult.TrxId)
		if result == true {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return trxresult.TrxId, nil
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

type Segmentinfo struct {
	Id     string `json:"id"`
	Sha256 string `json:"sha256"`
}

type Fileinfo struct {
	MediaType string         `json:"mediaType"`
	Name      string         `json:"name"`
	Title     string         `json:"title"`
	Sha256    string         `json:"sha256"`
	Segments  *[]Segmentinfo `json:"segments"`
}

type FileItem struct {
	Compression string `json:"compression"`
	MediaType   string `json:"mediaType"`
	Content     string `json:"content"`
}

type TrxResult struct {
	TrxId string `json:"trx_id"`
}

type Trx struct {
	TrxId   string `json:"TrxId"`
	GroupId string `json:"GroupId"`
}

type GroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   quorumpb.Object
	TypeUrl   string
	TimeStamp int64
}

func WriteFileinfo(segmentpath string, fileinfo *Fileinfo) {
	data, err := json.Marshal(fileinfo)
	if err != nil {
		log.Fatal(err)
	}
	err = WriteToFile(segmentpath, "fileinfo", data)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Write fileinfo to %s/%s", segmentpath, "fileinfo")
}

func FileToSegments(filename string, fileinfo *Fileinfo, tmpdir string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	segfileinfolist := []Segmentinfo{}

	segmentpath := filepath.Join(tmpdir, fmt.Sprintf("%s-segs", path.Base(filename)))
	log.Printf("Splitting file: %s ...", filename)
	log.Printf("Create a temp dir %s ...", segmentpath)
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
		//data := base64.StdEncoding.EncodeToString(buf)

		segname := fmt.Sprintf("seg-%d", nChunks)
		//log.Printf("save base64 file segment size %d (base64ed size %d) to %s/%s", len(buf), len(data), segmentpath, segname)
		log.Printf("save file segment size %d  to %s/%s", len(buf), segmentpath, segname)
		err = WriteToFile(segmentpath, segname, buf)
		if err == nil {
			seginfo := &Segmentinfo{Id: segname, Sha256: bufhashhex}
			segfileinfolist = append(segfileinfolist, *seginfo)
		} else {
			log.Fatal(err)
		}
	}
	log.Printf("File bytes: %d, Write Segments:%d", nBytes, nChunks)

	filesha256 := filehash.Sum(nil)
	fileinfo.Sha256 = hex.EncodeToString(filesha256[:])
	fileinfo.Segments = &segfileinfolist

	WriteFileinfo(segmentpath, fileinfo)
	log.Printf("file segments done: %s", segmentpath)
	return nil
}

func WriteToFile(tmpdir string, filename string, data []byte) error {
	f, err := os.Create(filepath.Join(tmpdir, path.Base(filename)))
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write(data)
	return nil
}

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

func VerifySegments(segmentpath string) (*Fileinfo, error) {
	log.Printf("Verify Segments %s ...", segmentpath)
	fi, err := os.Stat(segmentpath)
	if err != nil {
		log.Fatal(err)
	}
	if fi.IsDir() == false {
		log.Fatalf("Error, %s is not a file segments dir", segmentpath)
	}
	//read fileinfo
	f, err := os.Open(filepath.Join(segmentpath, "fileinfo"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var fileinfo Fileinfo
	err = json.Unmarshal(content, &fileinfo)
	if err != nil {
		return nil, err
	}

	for _, seg := range *fileinfo.Segments {
		f, err := os.Open(filepath.Join(segmentpath, seg.Id))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		filehash := sha256.New()
		_, err = io.Copy(filehash, f)
		if err != nil {
			return nil, err
		}
		filesha256 := filehash.Sum(nil)
		sha256hex := hex.EncodeToString(filesha256[:])
		if seg.Sha256 != sha256hex {
			log.Fatalf("File segment %s verify error. expect checksum: %s, but file hash: %s", seg.Id, seg.Sha256, sha256hex)
		}
	}
	return &fileinfo, nil
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

func HttpCheckTrxId(ApiPrefix string, groupid string, trxid string) (bool, error) {
	Url := fmt.Sprintf("%s/api/v1/trx/%s/%s", ApiPrefix, groupid, trxid)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(Url)

	if err != nil {
		log.Printf("HttpCheckTrxId err: %s", err)
		return false, err
	}
	if resp.StatusCode != 200 {
		return false, fmt.Errorf("http err code %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	trx := &Trx{}
	err = json.Unmarshal(body, &trx)
	if trx.TrxId == trxid && groupid == trx.GroupId {
		return true, nil
	}
	return false, nil
}

func HttpPostToGroup(ApiPrefix string, jsondata []byte) (*TrxResult, error) {
	Url := fmt.Sprintf("%s/api/v1/group/content", ApiPrefix)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(jsondata))
	if err != nil {
		log.Printf("new http request  err: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("post add tasks err: %s", err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http err code %d", resp.StatusCode)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		trxresult := &TrxResult{}
		err = json.Unmarshal(body, &trxresult)
		return trxresult, err
	}
}

func HttpGetFromContentApi(ApiPrefix string, groupid string, trxid string, num int, includetrx bool) ([]byte, error) {
	Url := fmt.Sprintf("%s/app/api/v1/group/%s/content?num=1&starttrx=%s&reverse=false&includestarttrx=%t", ApiPrefix, groupid, trxid, includetrx)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer([]byte(`{"senders":[]}`)))
	if err != nil {
		log.Printf("new http request  err: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("post add tasks err: %s", err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http err code %d", resp.StatusCode)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		return body, err
	}
}

func HttpGetNextFileFromGroupByTrx(ApiPrefix string, groupid string, trxid string) (*quorumpb.Object, string, error) {
	body, err := HttpGetFromContentApi(ApiPrefix, groupid, trxid, 1, false)
	if err != nil {
		return nil, "", err
	}
	groupcontentlist := []GroupContentObjectItem{}
	err = json.Unmarshal(body, &groupcontentlist)
	if err != nil {
		return nil, "", err
	}
	if len(groupcontentlist) == 1 {
		contentobj := groupcontentlist[0]
		return &contentobj.Content, contentobj.TrxId, nil
	} else {
		return nil, "", fmt.Errorf("content item more than 1, error")
	}
}

func HttpGetFileFromGroup(ApiPrefix string, groupid string, trxid string) (*quorumpb.Object, string, error) {

	body, err := HttpGetFromContentApi(ApiPrefix, groupid, trxid, 1, true)
	if err != nil {
		return nil, trxid, err
	}
	groupcontentlist := []GroupContentObjectItem{}
	err = json.Unmarshal(body, &groupcontentlist)
	if err != nil {
		return nil, trxid, err
	}
	if len(groupcontentlist) == 1 {
		contentobj := groupcontentlist[0]
		return &contentobj.Content, trxid, nil
	} else {
		return nil, trxid, fmt.Errorf("content item more than 1, error")
	}

	//curl --insecure -X POST -H 'Content-Type: application/json' -d '{"senders":[]}' "https://localhost:8004/app/api/v1/group/a6ceb724-02c3-4c7c-a50e-846389a6e887/content?num=1&starttrx=cac0fca3-6927-43f5-9e5d-3f8c1906ee2e&reverse=false&includestarttrx=true"

}

func ParseFileinfo(fileinfocontent *quorumpb.File) (*Fileinfo, error) {
	var fileinfo *Fileinfo
	err := json.Unmarshal(fileinfocontent.Content, &fileinfo)
	return fileinfo, err
}

func ParseFileSegment(fileobj *quorumpb.Object) ([]byte, error) {
	if fileobj.Type != "File" {
		return nil, fmt.Errorf("not a valid file segment, type:%s", fileobj.Type)
	}
	if fileobj.File.MediaType != "application/octet-stream" {
		return nil, fmt.Errorf("segment not include a valid file Type, type:%s", fileobj.File.MediaType)
	}

	if fileobj.File.Compression == quorumpb.File_none {

	} else {
		return nil, fmt.Errorf("unsupported file compression type:%s", fileobj.File.Compression)
	}

	return fileobj.File.Content, nil
}
