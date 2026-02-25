package scraper

import "sync"

// Result represents a single scraped page.
type Result struct {
	URL      string
	Markdown string
	Source   string // "native" or "converted"
	Err      error
}

// ResultStore is a thread-safe ordered collection of results.
type ResultStore struct {
	mu      sync.Mutex
	results []Result
	seen    map[string]bool
}

func NewResultStore() *ResultStore {
	return &ResultStore{
		seen: make(map[string]bool),
	}
}

func (rs *ResultStore) Add(r Result) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.seen[r.URL] {
		return
	}
	rs.seen[r.URL] = true
	rs.results = append(rs.results, r)
}

func (rs *ResultStore) Count() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return len(rs.results)
}

func (rs *ResultStore) Results() []Result {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	out := make([]Result, len(rs.results))
	copy(out, rs.results)
	return out
}
