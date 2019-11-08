package backend

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Http is the default http backend for the crawler
type Http struct {
	logger *zerolog.Logger
	client *http.Client

	// maximum body size per request
	maxBodySize int

	// custom http request
	request *http.Request
}

// Option is an optimal configuration option that can be applied to a worker
type Option func(b *Http)

// New instantiates a new http client
func New(logger *zerolog.Logger, opts ...Option) *Http {
	l := logger.With().Str("pkg", "http").Logger()
	b := Http{
		logger: &l,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(&b)
	}
	return &b
}

// SetTimeout changes the client timeout
func SetTimeout(t time.Duration) Option {
	return func(b *Http) {
		b.client.Timeout = t
	}
}

// SetMaxBodySize changes the maximum body size per request
func SetMaxBodySize(s int) Option {
	return func(b *Http) {
		b.maxBodySize = s
	}
}

// SetHTTPRequest changes the default http request
func SetHTTPRequest(request *http.Request) Option {
	return func(b *Http) {
		b.request = request
	}
}

// Do executes the http request
func (b *Http) Do(u *url.URL) ([]byte, error) {
	start := time.Now()
	b.logger.Debug().Str("url", u.String()).Msg("executing http request")

	// whether to use a custom request
	var request *http.Request
	if b.request != nil {
		request = b.request
		request.URL = u
	} else {
		request, _ = http.NewRequest(http.MethodGet, u.String(), nil)
	}

	// execute the http request
	res, err := b.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.Request != nil {
		*request = *res.Request
	}
	b.logger.Debug().Str("url", request.URL.String()).Str("status", res.Status).Int("code", res.StatusCode).Dur("elapsed", time.Since(start)).Msg("completed http request")

	// limit body size
	var bodyReader io.Reader = res.Body
	if b.maxBodySize > 0 {
		bodyReader = io.LimitReader(bodyReader, int64(b.maxBodySize))
	}

	// uncompress if gzip body
	if strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "gzip") || strings.Contains(strings.ToLower(res.Header.Get("Content-Encoding")), "gzip") {
		bodyReader, err = gzip.NewReader(bodyReader)
		if err != nil {
			return nil, err
		}
		defer bodyReader.(*gzip.Reader).Close()
	}

	// check if the request was successful
	if res.StatusCode >= http.StatusInternalServerError {
		return nil, nil
	}

	// read response body
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	return body, nil
}
