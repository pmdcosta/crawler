package worker

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/pmdcosta/crawler/internal/crawler"
	"github.com/rs/zerolog"
)

// Worker processes crawling tasks
type Worker struct {
	logger *zerolog.Logger
	// inbound task channel with tasks to be processed
	taskQueue chan crawler.Task
	// outbound task channel with tasks that were processed
	doneQueue chan crawler.TaskResult
	// outbound error channel with tasks that failed to be processed
	errorQueue chan crawler.TaskResult

	// list of callbacks that will run while processing a task
	preProcessors  []PreProcessor
	postProcessors []PostProcessor

	// http client for requests
	backend Backend

	// scraper is the function used to scrape a webpage
	scraper Scraper

	// gracefully shutdown worker
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
}

// Backend defines the backend client to make http requests
//go:generate mockgen -destination ../../mocks/backend_mock.go -package mocks -mock_names Backend=MockWorkerBackend github.com/pmdcosta/crawler/internal/worker Backend
type Backend interface {
	Do(u *url.URL) ([]byte, error)
}

// PreProcessor are custom functions that run before processing a task
// if an error is return the task will be failed
// if the ignore bool is true, the task will be ignored
type PreProcessor func(task *crawler.Task) (ignore bool, err error)

// PostProcessor are custom functions that run after processing a task
type PostProcessor func(task *crawler.TaskResult) error

// Option is an optimal configuration option that can be applied to a worker
type Option func(w *Worker)

// Scraper is the definition of the function used to scrape a webpage
type Scraper func(root *url.URL, page []byte) map[string]int

// New instantiates a new worker
func New(logger *zerolog.Logger, tasks chan crawler.Task, done chan crawler.TaskResult, errors chan crawler.TaskResult, http Backend, scraper Scraper, opts ...Option) *Worker {
	l := logger.With().Str("pkg", "worker").Logger()
	w := Worker{
		logger:     &l,
		taskQueue:  tasks,
		doneQueue:  done,
		errorQueue: errors,
		backend:    http,
		scraper:    scraper,
	}
	for _, opt := range opts {
		opt(&w)
	}
	return &w
}

// AddPreProcessor adds a function that will execute before processing a task
func AddPreProcessor(f PreProcessor) Option {
	return func(w *Worker) {
		w.preProcessors = append(w.preProcessors, f)
	}
}

// AddPostProcessor adds a function that will execute after processing a task
func AddPostProcessor(f PostProcessor) Option {
	return func(w *Worker) {
		w.postProcessors = append(w.postProcessors, f)
	}
}

// Start starts processing taskQueue
func (w *Worker) Start() error {
	if w.ctx != nil {
		return errors.New("worker already started")
	}

	// start worker
	ctx, cancel := context.WithCancel(context.Background())
	w.ctx = ctx
	w.cancel = cancel
	w.stopCh = make(chan struct{})
	go w.run()
	return nil
}

// Stop stops the worker
func (w Worker) Stop() {
	if w.ctx == nil {
		return
	}
	// stop the worker
	w.cancel()
	w.logger.Info().Msg("stopping worker")

	// wait for the worker to be gracefully stopped
	select {
	case <-w.stopCh:
		return
	case <-time.After(10 * time.Second):
		return
	}
}

// run is the main execution loop of the worker
func (w *Worker) run() {
	w.logger.Info().Msg("worker starting...")
	for {
		select {
		case <-w.ctx.Done():
			w.logger.Debug().Msg("worker stopping...")
			w.stopCh <- struct{}{}
			return
		case task, ok := <-w.taskQueue:
			if ok {
				result, err := w.processTask(&task)
				if err != nil {
					w.errorQueue <- result
				} else {
					w.doneQueue <- result
				}
			}
		}
	}
}

// run is the main execution loop of the worker
func (w *Worker) processTask(task *crawler.Task) (crawler.TaskResult, error) {
	w.logger.Info().Str("url", task.URL.String()).Int("try", task.Tries).Msg("processing task")

	// increment try counter
	task.Tries += 1

	// executing pre-processors
	for _, f := range w.preProcessors {
		ignore, err := f(task)
		if err != nil {
			return crawler.TaskResult{Task: *task, Children: nil, Error: &err}, err
		}
		if ignore {
			return crawler.TaskResult{Task: *task, Children: nil}, err
		}
	}

	// get the webpage
	body, err := w.backend.Do(task.URL)
	if err != nil {
		return crawler.TaskResult{Task: *task, Children: nil, Error: &err}, err
	}

	// scrape the webpage
	children := w.scraper(task.URL, body)
	result := crawler.TaskResult{Task: *task, Children: children}

	// executing post-processors
	for _, f := range w.postProcessors {
		if err := f(&result); err != nil {
			result.Error = &err
			return result, err
		}
	}

	w.logger.Debug().Str("url", task.URL.String()).Msg("task processed")
	return result, nil
}
