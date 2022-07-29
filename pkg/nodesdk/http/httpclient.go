package nodesdkhttpclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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

func (hc *HttpClient) RequestChainAPI(path string, method string, payload interface{}, headers http.Header, result interface{}) error {
	fullUrl, jwt, err := hc.getFullUrl(path)
	if err != nil {
		return err
	}

	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Content-Type", "application/json; charset=utf-8")
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))

	statusCode, content, err := utils.RequestAPI(fullUrl, method, payload, headers, result)
	if err != nil {
		return err
	}

	if statusCode >= 400 {
		errResult := utils.ErrorResponse{}
		if err := json.Unmarshal(content, &errResult); err != nil {
			return fmt.Errorf("request chain api failed: %s", errResult.Message)
		}
		return fmt.Errorf("request chain api failed: %s", content)
	}

	return nil
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
