package backend_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pmdcosta/crawler/internal/backend"
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
	client := backend.New(&logger)

	// execute http request
	u, _ := url.Parse(testServer.URL)
	reader, err := client.Do(u)
	require.Nil(t, err)

	// read response
	require.Nil(t, err)
	require.Equal(t, "body", string(reader))
}
