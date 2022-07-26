package nodesdkhttpclient

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	u2 "github.com/rumsystem/quorum/internal/pkg/utils"
)

var http_log = logging.Logger("http")

type APIServerItem struct {
	url      string
	jwt      string
	pinginms int
	memo     string
}

type HttpClient struct {
	APIs []*APIServerItem
}

func (hc *HttpClient) Init() error {
	http_log.Infof("Init called")
	return nil
}

func (hc *HttpClient) UpdApiServer(urls []string) error {
	http_log.Infof("UpdApiServer called")

	var apis []*APIServerItem

	for _, u := range urls {
		_url, jwt, err := utils.ParseChainapiURL(u)
		if err != nil {
			return errors.New("Invalid url")
		}
		if jwt == "" {
			return errors.New("Invalid jwt")
		}

		var urlItem *APIServerItem
		urlItem = &APIServerItem{
			url:      _url,
			jwt:      jwt,
			pinginms: 0,
			memo:     "",
		}
		apis = append(apis, urlItem)
	}

	//set group API
	hc.APIs = apis

	return nil
}

func (hc *HttpClient) Get(url string) ([]byte, error) {
	http_log.Infof("Get called, groupId <%s>", url)

	fullUrl, jwt, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	client, err := u2.NewHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, fullUrl, bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := hc.checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}
	return body, err
}

func (hc *HttpClient) GetWithBody(url string, reqData []byte) ([]byte, error) {
	http_log.Infof("Get called, groupId <%s>", url)

	fullUrl, jwt, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	client, err := u2.NewHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, fullUrl, bytes.NewBuffer(reqData))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := hc.checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}
	return body, err
}

func (hc *HttpClient) Post(url string, data []byte) ([]byte, error) {
	http_log.Infof("Post called, <%s>", url)

	fullUrl, jwt, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	client, err := u2.NewHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := hc.checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}

	return body, err
}

func (hc *HttpClient) Delete(url string, data []byte) ([]byte, error) {
	http_log.Infof("Delete called %s", url)

	fullUrl, jwt, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	client, err := u2.NewHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodDelete, fullUrl, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if jwt != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	jwtErr := hc.checkJWTError(string(body))
	if jwtErr != nil {
		return nil, jwtErr
	}
	return body, err
}

func (hc *HttpClient) getFullUrl(path string) (fullurl string, jwt string, err error) {
	//TBD: find the fastest api
	//TBD: skip unavailable api
	//now just return the first URL of remote api in the list
	jwt = hc.APIs[0].jwt
	baseUrl := hc.APIs[0].url

	u, err := url.Parse(baseUrl)
	if err != nil {
		return "", "", errors.New("Can not get Full Url, url invalid")
	}
	u.Path = path
	fullurl = u.String()

	http_log.Debugf("fullurl: %s", fullurl)

	return fullurl, jwt, nil
}

func (hc *HttpClient) checkJWTError(body string) error {
	if strings.Contains(body, "missing or malformed jwt") {
		return errors.New("missing or malformed jwt")
	}

	if strings.Contains(body, "please find jwt token in peer options") {
		return errors.New("Someone applied before, please find jwt token in peer options, then update your config and reload.")
	}
	return nil
}
