package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Gaurav-Gosain/scraped/output"
	"github.com/Gaurav-Gosain/scraped/scraper"
	"github.com/Gaurav-Gosain/scraped/tui"
	"github.com/spf13/cobra"
)

type config struct {
	OutputDir    string
	Depth        int
	Parallelism  int
	WordWrap     int
	MaxPages     int
	CrossDomains bool
}

func NewRootCmd() *cobra.Command {
	cfg := &config{}

	cmd := &cobra.Command{
		Use:   "scraped [urls...]",
		Short: "Scrape web pages and convert them to markdown",
		Long:  "A fast, parallelized CLI tool that scrapes web pages and converts them to beautiful markdown.\nSupports native markdown detection (Accept: text/markdown) with HTML-to-markdown fallback.",
		Example: `  # Scrape a single URL
  scraped https://example.com

  # Scrape multiple URLs, save to files
  scraped -o ./docs https://example.com https://go.dev

  # Pipe URLs from a file
  cat urls.txt | scraped

  # Crawl with depth
  scraped -d 2 -p 20 https://example.com`,
		RunE: func(c *cobra.Command, args []string) error {
			return run(c.Context(), cfg, args)
		},
		// Allow positional args (URLs) even though fang adds subcommands.
		TraverseChildren: true,
	}

	cmd.Flags().StringVarP(&cfg.OutputDir, "output-dir", "o", "", "Save .md files to directory (default: render to terminal)")
	cmd.Flags().IntVarP(&cfg.Depth, "depth", "d", 0, "Crawl depth (0 = only given URLs)")
	cmd.Flags().IntVarP(&cfg.Parallelism, "parallelism", "p", 10, "Number of parallel requests")
	cmd.Flags().IntVarP(&cfg.WordWrap, "word-wrap", "w", 80, "Word wrap width for terminal rendering")
	cmd.Flags().IntVarP(&cfg.MaxPages, "max-pages", "m", 0, "Max pages to scrape (0 = unlimited)")
	cmd.Flags().BoolVar(&cfg.CrossDomains, "cross-domains", false, "Allow crawling across different domains")

	return cmd
}

func run(ctx context.Context, cfg *config, args []string) error {
	urls := collectURLs(args)
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided; pass them as arguments or pipe via stdin")
	}

	opts := scraper.Options{
		URLs:         urls,
		Depth:        cfg.Depth,
		Parallelism:  cfg.Parallelism,
		MaxPages:     cfg.MaxPages,
		CrossDomains: cfg.CrossDomains,
	}

	results, err := tui.RunWithProgress(ctx, opts)
	if err != nil {
		return fmt.Errorf("scraping failed: %w", err)
	}

	if cfg.OutputDir != "" {
		return output.WriteFiles(results, cfg.OutputDir)
	}

	// Count successful results for browser decision.
	successCount := 0
	for _, r := range results {
		if r.Err == nil {
			successCount++
		}
	}

	// Launch interactive browser for multi-result TTY output.
	if successCount > 1 && tui.IsTTY() {
		return tui.RunBrowser(results)
	}

	return output.RenderTerminal(results, cfg.WordWrap)
}

func collectURLs(args []string) []string {
	urls := make([]string, 0, len(args))
	urls = append(urls, args...)

	// Read from stdin if piped (not a terminal).
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				urls = append(urls, line)
			}
		}
	}

	return urls
}
