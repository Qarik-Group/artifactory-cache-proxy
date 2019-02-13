package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/elazarl/goproxy"
	"github.com/starkandwayne/artifactory-cache-proxy/cache"
)

var (
	proxy  *goproxy.ProxyHttpServer
	logger *log.Logger
)

func main() {
	logger = log.New(os.Stdout, "", 0)
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")

	cache, err := cache.NewCache(cache.ArtifactoryConfig{
		URL:        os.Getenv("ARTIFACTORY_URL"),
		Token:      os.Getenv("ARTIFACTORY_TOKEN"),
		Repository: os.Getenv("ARTIFACTORY_REPO"),
	}, logger)
	if err != nil {
		logger.Println(err)
	}

	flag.Parse()
	proxy = goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	proxy.Logger = logger
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(cache).DoFunc(cache.ReqHandle)
	proxy.OnResponse(cache).DoFunc(cache.RespHandle)

	log.Fatal(http.ListenAndServe(*addr, proxy))
}
