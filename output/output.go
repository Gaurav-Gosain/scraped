package output

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gaurav-Gosain/scraped/scraper"
	"charm.land/glamour/v2"
)

// RenderTerminal renders all results to stdout using glamour.
func RenderTerminal(results []scraper.Result, wordWrap int) error {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "Error scraping %s: %v\n", r.URL, r.Err)
			continue
		}

		header := fmt.Sprintf("# %s\n\n", r.URL)
		rendered, err := renderer.Render(header + r.Markdown)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering %s: %v\n", r.URL, r.Err)
			continue
		}
		fmt.Print(rendered)
	}
	return nil
}

// WriteFiles writes each result as a .md file in the given directory.
func WriteFiles(results []scraper.Result, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "Error scraping %s: %v\n", r.URL, r.Err)
			continue
		}

		filename := urlToFilename(r.URL)
		path := filepath.Join(dir, filename)

		if err := os.WriteFile(path, []byte(r.Markdown), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", path, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "Saved: %s\n", path)
	}
	return nil
}

// urlToFilename converts a URL to a safe filename.
func urlToFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unknown.md"
	}

	name := u.Host + u.Path
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.Trim(name, "-")

	if name == "" {
		name = "index"
	}
	return name + ".md"
}
