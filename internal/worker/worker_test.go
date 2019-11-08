package worker_test

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pmdcosta/crawler/internal/crawler"
	"github.com/pmdcosta/crawler/internal/worker"
	"github.com/pmdcosta/crawler/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// Worker is a test wrapper for worker
type Worker struct {
	worker.Worker

	tasks   chan crawler.Task
	done    chan crawler.TaskResult
	errors  chan crawler.TaskResult
	backend *mocks.MockWorkerBackend
}

func newTestWorker(t *testing.T, scraper worker.Scraper, opts ...worker.Option) *Worker {
	logger := zerolog.Nop()
	mockCtrl := gomock.NewController(t)
	backend := mocks.NewMockWorkerBackend(mockCtrl)
	tasks := make(chan crawler.Task)
	done := make(chan crawler.TaskResult)
	errors := make(chan crawler.TaskResult)

	w := worker.New(&logger, tasks, done, errors, backend, scraper, opts...)
	return &Worker{
		Worker:  *w,
		tasks:   tasks,
		done:    done,
		errors:  errors,
		backend: backend,
	}
}

func TestWorker_ok(t *testing.T) {
	root, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: root, Tries: 1}
	body := []byte("body")
	children := map[string]int{"google.com/1": 1}
	result := crawler.TaskResult{Task: task, Children: children}

	// mock scraper
	var scraperCall bool
	scraper := func(arg *url.URL, page []byte) map[string]int {
		scraperCall = true
		require.Equal(t, root, arg)
		require.Equal(t, body, page)
		return children
	}

	// start worker
	w := newTestWorker(t, scraper)
	require.Nil(t, w.Start())
	defer w.Stop()

	// mock backend
	w.backend.EXPECT().Do(root).Times(1).Return(body, nil)

	// send the task to the worker
	w.tasks <- crawler.Task{URL: root}

	// assert response
	select {
	case r := <-w.done:
		require.Equal(t, result, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "result not received")
	}
	require.True(t, scraperCall)
}

func TestWorker_process(t *testing.T) {
	root, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: root, Tries: 1}
	body := []byte("body")
	children := map[string]int{"google.com/1": 1}
	result := crawler.TaskResult{Task: task, Children: children}

	// mock scraper
	var scraperCall bool
	scraper := func(arg *url.URL, page []byte) map[string]int {
		scraperCall = true
		require.Equal(t, root, arg)
		require.Equal(t, body, page)
		return children
	}

	// add pre and post processors
	var preCall, postCall bool
	preProcess := func(arg *crawler.Task) (ignore bool, err error) {
		preCall = true
		require.Equal(t, &task, arg)
		return false, nil
	}
	postProcess := func(arg *crawler.TaskResult) error {
		postCall = true
		require.Equal(t, &result, arg)
		return nil
	}

	// start worker
	w := newTestWorker(t, scraper, worker.AddPreProcessor(preProcess), worker.AddPostProcessor(postProcess))
	require.Nil(t, w.Start())
	defer w.Stop()

	// mock backend
	w.backend.EXPECT().Do(root).Times(1).Return(body, nil)

	// send the task to the worker
	w.tasks <- crawler.Task{URL: root}

	// assert response
	select {
	case r := <-w.done:
		require.Equal(t, result, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "result not received")
	}
	require.True(t, scraperCall)
	require.True(t, preCall)
	require.True(t, postCall)
}

func TestWorker_ignore(t *testing.T) {
	root, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: root, Tries: 1}
	result := crawler.TaskResult{Task: task, Children: nil}

	// mock scraper
	var scraperCall bool
	scraper := func(arg *url.URL, page []byte) map[string]int {
		scraperCall = true
		return nil
	}

	// add pre and post processors
	var preCall bool
	preProcess := func(arg *crawler.Task) (ignore bool, err error) {
		preCall = true
		require.Equal(t, &task, arg)
		return true, nil
	}

	// start worker
	w := newTestWorker(t, scraper, worker.AddPreProcessor(preProcess))
	require.Nil(t, w.Start())
	defer w.Stop()

	// send the task to the worker
	w.tasks <- crawler.Task{URL: root}

	// assert response
	select {
	case r := <-w.done:
		require.Equal(t, result, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "result not received")
	}
	require.False(t, scraperCall)
	require.True(t, preCall)
}

func TestWorker_error(t *testing.T) {
	root, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: root, Tries: 1}
	err := errors.New("failed to process")
	result := crawler.TaskResult{Task: task, Children: nil, Error: &err}

	// mock scraper
	var scraperCall bool
	scraper := func(arg *url.URL, page []byte) map[string]int {
		scraperCall = true
		return nil
	}

	// start worker
	w := newTestWorker(t, scraper)
	require.Nil(t, w.Start())
	defer w.Stop()

	// mock backend
	w.backend.EXPECT().Do(root).Times(1).Return(nil, err)

	// send the task to the worker
	w.tasks <- crawler.Task{URL: root}

	// assert response
	select {
	case r := <-w.errors:
		require.Equal(t, result, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "result not received")
	}
	require.False(t, scraperCall)
}
