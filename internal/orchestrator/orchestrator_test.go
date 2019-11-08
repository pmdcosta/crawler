package orchestrator_test

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/pmdcosta/crawler/internal/crawler"
	"github.com/pmdcosta/crawler/internal/orchestrator"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func newTestOrchestrator(t *testing.T, opts ...orchestrator.Option) *orchestrator.Orchestrator {
	logger := zerolog.Nop()
	return orchestrator.New(&logger, 1, opts...)
}

func TestOrchestrator_ok(t *testing.T) {
	host, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: host, Depth: 0, Tries: 1}

	// start orchestrator
	o := newTestOrchestrator(t)
	require.Nil(t, o.Start(host.String()))
	defer o.Stop()

	// mock worker loop 1
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host, Depth: 0, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task, Children: map[string]int{"http://google.com/1": 1}}

	// mock worker loop 2
	host1, _ := url.Parse("http://google.com/1")
	task1 := crawler.Task{URL: host1, Depth: 1, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task1, Children: map[string]int{"http://google.com": 1}}

	<-o.Done()
	expected := map[string]crawler.TaskResult{
		"http://google.com/1": {
			Task: crawler.Task{
				URL:   host1,
				Depth: 1,
				Tries: 1,
			},
			Children: map[string]int{
				"http://google.com": 1,
			},
		},
		"http://google.com": {
			Task: crawler.Task{
				URL:   host,
				Depth: 0,
				Tries: 1,
			},
			Children: map[string]int{
				"http://google.com/1": 1,
			},
		},
	}
	require.Equal(t, expected, o.Processed)
}

func TestOrchestrator_exactFilter(t *testing.T) {
	t.Run("exact filter ok", testOrchestrator_exactFilter_ok)
	t.Run("exact filter pass", testOrchestrator_exactFilter_pass)
}

func testOrchestrator_exactFilter_ok(t *testing.T) {
	host, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: host, Depth: 0, Tries: 1}

	// start orchestrator
	o := newTestOrchestrator(t, orchestrator.AddExactHostFilter("google.com"))
	require.Nil(t, o.Start(host.String()))
	defer o.Stop()

	// mock worker loop 1
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host, Depth: 0, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task, Children: map[string]int{"http://google.com/1": 1}}

	// mock worker loop 2
	host1, _ := url.Parse("http://google.com/1")
	task1 := crawler.Task{URL: host1, Depth: 1, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task1, Children: map[string]int{"http://docs.google.com": 1}}

	<-o.Done()
}

func testOrchestrator_exactFilter_pass(t *testing.T) {
	host, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: host, Depth: 0, Tries: 1}

	// start orchestrator
	o := newTestOrchestrator(t, orchestrator.AddExactHostFilter("google.com"), orchestrator.AddExactHostFilter("docs.google.com"))
	require.Nil(t, o.Start(host.String()))
	defer o.Stop()

	// mock worker loop 1
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host, Depth: 0, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task, Children: map[string]int{"http://google.com/1": 1}}

	// mock worker loop 2
	host1, _ := url.Parse("http://google.com/1")
	task1 := crawler.Task{URL: host1, Depth: 1, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task1, Children: map[string]int{"http://docs.google.com": 1}}

	// mock worker loop 3
	host2, _ := url.Parse("http://docs.google.com")
	task2 := crawler.Task{URL: host2, Depth: 2, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host2, Depth: 2, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task2, Children: map[string]int{"http://fail.google.com/1": 1}}
	<-o.Done()
}

func TestOrchestrator_subFilter(t *testing.T) {
	host, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: host, Depth: 0, Tries: 1}

	// start orchestrator
	o := newTestOrchestrator(t, orchestrator.AddSudDomainFilters("google.com"))
	require.Nil(t, o.Start(host.String()))
	defer o.Stop()

	// mock worker loop 1
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host, Depth: 0, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task, Children: map[string]int{"http://google.com/1": 1}}

	// mock worker loop 2
	host1, _ := url.Parse("http://google.com/1")
	task1 := crawler.Task{URL: host1, Depth: 1, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task1, Children: map[string]int{"http://docs.google.com": 1}}

	// mock worker loop 3
	host2, _ := url.Parse("http://docs.google.com")
	task2 := crawler.Task{URL: host2, Depth: 2, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host2, Depth: 2, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task2, Children: map[string]int{"http://google.fail.com/1": 1}}
	<-o.Done()
}

func TestOrchestrator_depth(t *testing.T) {
	host, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: host, Depth: 0, Tries: 1}

	// start orchestrator
	o := newTestOrchestrator(t, orchestrator.SetMaxDepth(1))
	require.Nil(t, o.Start(host.String()))
	defer o.Stop()

	// mock worker loop 1
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host, Depth: 0, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task, Children: map[string]int{"http://google.com/1": 1}}

	// mock worker loop 2
	host1, _ := url.Parse("http://google.com/1")
	task1 := crawler.Task{URL: host1, Depth: 1, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task1, Children: map[string]int{"http://docs.google.com": 1}}

	<-o.Done()
}

func TestOrchestrator_failed(t *testing.T) {
	host, _ := url.Parse("http://google.com")
	task := crawler.Task{URL: host, Depth: 0, Tries: 1}
	err := errors.New("failed")

	// start orchestrator
	o := newTestOrchestrator(t, orchestrator.SetMaxRetries(1))
	require.Nil(t, o.Start(host.String()))
	defer o.Stop()

	// mock worker loop 1
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host, Depth: 0, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.DoneQueue <- crawler.TaskResult{Task: task, Children: map[string]int{"http://google.com/1": 1}}

	// mock worker loop 2
	host1, _ := url.Parse("http://google.com/1")
	task1 := crawler.Task{URL: host1, Depth: 1, Tries: 1}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 0}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.ErrorQueue <- crawler.TaskResult{Task: task1, Children: nil, Error: &err}

	// mock worker loop 3
	task2 := crawler.Task{URL: host1, Depth: 1, Tries: 2}
	select {
	case r := <-o.TaskQueue:
		require.Equal(t, crawler.Task{URL: host1, Depth: 1, Tries: 1}, r)
	case <-time.After(1 * time.Second):
		require.FailNow(t, "task not received")
	}
	o.ErrorQueue <- crawler.TaskResult{Task: task2, Children: nil, Error: &err}

	<-o.Done()
	expected := map[string]crawler.TaskResult{
		"http://google.com": {
			Task: crawler.Task{
				URL:   host,
				Depth: 0,
				Tries: 1,
			},
			Children: map[string]int{
				"http://google.com/1": 1,
			},
		},
	}
	require.Equal(t, expected, o.Processed)
	expectedF := map[string]crawler.TaskResult{
		"http://google.com/1": {
			Task: crawler.Task{
				URL:   host1,
				Depth: 1,
				Tries: 2,
			},
			Error: &err,
		},
	}
	require.Equal(t, expectedF, o.Failed)
}
