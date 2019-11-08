package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	httpClient "github.com/pmdcosta/crawler/internal/http"
	"github.com/pmdcosta/crawler/internal/scraper"
	"github.com/rs/zerolog"
)

func main() {
	l := zerolog.New(os.Stdout).With().Logger()
	backend := httpClient.New(&l, 10*time.Second)

	host, _ := url.Parse("http://google.com")
	req, _ := http.NewRequest(http.MethodGet, host.String(), nil)
	b, _ := backend.Do(req, 10*1024*1024)

	urls := scraper.ScrapePage(host, b)
	for u, n := range urls {
		fmt.Println(u, n)
	}
}
