// +build !js

package utils

import (
	"bytes"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Press-One/go-update"
)

// releases pub key
const ED25519PublicKey = `untrusted comment: signify public key
RWStFU9JBrtWhvm1VVzbH63KKj/2CdSqM82HldQmDzS8kLq2rQPLeQJG
`

// export GITHUB_TOKEN=xxxxx before this project is opensourced
const LatestReleaseUrl = "https://api.github.com/repos/rumsystem/quorum/releases/latest"
const LatestReleaseUrlQingCloud = "https://static-assets.pek3b.qingstor.com"

func getGithub(url string, isRaw bool) ([]byte, error) {
	logger.Infof("Get: %s\n", url)

	ghToken := os.Getenv("GITHUB_TOKEN")

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer([]byte("")))
	req.Header.Set("Authorization", fmt.Sprintf("token %s", ghToken))
	if isRaw {
		// this will get the assets download url, see: https://stackoverflow.com/questions/25923939/how-do-i-download-binary-files-of-a-github-release
		req.Header.Set("Accept", "application/octet-stream")
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	content, _ := ioutil.ReadAll(resp.Body)

	return content, nil
}

func getQingCloud(url string, isRaw bool) ([]byte, error) {
	logger.Infof("Get: %s\n", url)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer([]byte("")))
	if isRaw {
		// this will get the assets download url, see: https://stackoverflow.com/questions/25923939/how-do-i-download-binary-files-of-a-github-release
		req.Header.Set("Accept", "application/octet-stream")
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	content, _ := ioutil.ReadAll(resp.Body)

	return content, nil
}

func CheckUpdate(curVersion string, binName string) error {
	content, err := getGithub(LatestReleaseUrl, false)
	if err != nil {
		return err
	}
	releaseInfo := GithubReleaseStruct{}
	err = json.Unmarshal(content, &releaseInfo)
	if err != nil {
		return err
	}
	tagName := releaseInfo.TagName
	if tagName == "" {
		return errors.New("Failed to fetch latest version number")
	}
	logger.Infof("Found new version: %s, current version: %s\n", tagName, curVersion)
	if tagName > curVersion {
		baseName := fmt.Sprintf("%s-%s-%s-%s", binName, tagName, runtime.GOOS, runtime.GOARCH)
		tarName := baseName + ".tar.gz"
		if runtime.GOOS == "windows" {
			tarName = baseName + ".zip"
		}
		sigName := tarName + ".sig"
		tarUrl := ""
		sigUrl := ""

		for _, asset := range releaseInfo.Assets {
			if asset.Name == tarName {
				tarUrl = asset.Url
			}
			if asset.Name == sigName {
				sigUrl = asset.Url
			}
		}

		signature, err := getGithub(sigUrl, true)
		if err != nil {
			return err
		}
		tarContent, err := getGithub(tarUrl, true)
		if err != nil {
			return err
		}
		opts := update.Options{
			Verifier:         update.NewED25519Verifier(),
			VerifyUseContent: false,
			PublicKey:        []byte(ED25519PublicKey),
			Signature:        signature,
			Hash:             crypto.SHA256,
		}
		if runtime.GOOS == "windows" {
			opts.IsZip = true
		} else {
			opts.IsTarGz = true
		}
		logger.Infof("Verifying..\n")
		err = update.Apply(bytes.NewReader(tarContent), opts)
		if err != nil {
			return err
		}
		logger.Infof("Update sucess!\n")
	}

	return nil
}

func CheckUpdateQingCloud(curVersion string, binName string) error {
	content, err := getQingCloud(fmt.Sprintf("%s/%s/VERSION.txt", LatestReleaseUrlQingCloud, binName), false)
	if err != nil {
		return err
	}
	version := string(content)
	tagName := strings.TrimSpace(strings.Split(version, "-")[0])
	logger.Infof("Found new version: %s, current version: %s\n", tagName, curVersion)
	if tagName > curVersion {
		baseName := fmt.Sprintf("%s-%s-%s-%s", binName, tagName, runtime.GOOS, runtime.GOARCH)
		tarName := baseName + ".tar.gz"
		if runtime.GOOS == "windows" {
			tarName = baseName + ".zip"
		}
		sigName := tarName + ".sig"
		tarUrl := fmt.Sprintf("%s/%s/%s", LatestReleaseUrlQingCloud, binName, tarName)
		sigUrl := fmt.Sprintf("%s/%s/%s", LatestReleaseUrlQingCloud, binName, sigName)

		signature, err := getQingCloud(sigUrl, true)
		if err != nil {
			return err
		}
		tarContent, err := getQingCloud(tarUrl, true)
		if err != nil {
			return err
		}
		opts := update.Options{
			Verifier:         update.NewED25519Verifier(),
			VerifyUseContent: false,
			PublicKey:        []byte(ED25519PublicKey),
			Signature:        signature,
			Hash:             crypto.SHA256,
		}
		if runtime.GOOS == "windows" {
			opts.IsZip = true
		} else {
			opts.IsTarGz = true
		}
		logger.Infof("Verifying..\n")
		err = update.Apply(bytes.NewReader(tarContent), opts)
		if err != nil {
			return err
		}
		logger.Infof("Update sucess!\n")
	}

	return nil
}

type GithubReleaseStruct struct {
	Assets  []GithubAssetStruct `json:"assets"`
	TagName string              `json:"tag_name"`
}

type GithubAssetStruct struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}
