package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

// mdParser is reused across calls for efficiency.
var mdParser = goldmark.New()

// extractMarkdownLinks parses markdown with goldmark and extracts all link
// destinations (inline links, reference links, autolinks).
func extractMarkdownLinks(md string, baseURL string) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	source := []byte(md)
	doc := mdParser.Parser().Parse(text.NewReader(source))

	var links []string
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		var dest []byte
		switch node := n.(type) {
		case *ast.Link:
			dest = node.Destination
		case *ast.AutoLink:
			dest = node.URL(source)
		}

		if len(dest) == 0 {
			return ast.WalkContinue, nil
		}

		href := strings.TrimSpace(string(dest))
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") {
			return ast.WalkContinue, nil
		}

		u, err := url.Parse(href)
		if err != nil {
			return ast.WalkContinue, nil
		}

		resolved := base.ResolveReference(u)
		resolved.Fragment = ""
		if resolved.Scheme == "http" || resolved.Scheme == "https" {
			links = append(links, resolved.String())
		}

		return ast.WalkContinue, nil
	})

	return links
}

// Event is emitted during scraping for progress tracking.
type Event struct {
	Type   string // "fetching", "done", "error"
	URL    string
	Source string // "native" or "converted" (only for "done" events)
	Err    error  // only for "error" events
}

// Options configures the scraper engine.
type Options struct {
	URLs         []string
	Depth        int
	Parallelism  int
	MaxPages     int         // 0 = unlimited
	CrossDomains bool        // allow crawling across different domains
	OnEvent      func(Event) // optional progress callback
}

func (o *Options) emit(e Event) {
	if o.OnEvent != nil {
		o.OnEvent(e)
	}
}

// extractDomains returns the unique hostnames from a list of URLs.
func extractDomains(urls []string) []string {
	seen := make(map[string]bool)
	var domains []string
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			continue
		}
		host := u.Hostname()
		if !seen[host] {
			seen[host] = true
			domains = append(domains, host)
		}
		// Also allow www. variant and vice versa.
		if after, ok := strings.CutPrefix(host, "www."); ok {
			bare := after
			if !seen[bare] {
				seen[bare] = true
				domains = append(domains, bare)
			}
		} else {
			www := "www." + host
			if !seen[www] {
				seen[www] = true
				domains = append(domains, www)
			}
		}
	}
	return domains
}

func cleanLink(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	return u.String()
}

// Run scrapes all seed URLs and returns the collected results.
func Run(ctx context.Context, opts Options) ([]Result, error) {
	store := NewResultStore()

	// Colly depth model: c.Visit() starts at depth 1, children are depth 2, etc.
	// MaxDepth(N) rejects depth > N. So --depth 0 (seeds only) = MaxDepth(1),
	// --depth 1 (seeds + 1 level) = MaxDepth(2), etc.
	collectorOpts := []colly.CollectorOption{
		colly.MaxDepth(opts.Depth + 1),
		colly.Async(),
	}

	// When crawling (depth > 0), restrict to seed URL domains unless --cross-domains.
	if opts.Depth > 0 && !opts.CrossDomains {
		domains := extractDomains(opts.URLs)
		if len(domains) > 0 {
			collectorOpts = append(collectorOpts, colly.AllowedDomains(domains...))
		}
	}

	c := colly.NewCollector(collectorOpts...)

	c.SetRequestTimeout(15 * time.Second)

	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: opts.Parallelism,
	})

	var started atomic.Int64

	c.OnRequest(func(r *colly.Request) {
		if ctx.Err() != nil {
			r.Abort()
			return
		}
		if opts.MaxPages > 0 && int(started.Load()) >= opts.MaxPages {
			r.Abort()
			return
		}
		started.Add(1)
		r.Headers.Set("Accept", "text/markdown")
		opts.emit(Event{Type: "fetching", URL: r.URL.String()})
	})

	c.OnResponse(func(r *colly.Response) {
		ct := r.Headers.Get("Content-Type")
		reqURL := r.Request.URL.String()

		switch {
		case strings.Contains(ct, "text/markdown"):
			body := string(r.Body)
			store.Add(Result{
				URL:      reqURL,
				Markdown: body,
				Source:   "native",
			})
			opts.emit(Event{Type: "done", URL: reqURL, Source: "native"})
			// Native markdown has no HTML DOM for colly to parse.
			// Extract links from the markdown AST and queue them.
			if opts.Depth > 0 {
				for _, link := range extractMarkdownLinks(body, reqURL) {
					_ = r.Request.Visit(link)
				}
			}

		case strings.Contains(ct, "text/html"):
			md, err := htmltomarkdown.ConvertString(
				string(r.Body),
				converter.WithDomain(reqURL),
			)
			if err != nil {
				store.Add(Result{
					URL: reqURL,
					Err: fmt.Errorf("markdown conversion failed: %w", err),
				})
				opts.emit(Event{Type: "error", URL: reqURL, Err: err})
				return
			}
			store.Add(Result{
				URL:      reqURL,
				Markdown: md,
				Source:   "converted",
			})
			opts.emit(Event{Type: "done", URL: reqURL, Source: "converted"})

		default:
			// Non-HTML/markdown (CSS, JS, images, etc.) — emit so TUI can clean up.
			opts.emit(Event{Type: "done", URL: reqURL, Source: "skipped"})
		}
	})

	if opts.Depth > 0 {
		c.OnHTML("a[href]", func(e *colly.HTMLElement) {
			link := e.Request.AbsoluteURL(e.Attr("href"))
			if link == "" {
				return
			}
			_ = e.Request.Visit(cleanLink(link))
		})
	}

	c.OnError(func(r *colly.Response, err error) {
		// Silently ignore aborted requests (context cancellation or max-pages).
		if ctx.Err() != nil || r.StatusCode == 0 {
			return
		}
		reqURL := r.Request.URL.String()
		store.Add(Result{
			URL: reqURL,
			Err: fmt.Errorf("request failed (status %d): %w", r.StatusCode, err),
		})
		opts.emit(Event{Type: "error", URL: reqURL, Err: err})
	})

	for _, u := range opts.URLs {
		_ = c.Visit(u)
	}

	c.Wait()

	return store.Results(), nil
}
