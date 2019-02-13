package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/atlassian/go-artifactory/pkg/artifactory"
	"github.com/elazarl/goproxy"
	"github.com/google/uuid"
)

type Cache struct {
	client *artifactory.Client
	log    *log.Logger
	config ArtifactoryConfig
}

type ArtifactoryConfig struct {
	URL        string
	Token      string
	Repository string
}

func NewCache(config ArtifactoryConfig, logger *log.Logger) (*Cache, error) {
	tp := artifactory.TokenAuthTransport{Token: config.Token}

	client, err := artifactory.NewClient(config.URL, tp.Client())
	if err != nil {
		return nil, err
	}

	_, _, err = client.System.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	cache := Cache{client: client, log: logger, config: config}

	return &cache, nil
}

func (c *Cache) ReqHandle(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	url, ok := c.isCached(req)
	if ok {
		c.log.Printf("Hitting cache: %s for: %s", url, req.URL.String())
		req.URL = url
	} else {
		c.log.Printf("Caching: %s here: %s", req.URL.String(), url)
		go c.cache(req, ctx)
	}
	return req, nil
}

func (c *Cache) RespHandle(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp.StatusCode == http.StatusFound {
		location, _ := resp.Location()
		req, err := http.NewRequest(resp.Request.Method, location.String(), nil)
		if err != nil {
			c.log.Printf("Error while creating redirect request for %s: %s",
				req.URL.String(), err)
			return nil
		}
		c.log.Printf("resp handler: %s", location)

		resp, err = ctx.RoundTrip(req)
		if err != nil {
			c.log.Printf("Error while resolving redirect request for %s: %s",
				req.URL.String(), err)
			return nil
		}
	}
	return resp
}

func (c *Cache) HandleReq(req *http.Request, _ *goproxy.ProxyCtx) bool {
	if req.Method == http.MethodGet {
		return true
	}
	return false
}

func (c *Cache) HandleResp(_ *http.Response, _ *goproxy.ProxyCtx) bool {
	return true
}

func (c *Cache) cache(req *http.Request, ctx *goproxy.ProxyCtx) {
	creq, err := http.NewRequest(req.Method, req.URL.String(), nil)
	if err != nil {
		c.log.Printf("Error while creating cache request for %s: %s",
			req.URL.String(), err)
		return
	}

	client := &http.Client{}
	client.Transport = ctx
	resp, err := client.Do(creq)
	if err != nil {
		c.log.Printf("Error while performing cache request for %s: %s",
			req.URL.String(), err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.log.Printf("Error while reading cache response body for %s: %s",
			req.URL.String(), err)
		return
	}

	areq, err := c.client.NewRequest(
		http.MethodPut,
		c.getCachePath(req),
		bytes.NewReader(body),
	)

	sha := sha256.Sum256(body)
	areq.Header.Add("X-Checksum-Deploy", `true`)
	areq.Header.Add("X-Checksum-Sha256", fmt.Sprintf("%x", sha))

	if err != nil {
		c.log.Printf("Error while building artifactory cache request for %s: %s",
			req.URL.String(), err)
		return
	}

	_, err = c.client.Do(context.Background(), areq, nil)
	if err != nil {
		c.log.Printf("Error while performing artifactory cache request for %s: %s",
			req.URL.String(), err)
		return
	}
}

func (c *Cache) isCached(req *http.Request) (*url.URL, bool) {
	url := c.getCacheURL(req)
	hreq, _ := http.NewRequest(http.MethodHead, url.String(), nil)
	resp, err := c.client.Do(context.Background(), hreq, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		return url, true
	}
	return url, false
}

func (c *Cache) getCacheURL(req *http.Request) *url.URL {
	url, _ := url.Parse(c.config.URL)
	url.Path = filepath.Join(url.Path, c.getCachePath(req))
	return url
}

func (c *Cache) getCachePath(req *http.Request) string {
	return filepath.Join(c.config.Repository,
		req.URL.Hostname(),
		req.URL.Path,
		req.URL.RawQuery,
		getId(req).String(),
	)
}

func getId(req *http.Request) uuid.UUID {
	return uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("%s %s", req.Method, req.URL)))
}
