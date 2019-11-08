package main

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/namsral/flag"
	"github.com/pmdcosta/crawler/internal/backend"
	"github.com/pmdcosta/crawler/internal/orchestrator"
	"github.com/pmdcosta/crawler/internal/scraper"
	"github.com/pmdcosta/crawler/internal/worker"
	"github.com/rs/zerolog"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	l := zerolog.New(os.Stdout).With().Logger()

	// handle flags
	var (
		debug           = flag.Bool("debug", false, "increase verbosity")
		host            = flag.String("host", "https://google.com", "host to crawl")
		retries         = flag.Int("retries", 3, "set retry attempts")
		depth           = flag.Int("depth", 1, "set max depth")
		sameHost        = flag.Bool("same-host", true, "only crawl the same host")
		filterSubDomain = flag.String("filter-subdomain", "", "only crawl subdomain")
		filterHost      = flag.String("filter-host", "", "only crawl host")
		parallel        = flag.Int("parallelism", 10, "number of concurrent requests")
		output          = flag.String("output", "json", "output format (raw, json)")
	)
	flag.Parse()

	// handle flags
	if !*debug {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	if *host == "" {
		l.Fatal().Msg("host to crawl is required")
	}
	var options = []orchestrator.Option{
		orchestrator.SetMaxRetries(*retries),
	}
	if *depth != 0 {
		options = append(options, orchestrator.SetMaxDepth(*depth))
	}
	if *sameHost {
		u, _ := url.Parse(*host)
		options = append(options, orchestrator.AddSudDomainFilters(u.Host))
	}
	if *filterSubDomain != "" {
		options = append(options, orchestrator.AddExactHostFilter(*filterSubDomain))
	}
	if *filterHost != "" {
		options = append(options, orchestrator.AddSudDomainFilters(*filterHost))
	}

	// initiate the crawler
	o := orchestrator.New(&l, 10000000, options...)
	var workers []*worker.Worker
	for i := 0; i < *parallel; i++ {
		w := worker.New(&l, o.TaskQueue, o.DoneQueue, o.ErrorQueue, backend.New(&l), scraper.ScrapePage)
		_ = w.Start()
		workers = append(workers, w)
	}

	// start crawling
	_ = o.Start(*host)

	// wait for signal or end
	select {
	case <-o.Done():
	case <-sigs:
		l.Info().Msg("Interrupted crawling")
	}

	// stop workers
	for _, w := range workers {
		w.Stop()
	}
	o.Stop()

	// output
	if *output == "json" {
		fmt.Println(o.GetJson())
	} else if *output == "raw" {
		fmt.Println(o.GetHits())
	}
	l.Info().Int("hits", len(o.Processed)).Msg("Finished crawling")
}
