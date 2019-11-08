package crawler

import "net/url"

// Task to be processed
type Task struct {
	URL   *url.URL
	Depth int
	Tries int
}

// TaskResult result of processing a task
type TaskResult struct {
	Task
	Children map[string]int
	Error    *error
}
