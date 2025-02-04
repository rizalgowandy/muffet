package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type fasthttpHttpClient struct {
	client          *fasthttp.Client
	maxRedirections int
	timeout         time.Duration
	headers         map[string]string
}

func newFasthttpHttpClient(c *fasthttp.Client, maxRedirections int, timeout time.Duration, headers map[string]string) httpClient {
	return &fasthttpHttpClient{c, maxRedirections, timeout, headers}
}

func (c *fasthttpHttpClient) Get(u *url.URL) (httpResponse, error) {
	req, res := fasthttp.Request{}, fasthttp.Response{}
	req.SetRequestURI(u.String())
	req.SetConnectionClose()

	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	// Some HTTP servers require "Accept" headers to be set explicitly.
	if !includeHeader(c.headers, "Accept") {
		req.Header.Add("Accept", "*/*")
	}

	i := 0

	for {
		err := c.client.DoTimeout(&req, &res, c.timeout)
		if err != nil && i > 0 {
			return nil, fmt.Errorf("%w (following redirect %v)", err, req.URI())
		} else if err != nil {
			return nil, err
		}

		switch res.StatusCode() / 100 {
		case 2:
			return newFasthttpHttpResponse(req.URI(), &res), nil
		case 3:
			i++

			if i > c.maxRedirections {
				return nil, errors.New("too many redirections")
			}

			u := res.Header.Peek("Location")

			if len(u) == 0 {
				return nil, errors.New("location header not found")
			}

			req.URI().UpdateBytes(u)
		default:
			return nil, fmt.Errorf("%v", res.StatusCode())
		}
	}
}

func includeHeader(hs map[string]string, h string) bool {
	for k := range hs {
		if strings.EqualFold(k, h) {
			return true
		}
	}

	return false
}
