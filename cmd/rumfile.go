package cmd

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"github.com/spf13/cobra"
)

const ChunkSize int = 150 * 1024

var ( // flags
	rumfileApiPrefix string
	rumfilePath      string
	rumfileDir       string
	rumfileGroupId   string
	rumfileTrxId     string
)

var rumfileCmd = &cobra.Command{
	Use:   "rumfile",
	Short: "A tool to upload and download files from rum network",
}

var rumfileSplitCmd = &cobra.Command{
	Use:   "split",
	Short: "Split",
	Run: func(cmd *cobra.Command, args []string) {
		if err := Split(rumfilePath, rumfileDir); err != nil {
			logger.Fatal(err)
		}
	},
}

var rumfileUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload",
	Run: func(cmd *cobra.Command, args []string) {
		if err := Upload(rumfileDir, rumfileGroupId, rumfileApiPrefix); err != nil {
			logger.Fatal(err)
		}
	},
}

var rumfileDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download",
	Run: func(cmd *cobra.Command, args []string) {
		if err := Download(rumfileDir, rumfileApiPrefix, rumfileGroupId, rumfileTrxId); err != nil {
			logger.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(rumfileCmd)
	rumfileCmd.AddCommand(rumfileSplitCmd)
	rumfileCmd.AddCommand(rumfileUploadCmd)
	rumfileCmd.AddCommand(rumfileDownloadCmd)
	rumfileCmd.Flags().SortFlags = false

	// split
	splitFlags := rumfileSplitCmd.Flags()
	splitFlags.SortFlags = false
	splitFlags.StringVar(&rumfilePath, "file", "", "a file name")
	splitFlags.StringVar(&rumfileDir, "tmpdir", "/tmp", "a dir name")
	rumfileSplitCmd.MarkFlagRequired("file")

	// upload
	uploadFlags := rumfileUploadCmd.Flags()
	uploadFlags.SortFlags = false
	uploadFlags.StringVar(&rumfileApiPrefix, "api", "http://localhost:8000", "api prefix of the rumservice")
	uploadFlags.StringVar(&rumfileDir, "dir", "", "the file segments dir.(the result of split cmd)")
	uploadFlags.StringVar(&rumfileGroupId, "groupid", "", "the upload target groupid")
	rumfileUploadCmd.MarkFlagRequired("dir")
	rumfileUploadCmd.MarkFlagRequired("groupid")

	// download
	downloadFlags := rumfileDownloadCmd.Flags()
	downloadFlags.SortFlags = false
	downloadFlags.StringVar(&rumfileApiPrefix, "api", "http://localhost:8000", "api prefix of the rumservice")
	downloadFlags.StringVar(&rumfileDir, "dir", ".", "the file segments dir.(the result of split cmd)")
	downloadFlags.StringVar(&rumfileGroupId, "groupid", "", "group_id of the SeedNetwork")
	downloadFlags.StringVar(&rumfileTrxId, "trxid", "", "trx_id of the fileinfo")
	rumfileDownloadCmd.MarkFlagRequired("groupid")
	rumfileDownloadCmd.MarkFlagRequired("trxid")
}

func Download(destdir, apiPrefix, groupid, trxid string) error {
	logger.Infof("download...%s from group %s with  %s\n", trxid, groupid, apiPrefix)
	//first, get the fileinfo file
	fileobj, _, err := HttpGetFileFromGroup(apiPrefix, groupid, trxid)
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
	logger.Infof("file %s has %d segments, downloading...", fileinfo.Name, len(*fileinfo.Segments))

	f, err := os.Create(filepath.Join(destdir, fileinfo.Name))
	if err != nil {
		return err
	}
	defer f.Close()

	readnexttrxid := trxid
	for _, seg := range *fileinfo.Segments {
		fileobj, trxid, err := HttpGetNextFileFromGroupByTrx(apiPrefix, groupid, readnexttrxid)
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

func Upload(segmentsdir, groupid, apiPrefix string) error {
	fileinfo, err := VerifySegments(segmentsdir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uploading to ..", apiPrefix)
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
	trxid, r := PostFileToGroupApi(apiPrefix, groupid, post)
	if r != nil {
		return fmt.Errorf("post %s failed, err %s", obj.Name, r)
	} else {
		logger.Infof("post %s succeed at %s/%s", obj.Name, groupid, trxid)
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
		trxid, r = PostFileToGroupApi(apiPrefix, groupid, post)
		if r != nil {
			return fmt.Errorf("post %s failed, err: %s", obj.Name, r)
		} else {
			logger.Infof("post %s succeed at %s/%s", obj.Name, groupid, trxid)
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
		result, _ := HttpCheckTrxId(apiPrefix, groupid, trxresult.TrxId)
		if result == true {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return trxresult.TrxId, nil
}

func Split(filename, tmpdir string) error {
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

	logger.Infof("Write fileinfo to %s/%s", segmentpath, "fileinfo")
}

func FileToSegments(filename string, fileinfo *Fileinfo, tmpdir string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	segfileinfolist := []Segmentinfo{}

	segmentpath := filepath.Join(tmpdir, fmt.Sprintf("%s-segs", path.Base(filename)))
	logger.Infof("Splitting file: %s ...", filename)
	logger.Infof("Create a temp dir %s ...", segmentpath)
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
		//logger.Infof("save base64 file segment size %d (base64ed size %d) to %s/%s", len(buf), len(data), segmentpath, segname)
		logger.Infof("save file segment size %d  to %s/%s", len(buf), segmentpath, segname)
		err = WriteToFile(segmentpath, segname, buf)
		if err == nil {
			seginfo := &Segmentinfo{Id: segname, Sha256: bufhashhex}
			segfileinfolist = append(segfileinfolist, *seginfo)
		} else {
			log.Fatal(err)
		}
	}
	logger.Infof("File bytes: %d, Write Segments:%d", nBytes, nChunks)

	filesha256 := filehash.Sum(nil)
	fileinfo.Sha256 = hex.EncodeToString(filesha256[:])
	fileinfo.Segments = &segfileinfolist

	WriteFileinfo(segmentpath, fileinfo)
	logger.Infof("file segments done: %s", segmentpath)
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
	logger.Infof("Verify Segments %s ...", segmentpath)
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

func HttpCheckTrxId(apiPrefix string, groupid string, trxid string) (bool, error) {
	Url := fmt.Sprintf("%s/api/v1/trx/%s/%s", apiPrefix, groupid, trxid)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(Url)

	if err != nil {
		logger.Infof("HttpCheckTrxId err: %s", err)
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

func HttpPostToGroup(apiPrefix string, jsondata []byte) (*TrxResult, error) {
	Url := fmt.Sprintf("%s/api/v1/group/content", apiPrefix)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer(jsondata))
	if err != nil {
		logger.Infof("new http request  err: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Infof("post add tasks err: %s", err)
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

func HttpGetFromContentApi(apiPrefix string, groupid string, trxid string, num int, includetrx bool) ([]byte, error) {
	Url := fmt.Sprintf("%s/app/api/v1/group/%s/content?num=1&starttrx=%s&reverse=false&includestarttrx=%t", apiPrefix, groupid, trxid, includetrx)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", Url, bytes.NewBuffer([]byte(`{"senders":[]}`)))
	if err != nil {
		logger.Infof("new http request  err: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Infof("post add tasks err: %s", err)
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

func HttpGetNextFileFromGroupByTrx(apiPrefix string, groupid string, trxid string) (*quorumpb.Object, string, error) {
	body, err := HttpGetFromContentApi(apiPrefix, groupid, trxid, 1, false)
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

func HttpGetFileFromGroup(apiPrefix string, groupid string, trxid string) (*quorumpb.Object, string, error) {

	body, err := HttpGetFromContentApi(apiPrefix, groupid, trxid, 1, true)
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

	//curl -X POST -H 'Content-Type: application/json' -d '{"senders":[]}' "http://localhost:8004/app/api/v1/group/a6ceb724-02c3-4c7c-a50e-846389a6e887/content?num=1&starttrx=cac0fca3-6927-43f5-9e5d-3f8c1906ee2e&reverse=false&includestarttrx=true"

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
