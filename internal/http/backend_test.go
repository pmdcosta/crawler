package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpClient "github.com/pmdcosta/crawler/internal/http"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestBackend(t *testing.T) {
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte("body"))
	}))
	defer testServer.Close()

	// build backend
	logger := zerolog.Nop()
	backend := httpClient.New(&logger, 10*time.Second)

	// build http request
	req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
	require.Nil(t, err)

	// execute http request
	reader, err := backend.Do(req, 10*1024*1024)
	require.Nil(t, err)

	// read response
	require.Nil(t, err)
	require.Equal(t, "body", string(reader))
}
