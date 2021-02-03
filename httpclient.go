package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/buger/jsonparser"
	logger "github.com/sirupsen/logrus"
)

type httpClient struct {
	ak, sk   string
	endpoint *url.URL
	client   *http.Client
}

func initHttpClient() {
	endpoint, err := url.Parse(cattleURL)
	if err != nil {
		panic(err)
	}
	hc = &httpClient{
		ak:       cattleAccessKey,
		sk:       cattleSecretKey,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: timeout,
		},
	}
	hc.client.Transport = hc
}

func (r *httpClient) get(uri string, queries url.Values) ([]byte, error) {
	after := getSubAddress(r.endpoint, uri)
	after.RawQuery = queries.Encode()
	resp, err := r.client.Get(after.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (r *httpClient) getByProject(uri string, queries url.Values) ([]byte, error) {
	return r.get(path.Join("projects", projectID, uri), queries)
}

func (r *httpClient) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(r.ak, r.sk)
	return http.DefaultTransport.RoundTrip(req)
}

func (r *httpClient) foreachCollection(uri string, queries url.Values, wg *sync.WaitGroup, contentHandler func(data []byte)) error {
	defaultQuery := url.Values{
		"limit": []string{"100"},
		"sort":  []string{"id"},
	}
	if hideSys {
		defaultQuery.Set("system", "false")
	}

	for k, v := range queries {
		defaultQuery[k] = v
	}

	hasNext := true

	for hasNext {
		collectionData, err := r.getByProject(uri, defaultQuery)
		if err != nil {
			return err
		}

		rtnType, err := jsonparser.GetString(collectionData, "type")
		if err != nil {
			return err
		}

		if rtnType != "collection" {
			return fmt.Errorf("uri %s is not a collection uri", uri)
		}

		if next, _ := jsonparser.GetString(collectionData, "pagination", "next"); len(next) == 0 {
			hasNext = false
		} else {
			u, err := url.Parse(next)
			if err != nil {
				logger.Warnf("failed to parse next url, %v", err)
				hasNext = false
			}
			defaultQuery = u.Query()
		}

		syncFunc := func(wg *sync.WaitGroup) error {
			if wg != nil {
				defer wg.Done()
			}
			if contentHandler == nil {
				return nil
			}
			if _, err := jsonparser.ArrayEach(collectionData, func(content []byte, dataType jsonparser.ValueType, offset int, err error) {
				contentHandler(content)
			}, "data"); err != nil {
				return err
			}
			return nil
		}

		if wg == nil {
			return syncFunc(wg)
		} else {
			wg.Add(1)
			go func() {
				_ = syncFunc(wg)
			}()
		}
	}

	return nil
}
