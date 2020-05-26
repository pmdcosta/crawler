# Crawler

A slim and concurrent web crawler written in Go. It fetches all the links in a single webpage and recursively follows
those links until either all recursive links have been visited or the max depth has been reached.

It is written in idiomatic Go and is quite flexible, most components can be easily swapped. It supports an arbitrary number
of parallel requests as well as custom filtering.

It was mainly designed to be used as a cli, but it can easily be imported and used as a lib.

The main components of the program are the workers and the orchestrator. The workers are responsible for making the http
calls using an injected http backend and parsing the html to find the references. The orchestrator manages the workers, 
it filters and requests tasks(links) to be fetched and parsed by the workers and then handles the responses, either errors
that can be retried; or a list of all the links and the number of times that link exists on the page. Data is shared through
channels between the workers and the orchestrator.

## Usage
```
Usage of ./crawler:
  -debug=false: increase verbosity
  -depth=1: set max depth
  -filter-host="": only crawl host
  -filter-subdomain="": only crawl subdomain
  -host="https://google.com": host to crawl
  -output="json": output format (raw, json)
  -parallelism=10: number of concurrent requests
  -retries=3: set retry attempts
  -same-host=true: only crawl the same host
```

## Improvments
- The current implementation can lead to a deadlock when the task channels are full. Need to take a look at this.
