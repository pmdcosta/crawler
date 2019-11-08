package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/pmdcosta/crawler/internal/crawler"

	"github.com/rs/zerolog"
)

// Orchestrator manages the crawler state and workload
type Orchestrator struct {
	logger *zerolog.Logger
	// outbound task channel with tasks to be Processed
	TaskQueue chan crawler.Task
	// inbound task channel with tasks that were Processed
	DoneQueue chan crawler.TaskResult
	// inbound error channel with tasks that Failed to be Processed
	ErrorQueue chan crawler.TaskResult

	// all the tasks Processed
	Processed map[string]crawler.TaskResult
	// all the tasks that Failed to be Processed
	Failed map[string]crawler.TaskResult

	// max number of retries for each Failed task
	maxRetry int
	// max depth of the tree
	maxDepth int

	// host filters
	exactHostFilters map[string]struct{}
	sudDomainFilters []string
	filters          []Filter

	// number of tasks being Processed at this moment
	inProcess int

	// gracefully shutdown orchestrator
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
	doneCh chan struct{}
}

// Option is an optimal configuration option that can be applied to an orchestrator
type Option func(o *Orchestrator)

// Filter is an optimal filtering function to evaluate if the task should be Processed
type Filter func(string) (allow bool)

// New instantiates a new orchestrator
func New(logger *zerolog.Logger, size int, opts ...Option) *Orchestrator {
	l := logger.With().Str("pkg", "orchestrator").Logger()
	w := Orchestrator{
		logger:   &l,
		maxRetry: 3,

		TaskQueue:  make(chan crawler.Task, size),
		DoneQueue:  make(chan crawler.TaskResult, size),
		ErrorQueue: make(chan crawler.TaskResult, size),

		Processed: make(map[string]crawler.TaskResult),
		Failed:    make(map[string]crawler.TaskResult),
	}
	for _, opt := range opts {
		opt(&w)
	}
	return &w
}

// SetMaxRetries sets the max retry count for each Failed task
func SetMaxRetries(n int) Option {
	return func(o *Orchestrator) {
		o.maxRetry = n
	}
}

// SetMaxDepth sets the max depth of the tree
func SetMaxDepth(n int) Option {
	return func(o *Orchestrator) {
		o.maxDepth = n
	}
}

// AddExactHostFilter adds a host name to filter tasks
// hosts added are whitelisted if there's an exact match on the provided host
func AddExactHostFilter(host string) Option {
	return func(o *Orchestrator) {
		if o.exactHostFilters == nil {
			o.exactHostFilters = make(map[string]struct{})
		}
		o.exactHostFilters[host] = struct{}{}
	}
}

// AddSudDomainFilters adds a host name to filter tasks
// tasks are Processed if they are a subdomain of the provided host
func AddSudDomainFilters(host string) Option {
	return func(o *Orchestrator) {
		o.sudDomainFilters = append(o.sudDomainFilters, host)
	}
}

// AddCustomFilter adds a custom filter to the orchestrator
func AddCustomFilter(filter Filter) Option {
	return func(o *Orchestrator) {
		o.filters = append(o.filters, filter)
	}
}

// Start starts processing TaskQueue
func (o *Orchestrator) Start(host string) error {
	if o.ctx != nil {
		return errors.New("orchestrator already started")
	}

	// enqueue the first task
	o.queueHost(host, 0)

	// start orchestrator
	ctx, cancel := context.WithCancel(context.Background())
	o.ctx = ctx
	o.cancel = cancel
	o.stopCh = make(chan struct{})
	o.doneCh = make(chan struct{})
	go o.run()
	return nil
}

// Stop stops the worker
func (o Orchestrator) Stop() {
	if o.ctx == nil {
		return
	}
	// stop the orchestrator
	o.cancel()
	o.logger.Info().Msg("stopping orchestrator")

	// wait for the orchestrator to be gracefully stopped
	select {
	case <-o.stopCh:
		return
	case <-time.After(10 * time.Second):
		return
	}
}

// Done waits until the crawling is finished
func (o Orchestrator) Done() <-chan struct{} {
	return o.doneCh
}

// run is the main execution loop of the worker
func (o *Orchestrator) run() {
	o.logger.Info().Msg("orchestrator started...")
	for {
		o.checkFinished()
		select {
		case <-o.ctx.Done():
			o.logger.Info().Msg("orchestrator stopping...")
			o.stopCh <- struct{}{}
			return
		case task, ok := <-o.DoneQueue:
			if ok {
				o.handleTask(task)
			}
		case task, ok := <-o.ErrorQueue:
			if ok {
				o.handleFailed(task)
			}
		}
	}
}

// handleTask handles successfully  Processed tasks
func (o *Orchestrator) handleTask(result crawler.TaskResult) {
	o.inProcess -= 1
	// add the task to the Processed cache
	o.Processed[result.URL.String()] = result

	// check if the children have been Processed already
	for u, _ := range result.Children {
		if _, found := o.Processed[u]; !found {
			// check if the children should be Processed based on filters
			if o.applyFilters(u) {
				o.queueHost(u, result.Depth+1)
			}
		}
	}
}

// applyFilters checks if the task should be Processed using the filters
func (o *Orchestrator) applyFilters(u string) bool {
	if !o.applyHostFilters(u) {
		return false
	}
	for _, f := range o.filters {
		if !f(u) {
			return false
		}
	}
	return true
}

// applyHostFilters checks if the task should be Processed using the default host filters
func (o *Orchestrator) applyHostFilters(u string) bool {
	host, _ := url.Parse(u)
	u = host.Host
	if o.exactHostFilters != nil {
		if _, found := o.exactHostFilters[u]; !found {
			return false
		}
	}
	if o.sudDomainFilters != nil {
		for _, d := range o.sudDomainFilters {
			if strings.HasSuffix(u, d) {
				return true
			}
		}
		return false
	}
	return true
}

// queueHost creates a new task and schedules it to be Processed
func (o *Orchestrator) queueHost(u string, depth int) {
	// dont queue task if we hit the depth limit
	if o.maxDepth != 0 && depth > o.maxDepth {
		return
	}
	host, err := url.Parse(u)
	if err != nil {
		return
	}
	o.processTask(crawler.Task{URL: host, Depth: depth})
}

// handleFailed handles tasks that Failed to be Processed
func (o *Orchestrator) handleFailed(result crawler.TaskResult) {
	o.inProcess -= 1
	if result.Tries > o.maxRetry {
		// add the task to the Failed cache
		o.Failed[result.URL.String()] = result
		return
	}
	// retry the task
	o.processTask(result.Task)
}

// processTask queues a task to be Processed
func (o *Orchestrator) processTask(task crawler.Task) {
	o.TaskQueue <- task
	o.inProcess += 1
}

// checkFinished checks if all the tasks have been Processed
func (o *Orchestrator) checkFinished() {
	// there are no tasks being Processed
	if o.inProcess == 0 {
		o.doneCh <- struct{}{}
	}
}

// GetHits returns the crawled pages
func (o *Orchestrator) GetHits() map[string]map[string]int {
	var result = make(map[string]map[string]int)
	for _, p := range o.Processed {
		result[p.URL.String()] = p.Children
	}
	return result
}

// GetJson returns a json formatted version of the crawled pages
func (o *Orchestrator) GetJson() string {
	j, _ := json.Marshal(o.GetHits())
	return string(j)
}
