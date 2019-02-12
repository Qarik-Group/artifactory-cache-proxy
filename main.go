package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/atlassian/go-artifactory/pkg/artifactory"
	"github.com/elazarl/goproxy"
	"github.com/google/uuid"
)

var (
	proxy  *goproxy.ProxyHttpServer
	logger *log.Logger
)

// copied/converted from https.go
func dial(proxy *goproxy.ProxyHttpServer, network, addr string) (c net.Conn, err error) {
	if proxy.Tr.Dial != nil {
		return proxy.Tr.Dial(network, addr)
	}
	return net.Dial(network, addr)
}

// copied/converted from https.go
func connectDial(proxy *goproxy.ProxyHttpServer, network, addr string) (c net.Conn, err error) {
	if proxy.ConnectDial == nil {
		return dial(proxy, network, addr)
	}
	return proxy.ConnectDial(network, addr)
}

func handleRequest(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
	logger.Println("making request")
	tp := artifactory.TokenAuthTransport{
		Token: "AKCp5budTFpbypBqQbGJPz3pGCi28pPivfWczqjfYb9drAmd9LbRZbj6UpKFxJXA8ksWGc9fM",
	}

	aclient, err := artifactory.NewClient("http://localhost:8081/artifactory", tp.Client())
	if err != nil {
		fmt.Println(err)
	}

	_, _, err = aclient.System.Ping(context.Background())
	if err != nil {
		fmt.Printf("\nerror: %v\n", err)
	} else {
		fmt.Println("OK")
	}

	clientBuf := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))

	remote, err := connectDial(proxy, "tcp", req.URL.Host)
	if err != nil {
		fmt.Println(err)
	}

	remoteBuf := bufio.NewReadWriter(bufio.NewReader(remote), bufio.NewWriter(remote))

	req, err = http.ReadRequest(clientBuf.Reader)
	if err != nil {
		fmt.Println(err)
	}

	err = req.Write(remoteBuf)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	err = remoteBuf.Flush()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	resp, err := http.ReadResponse(remoteBuf.Reader, req)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	// create the pipe and tee reader
	pr, pw := io.Pipe()
	tr := io.TeeReader(pr, clientBuf.Writer)

	// create channel to synchronize
	done := make(chan bool)
	defer close(done)

	go func() {
		defer pr.Close()
		err = resp.Write(pw)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		err = clientBuf.Flush()
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		done <- true
	}()

	go func() {
		id := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("%s %s", req.Method, req.URL)))

		buffreader := bufio.NewReader(tr)
		foo, err := http.ReadResponse(buffreader, nil)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		b, err := ioutil.ReadAll(foo.Body)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		areq, err := aclient.NewRequest("PUT", filepath.Join("/proxy", id.String()), bytes.NewReader(b))
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		_, err = aclient.Do(context.Background(), areq, nil)
		if err != nil {
			fmt.Println("Foo", err)
			panic(err)
		}
		done <- true
	}()

	for c := 0; c < 2; c++ {
		<-done
	}
	return
}

func main() {
	logger = log.New(os.Stdout, "", 0)
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", ":8080", "proxy listen address")

	flag.Parse()
	proxy = goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	proxy.Logger = logger
	proxy.OnRequest().HijackConnect(handleRequest)

	log.Fatal(http.ListenAndServe(*addr, proxy))
}
