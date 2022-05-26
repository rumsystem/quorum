package nodesdkhttpclient

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/rumsystem/quorum/cmd/cli/config"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	u2 "github.com/rumsystem/quorum/internal/pkg/utils"
)

var http_log = logging.Logger("http")

type APIServerItem struct {
	url      string
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

	/*
		if len(urls) == 0 {
			return errors.New("At least 1 url should be provided")
		}
	*/

	var apis []*APIServerItem

	for _, u := range urls {
		_, err := url.Parse(u)
		if err != nil {
			return errors.New("Invalid url")
		}

		var urlItem *APIServerItem
		urlItem = &APIServerItem{}

		urlItem.url = u
		urlItem.pinginms = 0
		urlItem.memo = ""
		apis = append(apis, urlItem)
	}
	//set group API
	hc.APIs = apis

	return nil
}

func (hc *HttpClient) Get(url string) ([]byte, error) {
	http_log.Infof("Get called, groupId <%s>", url)

	fullUrl, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	jwt := config.RumConfig.Quorum.JWT
	client, err := u2.NewHTTPClient() // hc.newHTTPClient()
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

func (hc *HttpClient) Post(url string, data []byte) ([]byte, error) {
	http_log.Infof("Post called, <%s>", url)

	fullUrl, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	jwt := config.RumConfig.Quorum.JWT
	client, err := u2.NewHTTPClient() // hc.newHTTPClient()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(data))
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

	if strings.Contains(string(body), "error") {
		return nil, errors.New(string(body))
	}

	return body, err
}

func (hc *HttpClient) Delete(url string, data []byte) ([]byte, error) {
	http_log.Infof("Delete called %s", url)

	fullUrl, err := hc.getFullUrl(url)
	if err != nil {
		return nil, err
	}

	jwt := config.RumConfig.Quorum.JWT
	client, err := u2.NewHTTPClient() //hc.newHTTPClient()
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

func (hc *HttpClient) getFullUrl(u2 string) (fullurl string, err error) {
	//TBD: find the fastest api
	//TBD: skip unavailable api
	//now just return the first URL of remote api in the list
	result := hc.APIs[0].url + u2

	http_log.Infof("url %s", result)

	_, err = url.Parse(result)
	if err != nil {
		return "", errors.New("Can not get Full Url, url invalid")
	}

	return result, nil
}

/*
func (hc *HttpClient) newHTTPClient() (*http.Client, error) {
	certPath, err := filepath.Abs(config.RumConfig.Quorum.ServerSSLCertificate)
	if err != nil {
		return nil, err
	}

	if certPath != "" {
		caCert, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		if err != nil {
			return nil, err
		}
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		if config.RumConfig.Quorum.ServerSSLInsecure {
			tlsConfig.InsecureSkipVerify = true
		}

		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true}
		// 5 seconds timeout, all timeout will be ignored, since we refresh all data every half second
		return &http.Client{Transport: transport, Timeout: 5 * time.Second}, nil
	}

	return &http.Client{}, nil
}
*/

func (hc *HttpClient) checkJWTError(body string) error {
	if strings.Contains(body, "missing or malformed jwt") {
		return errors.New("missing or malformed jwt")
	}

	if strings.Contains(body, "please find jwt token in peer options") {
		return errors.New("Someone applied before, please find jwt token in peer options, then update your config and reload.")
	}
	return nil
}
