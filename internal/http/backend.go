package http

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Backend struct {
	logger *zerolog.Logger
	client *http.Client
}

// New instantiates a new Backend
func New(logger *zerolog.Logger, timeout time.Duration) *Backend {
	l := logger.With().Str("pkg", "http").Logger()
	return &Backend{
		logger: &l,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Do executes the http request
func (b *Backend) Do(request *http.Request, bodySize int) ([]byte, error) {
	b.logger.Debug().Str("url", request.URL.String()).Msg("executing http request")

	// execute the http request
	res, err := b.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.Request != nil {
		*request = *res.Request
	}
	b.logger.Debug().Str("url", request.URL.String()).Str("status", res.Status).Int("code", res.StatusCode).Msg("completed http request")

	// limit body size
	var bodyReader io.Reader = res.Body
	if bodySize > 0 {
		bodyReader = io.LimitReader(bodyReader, int64(bodySize))
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
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	return body, nil
}
