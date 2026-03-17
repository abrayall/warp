# Warp

Warp takes an existing website URL, scrapes it (including JS-rendered sites), uses Claude Code to modernize it with fresh design, and deploys via the lightspeed platform.

## Architecture

- `core/lib/` — Business logic (scraper, modernizer, publisher, UI)
- `framework/cli/` — Cobra CLI entry point and commands
- `framework/server/` — Server stub (future)

## Build & Run

```bash
go build -o warp ./framework/cli
./warp --help
./warp <url>
```

## Pipeline

1. **Scrape** — chromedp loads page, waits for JS, captures rendered HTML + assets
2. **Modernize** — Claude CLI rewrites as modern lightspeed PHP site
3. **Deploy** — `lightspeed deploy` in generated site directory
