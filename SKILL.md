---
name: scraping-webpages-with-scraped
description: Scrapes one or more web pages into markdown with the scraped CLI, including recursive crawling, raw output, and file export. Use when converting web pages to markdown locally, crawling documentation sites, or when the user mentions scraped, markdown scraping, or recursive crawling.
compatibility: Requires the scraped CLI
license: MIT
---

Follow this workflow

- [ ] Choose output mode: raw markdown, or files
- [ ] Provide seed URLs as arguments or on stdin
- [ ] If crawling, set depth, page cap, and cross-domain behavior
- [ ] Check stderr for per-URL failures and confirm whether pages were native or converted

## Common command patterns

Scrape and immediately read one page

```bash
scraped --raw https://example.com
```

Scrape multiple pages and read them all directly

```bash
scraped --raw https://example.com https://go.dev
```

Pipe URLs on stdin

```bash
printf '%s\n' https://example.com https://go.dev | scraped --raw
```

Crawl a site with limits and save to files without immediately reading

```bash
scraped --output-dir ./docs --depth 1 --max-pages 20 --parallelism 5 \
  https://example.com/docs
```

Save to a dir and search around in the files

```bash
scraped --output-dir ./docs https://example.com https://go.dev
grep -R "search term" ./docs
```

## Flags and subcommands

- `-o`, `--output-dir`: write `.md` files to a directory
- `-d`, `--depth`: recursive crawl depth
- `-p`, `--parallelism`: concurrent requests, default `10`
- `-m`, `--max-pages`: cap the crawl, default `0` for unlimited
- `--cross-domains`: allow crawling across domains
- `-r`, `--raw`: disable TUI and ANSI formatting
- `-h`, `--help`: show help for the current command

## Crawl behavior

- `--depth 0` means only the seed URLs are fetched.
- `--depth 1` includes links found on the seed pages, and larger values continue recursively.
- Recursive crawls stay on the seed domains by default.
- `--cross-domains` allows the crawler to follow links onto other domains.
- `--max-pages` limits the number of requests started; `0` means unlimited.
- `--parallelism` controls concurrent requests.
- Native markdown pages can still be crawled because scraped extracts links from the markdown AST.

## Output behavior

- If `--output-dir` is set, scraped writes one `.md` file per successful page and does not launch the TUI browser.
- If `--raw` is set, scraped writes plain markdown with YAML frontmatter:

```yaml
---
url: https://example.com
source: native
---
```

The interactive browser only opens when there are multiple successful results and a TTY is available. Avoid this unless you're using tmux. Then probably still avoid this unless the user says to browse them with tmux.

## Good defaults

- Use `--raw` when scraping a few small pages or one larger page. If many pages or a few larger pages, prefer to save them somewhere so you don't have to keep scraping them.
- Use `--output-dir $(mktemp -d)` any time you're scraping multiple pages so you can refer to them later without causing additional load. If the user wants the docs kept somewhere else, specify a non-temporary directory.
- Use the default domain restriction unless the user explicitly wants off-site crawling.
- Start with shallow crawls and a small `--max-pages` value on unfamiliar sites. Prefer to find an index page and scrape just the URLs you need.
- If you find an index, read it to get a good overview of what's where so you can navigate more precisely and read less irrelevant content.

## Pitfalls

- A single invalid seed URL aborts URL collection.
- Non-HTML and non-markdown assets do not become results.
- If scraped is unavailable, direct to the user to install it from <https://github.com/Gaurav-Gosain/scraped>
