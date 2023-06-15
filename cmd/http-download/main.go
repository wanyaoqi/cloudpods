package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/http/httpproxy"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/streamutils"
)

func HttpTransportProxyFunc() httputils.TransportProxyFunc {
	cfg := &httpproxy.Config{
		HTTPProxy:  os.Getenv("http_proxy"),
		HTTPSProxy: os.Getenv("https_proxy"),
	}
	proxyFunc := cfg.ProxyFunc()
	return func(req *http.Request) (*url.URL, error) {
		return proxyFunc(req.URL)
	}
}

func main() {
	header := http.Header{}
	client := httputils.GetTimeoutClient(0)
	transport := httputils.GetTransport(true)
	transport.Proxy = HttpTransportProxyFunc()
	client.Transport = transport
	resp, err := httputils.Request(client, context.Background(), httputils.GET, os.Args[1], header, nil, false)
	if err != nil {
		panic(err)
		return
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body

	fp, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
		return
	}
	var curSaved int64 = 0
	_, err = streamutils.StreamPipe(reader, fp, false, func(saved int64) {
		if saved-curSaved > 100*1024*1024 {
			log.Infof("saved %d", saved)
			curSaved = saved
		}

	})
	if err != nil {
		panic(err)
	}
}
