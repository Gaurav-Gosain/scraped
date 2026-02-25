<div align="center">
  <h1>scraped</h1>
  <p>A fast, parallelized CLI tool that scrapes web pages and converts them to markdown with an interactive TUI browser.</p>

  <a href="https://github.com/Gaurav-Gosain/scraped/releases"><img src="https://img.shields.io/github/release/Gaurav-Gosain/scraped.svg" alt="Latest Release"></a>
  <a href="https://pkg.go.dev/github.com/Gaurav-Gosain/scraped?tab=doc"><img src="https://godoc.org/github.com/Gaurav-Gosain/scraped?status.svg" alt="GoDoc"></a>
  <a href="https://deepwiki.com/Gaurav-Gosain/scraped"><img src="https://deepwiki.com/badge.svg" alt="Ask DeepWiki"></a>
</div>

---

scraped takes one or more URLs, fetches them in parallel, and converts each page to clean markdown. It first tries requesting native markdown from the server (`Accept: text/markdown`), and falls back to converting HTML when that is not available.

When scraping multiple pages, the results are presented in an interactive TUI browser where you can browse through pages, filter URLs, and search within content. Single-page results are rendered directly to the terminal.

<details>
<summary>Table of Contents</summary>

- [Installation](#installation)
- [Usage](#usage)
- [Features](#features)
- [Interactive Browser](#interactive-browser)
- [Development](#development)
- [License](#license)

</details>

## Installation

### Package Managers

**Homebrew (macOS/Linux):**
```bash
brew tap Gaurav-Gosain/tap
brew install scraped
```

**Arch Linux (AUR):**
```bash
yay -S scraped-bin
```

### Quick Install Script

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/Gaurav-Gosain/scraped/main/install.sh | bash
```

### Other Methods

- **[GitHub Releases](https://github.com/Gaurav-Gosain/scraped/releases)** - Download pre-built binaries
- **Go Install:** `go install github.com/Gaurav-Gosain/scraped@latest`
- **Build from Source:** See [Development](#development) below

**Requirements:**
- A terminal with true color support (most modern terminals work fine)
- Go 1.25+ (if building from source)

## Usage

```bash
# Scrape a single URL
scraped https://example.com

# Scrape multiple URLs and save to files
scraped -o ./docs https://example.com https://go.dev

# Crawl with depth, limiting to 20 pages
scraped -d 2 -m 20 https://example.com

# Pipe URLs from a file
cat urls.txt | scraped

# Allow crawling across different domains
scraped --cross-domains -d 1 https://example.com
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output-dir` | `-o` | | Save .md files to a directory instead of rendering to terminal |
| `--depth` | `-d` | `0` | Crawl depth (0 = only given URLs) |
| `--parallelism` | `-p` | `10` | Number of parallel requests |
| `--word-wrap` | `-w` | `80` | Word wrap width for terminal rendering |
| `--max-pages` | `-m` | `0` | Max pages to scrape (0 = unlimited) |
| `--cross-domains` | | `false` | Allow crawling across different domains |

## Features

- **Parallel scraping** with configurable concurrency
- **Native markdown detection** via `Accept: text/markdown` header, with automatic HTML-to-markdown fallback
- **Recursive crawling** with configurable depth and page limits
- **Interactive TUI browser** for exploring multi-page results
- **Progress display** with real-time scraping status and smooth animations
- **File output** for saving results as individual .md files
- **Pipe-friendly** input from stdin for batch processing
- **Cross-domain crawling** when explicitly enabled

## Interactive Browser

When scraping multiple pages to the terminal, scraped launches an interactive TUI browser with two views:

- **List view** shows all scraped URLs with status indicators (native markdown, converted, or error). Press `/` to fuzzy-filter by URL.
- **Pager view** opens when you select a URL, showing the full page content rendered with glamour (Tokyo Night theme). Press `/` to search within content, and `n`/`N` to jump between matches.

Navigate between the two views with `enter`/`l` to open a page and `esc`/`h` to go back.

### Keyboard Shortcuts

**List view:**

| Key | Action |
|-----|--------|
| `j` / Down | Move cursor down |
| `k` / Up | Move cursor up |
| `enter` / `l` | Open selected page in pager |
| `/` | Filter URLs |
| `g` / `G` | Go to top / bottom |
| `q` / `ctrl+c` | Quit |

**Pager view:**

| Key | Action |
|-----|--------|
| `esc` / `h` | Go back to list |
| `/` | Search within content |
| `n` / `N` | Next / previous search match |
| `g` / `G` | Go to top / bottom |
| `pgup` / `pgdn` | Page up / down |
| `q` / `ctrl+c` | Quit |

## Development

Contributions are welcome. Feel free to open issues or pull requests.

**Build from source:**
```bash
git clone https://github.com/Gaurav-Gosain/scraped.git
cd scraped
go build -o scraped .
./scraped --help
```

**Run tests:**
```bash
go test ./...
```

**Support:** [![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/B0B81N8V1R)

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Gaurav-Gosain/scraped&type=Date&theme=dark)](https://star-history.com/#Gaurav-Gosain/scraped&Date)

<p style="display:flex;flex-wrap:wrap;">
<img alt="GitHub Language Count" src="https://img.shields.io/github/languages/count/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Top Language" src="https://img.shields.io/github/languages/top/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="Repo Size" src="https://img.shields.io/github/repo-size/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Issues" src="https://img.shields.io/github/issues/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Closed Issues" src="https://img.shields.io/github/issues-closed/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Pull Requests" src="https://img.shields.io/github/issues-pr/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Closed Pull Requests" src="https://img.shields.io/github/issues-pr-closed/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Contributors" src="https://img.shields.io/github/contributors/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
<img alt="GitHub Last Commit" src="https://img.shields.io/github/last-commit/Gaurav-Gosain/scraped" style="padding:5px;margin:5px;" />
</p>

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
