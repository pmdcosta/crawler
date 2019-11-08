package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/pmdcosta/crawler/internal/backend"
	"github.com/pmdcosta/crawler/internal/scraper"
	"github.com/rs/zerolog"
)

func main() {
	l := zerolog.New(os.Stdout).With().Logger()
	b := backend.New(&l)

	host, _ := url.Parse("http://google.com")
	body, _ := b.Do(host)
	urls := scraper.ScrapePage(host, body)
	for u, n := range urls {
		fmt.Println(u, n)
	}
}
