package main

import (
	"flag"
	"fmt"
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
		URL:        "http://localhost:8081/artifactory",
		Token:      "AKCp5budTFpbypBqQbGJPz3pGCi28pPivfWczqjfYb9drAmd9LbRZbj6UpKFxJXA8ksWGc9fM",
		Repository: "proxy",
	}, logger)
	if err != nil {
		fmt.Println(err)
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
