package httpc

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	DefaultDialTimeout    = 5 * time.Second
	DefaultRequestTimeout = 10 * time.Second
)

var (
	transportOnce   sync.Once
	sharedTransport *http.Transport
)

// defaultTransport reuses connections across requests. DefaultDialTimeout is
// read when the transport is first used; DefaultRequestTimeout is read per request.
func defaultTransport() *http.Transport {
	transportOnce.Do(func() {
		sharedTransport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   DefaultDialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
	})
	return sharedTransport
}

func request(req *http.Request) (*http.Response, error) {
	cli := http.Client{
		Transport: defaultTransport(),
		Timeout:   DefaultRequestTimeout,
	}
	return cli.Do(req)
}

func requestBody(req *http.Request) (int, []byte, error) {
	resp, err := request(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var body []byte
	code := resp.StatusCode
	body, err = io.ReadAll(resp.Body)
	return code, body, err
}
